package coordinator

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/v3dang/vex/internal/ring"
	"github.com/v3dang/vex/internal/store"
	pb "github.com/v3dang/vex/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	ErrQuorumNotReached = errors.New("write quorum not reached")
	ErrNotFound         = errors.New("vector not found")
)

// Config configures the distributed coordinator.
type Config struct {
	NodeID      string
	Replicas    int
	WriteQuorum int
	Timeout     time.Duration
}

// Coordinator orchestrates distributed vector operations across cluster shards.
type Coordinator struct {
	cfg        Config
	localStore *store.Store
	ring       *ring.Ring

	clientsMu sync.RWMutex
	clients   map[string]pb.VectorStoreClient
	conns     map[string]*grpc.ClientConn
}

// New creates a new distributed Coordinator.
func New(cfg Config, s *store.Store, r *ring.Ring) *Coordinator {
	if cfg.Replicas <= 0 {
		cfg.Replicas = 2
	}
	if cfg.WriteQuorum <= 0 {
		cfg.WriteQuorum = 1
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}

	return &Coordinator{
		cfg:        cfg,
		localStore: s,
		ring:       r,
		clients:    make(map[string]pb.VectorStoreClient),
		conns:      make(map[string]*grpc.ClientConn),
	}
}

// getClient returns or lazily creates a gRPC client to a target peer node.
func (c *Coordinator) getClient(node ring.Node) (pb.VectorStoreClient, error) {
	c.clientsMu.RLock()
	client, ok := c.clients[node.ID]
	c.clientsMu.RUnlock()
	if ok {
		return client, nil
	}

	c.clientsMu.Lock()
	defer c.clientsMu.Unlock()

	// Double check
	if client, ok := c.clients[node.ID]; ok {
		return client, nil
	}

	conn, err := grpc.NewClient(node.GrpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial node %s (%s): %w", node.ID, node.GrpcAddr, err)
	}

	client = pb.NewVectorStoreClient(conn)
	c.conns[node.ID] = conn
	c.clients[node.ID] = client
	return client, nil
}

// Close closes all cached peer gRPC connections.
func (c *Coordinator) Close() error {
	c.clientsMu.Lock()
	defer c.clientsMu.Unlock()

	for _, conn := range c.conns {
		_ = conn.Close()
	}
	c.clients = make(map[string]pb.VectorStoreClient)
	c.conns = make(map[string]*grpc.ClientConn)
	return nil
}

// CoordinatedInsert hashes the ID and replicates the insert to primary and replica nodes.
func (c *Coordinator) CoordinatedInsert(ctx context.Context, req *pb.InsertRequest) error {
	targets, err := c.ring.GetNodes(req.Id, c.cfg.Replicas)
	if err != nil {
		return fmt.Errorf("get target nodes: %w", err)
	}
	if len(targets) == 0 {
		return ring.ErrEmptyRing
	}

	requiredQuorum := c.cfg.WriteQuorum
	if requiredQuorum > len(targets) {
		requiredQuorum = len(targets)
	}

	var successfulAcks int32
	var wg sync.WaitGroup

	ctxTimeout, cancel := context.WithTimeout(ctx, c.cfg.Timeout)
	defer cancel()

	for _, target := range targets {
		wg.Add(1)
		go func(node ring.Node) {
			defer wg.Done()

			if node.ID == c.cfg.NodeID {
				// Local shard write
				if err := c.localStore.Insert(req.Id, req.Vector, req.Payload); err == nil {
					atomic.AddInt32(&successfulAcks, 1)
				}
				return
			}

			// Remote shard write via gRPC
			client, err := c.getClient(node)
			if err != nil {
				return
			}

			resp, err := client.ShardInsert(ctxTimeout, req)
			if err == nil && resp.Success {
				atomic.AddInt32(&successfulAcks, 1)
			}
		}(target)
	}

	wg.Wait()

	if int(atomic.LoadInt32(&successfulAcks)) < requiredQuorum {
		return fmt.Errorf("%w: got %d acks, needed %d", ErrQuorumNotReached, atomic.LoadInt32(&successfulAcks), requiredQuorum)
	}

	return nil
}

