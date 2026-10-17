package client

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	pb "github.com/v3dang/vex/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	ErrNoEndpoints = errors.New("no cluster endpoints provided")
)

// Client is a thread-safe client for interacting with a VEX vector database cluster.
type Client struct {
	endpoints []string
	conns     []*grpc.ClientConn
	clients   []pb.VectorStoreClient
	idx       uint64
	mu        sync.RWMutex
}

// New creates a new VEX client connected to the provided endpoints.
// Client requests will round-robin across available cluster entrypoints.
func New(endpoints []string) (*Client, error) {
	if len(endpoints) == 0 {
		return nil, ErrNoEndpoints
	}

	c := &Client{
		endpoints: endpoints,
		conns:     make([]*grpc.ClientConn, 0, len(endpoints)),
		clients:   make([]pb.VectorStoreClient, 0, len(endpoints)),
	}

	for _, ep := range endpoints {
		conn, err := grpc.NewClient(ep, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			c.Close()
			return nil, fmt.Errorf("connect to %s: %w", ep, err)
		}
		c.conns = append(c.conns, conn)
		c.clients = append(c.clients, pb.NewVectorStoreClient(conn))
	}

	return c, nil
}

func (c *Client) nextClient() pb.VectorStoreClient {
	idx := atomic.AddUint64(&c.idx, 1)
	return c.clients[int(idx)%len(c.clients)]
}

// Insert adds or updates a vector with metadata payload in the cluster.
func (c *Client) Insert(ctx context.Context, id string, vector []float32, payload []byte) error {
	client := c.nextClient()
	resp, err := client.Insert(ctx, &pb.InsertRequest{
		Id:      id,
		Vector:  vector,
		Payload: payload,
	})
	if err != nil {
		return err
	}
	if !resp.Success {
		return errors.New("insert failed")
	}
	return nil
}

// Search queries the cluster for the k nearest neighbors using approximate nearest neighbor search.
func (c *Client) Search(ctx context.Context, vector []float32, k int, ef int) ([]*pb.SearchResult, error) {
	client := c.nextClient()
	resp, err := client.Search(ctx, &pb.SearchRequest{
		Vector: vector,
		K:      int32(k),
		Ef:     int32(ef),
	})
	if err != nil {
		return nil, err
	}
	return resp.Results, nil
}

// Delete removes a vector by ID across its replica set.
func (c *Client) Delete(ctx context.Context, id string) error {
	client := c.nextClient()
	resp, err := client.Delete(ctx, &pb.DeleteRequest{Id: id})
	if err != nil {
		return err
	}
	if !resp.Success {
		return errors.New("delete failed")
	}
	return nil
}

// Get fetches a vector and its payload by ID.
func (c *Client) Get(ctx context.Context, id string) (*pb.GetResponse, error) {
	client := c.nextClient()
	return client.Get(ctx, &pb.GetRequest{Id: id})
}

// Health checks cluster node status.
func (c *Client) Health(ctx context.Context) (*pb.HealthResponse, error) {
	client := c.nextClient()
	return client.Health(ctx, &pb.HealthRequest{})
}

// Close terminates all client gRPC connections.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var firstErr error
	for _, conn := range c.conns {
		if err := conn.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	c.conns = nil
	c.clients = nil
	return firstErr
}