// CoordinatedSearch scatters the search query across all active shards in the cluster
// and merges the local top-K results into a global top-K response.
func (c *Coordinator) CoordinatedSearch(ctx context.Context, req *pb.SearchRequest) ([]*pb.SearchResult, error) {
	allNodes := c.ring.GetAllNodes()
	if len(allNodes) == 0 {
		// Fallback to local store if ring has no other nodes yet
		return c.localSearch(req)
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, c.cfg.Timeout)
	defer cancel()

	resultsCh := make(chan []*pb.SearchResult, len(allNodes))
	var wg sync.WaitGroup

	for _, node := range allNodes {
		wg.Add(1)
		go func(target ring.Node) {
			defer wg.Done()

			if target.ID == c.cfg.NodeID {
				res, err := c.localSearch(req)
				if err == nil {
					resultsCh <- res
				}
				return
			}

			client, err := c.getClient(target)
			if err != nil {
				return
			}

			resp, err := client.ShardSearch(ctxTimeout, req)
			if err == nil && resp != nil {
				resultsCh <- resp.Results
			}
		}(node)
	}

	wg.Wait()
	close(resultsCh)

	// Gather and deduplicate candidate results
	seen := make(map[string]bool)
	var allCandidates []*pb.SearchResult

	for shardResults := range resultsCh {
		for _, r := range shardResults {
			if !seen[r.Id] {
				seen[r.Id] = true
				allCandidates = append(allCandidates, r)
			}
		}
	}

	// Sort candidates by distance ascending (nearest first)
	sort.Slice(allCandidates, func(i, j int) bool {
		return allCandidates[i].Distance < allCandidates[j].Distance
	})

	// Take top-K
	k := int(req.K)
	if k <= 0 {
		k = 10
	}
	if len(allCandidates) > k {
		allCandidates = allCandidates[:k]
	}

	return allCandidates, nil
}

// CoordinatedDelete replicates the delete across the key's target nodes.
func (c *Coordinator) CoordinatedDelete(ctx context.Context, req *pb.DeleteRequest) error {
	targets, err := c.ring.GetNodes(req.Id, c.cfg.Replicas)
	if err != nil {
		return fmt.Errorf("get target nodes: %w", err)
	}
	if len(targets) == 0 {
		return ring.ErrEmptyRing
	}

	requiredQuorum := c.cfg.WriteQuorum
	if requiredQuorum > len(targets) {
		requiredQuorum = len(targets)
	}

	var successfulAcks int32
	var wg sync.WaitGroup

	ctxTimeout, cancel := context.WithTimeout(ctx, c.cfg.Timeout)
	defer cancel()

	for _, target := range targets {
		wg.Add(1)
		go func(node ring.Node) {
			defer wg.Done()

			if node.ID == c.cfg.NodeID {
				if err := c.localStore.Delete(req.Id); err == nil {
					atomic.AddInt32(&successfulAcks, 1)
				}
				return
			}

			client, err := c.getClient(node)
			if err != nil {
				return
			}

			resp, err := client.ShardDelete(ctxTimeout, req)
			if err == nil && resp.Success {
				atomic.AddInt32(&successfulAcks, 1)
			}
		}(target)
	}

	wg.Wait()

	if int(atomic.LoadInt32(&successfulAcks)) < requiredQuorum {
		return fmt.Errorf("%w: got %d acks, needed %d", ErrQuorumNotReached, atomic.LoadInt32(&successfulAcks), requiredQuorum)
	}

	return nil
}

// CoordinatedGet retrieves a vector by ID, checking primary and replica nodes.
func (c *Coordinator) CoordinatedGet(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	targets, err := c.ring.GetNodes(req.Id, c.cfg.Replicas)
	if err != nil {
		return nil, err
	}
	if len(targets) == 0 {
		return nil, ring.ErrEmptyRing
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, c.cfg.Timeout)
	defer cancel()

	for _, target := range targets {
		if target.ID == c.cfg.NodeID {
			if node, ok := c.localStore.Get(req.Id); ok {
				return &pb.GetResponse{
					Found:   true,
					Id:      node.ID,
					Vector:  node.Vector,
					Payload: node.Payload,
				}, nil
			}
			continue
		}

		client, err := c.getClient(target)
		if err != nil {
			continue
		}

		resp, err := client.ShardGet(ctxTimeout, req)
		if err == nil && resp != nil && resp.Found {
			return resp, nil
		}
	}

	return &pb.GetResponse{Found: false, Id: req.Id}, nil
}

// localSearch executes an ANN search on the local HNSW store.
func (c *Coordinator) localSearch(req *pb.SearchRequest) ([]*pb.SearchResult, error) {
	k := int(req.K)
	if k <= 0 {
		k = 10
	}
	results, err := c.localStore.Search(req.Vector, k)
	if err != nil {
		return nil, err
	}

	out := make([]*pb.SearchResult, len(results))
	for i, r := range results {
		out[i] = &pb.SearchResult{
			Id:       r.ID,
			Distance: r.Distance,
			Payload:  r.Payload,
		}
	}
	return out, nil
}
