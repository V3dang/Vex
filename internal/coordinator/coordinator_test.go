package coordinator

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/v3dang/vex/internal/ring"
	"github.com/v3dang/vex/internal/store"
	pb "github.com/v3dang/vex/pb"
	"google.golang.org/grpc"
)

// mockShardServer implements the Shard RPCs for testing
type mockShardServer struct {
	pb.UnimplementedVectorStoreServer
	store *store.Store
}

func (m *mockShardServer) ShardInsert(_ context.Context, req *pb.InsertRequest) (*pb.InsertResponse, error) {
	err := m.store.Insert(req.Id, req.Vector, req.Payload)
	return &pb.InsertResponse{Success: err == nil}, err
}

func (m *mockShardServer) ShardSearch(_ context.Context, req *pb.SearchRequest) (*pb.SearchResponse, error) {
	res, err := m.store.Search(req.Vector, int(req.K))
	if err != nil {
		return nil, err
	}
	out := make([]*pb.SearchResult, len(res))
	for i, r := range res {
		out[i] = &pb.SearchResult{
			Id:       r.ID,
			Distance: r.Distance,
			Payload:  r.Payload,
		}
	}
	return &pb.SearchResponse{Results: out}, nil
}

func (m *mockShardServer) ShardGet(_ context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	node, ok := m.store.Get(req.Id)
	if !ok {
		return &pb.GetResponse{Found: false, Id: req.Id}, nil
	}
	return &pb.GetResponse{
		Found:   true,
		Id:      node.ID,
		Vector:  node.Vector,
		Payload: node.Payload,
	}, nil
}

func startTestShardServer(t *testing.T, s *store.Store) (string, func()) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterVectorStoreServer(grpcServer, &mockShardServer{store: s})

	go func() {
		_ = grpcServer.Serve(lis)
	}()

	addr := lis.Addr().String()
	stop := func() {
		grpcServer.Stop()
		_ = lis.Close()
	}
	return addr, stop
}

func TestScatterGatherSearchAndReplication(t *testing.T) {
	// Setup Node 1
	s1, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store 1: %v", err)
	}
	defer s1.Close()
	addr1, stop1 := startTestShardServer(t, s1)
	defer stop1()

	// Setup Node 2
	s2, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store 2: %v", err)
	}
	defer s2.Close()
	addr2, stop2 := startTestShardServer(t, s2)
	defer stop2()

	// Setup shared ring
	r := ring.New(32)
	r.AddNode(ring.Node{ID: "node-1", GrpcAddr: addr1})
	r.AddNode(ring.Node{ID: "node-2", GrpcAddr: addr2})

	// Coordinator on Node 1 with 2 replicas and quorum 2
	coord1 := New(Config{
		NodeID:      "node-1",
		Replicas:    2,
		WriteQuorum: 2,
		Timeout:     2 * time.Second,
	}, s1, r)
	defer coord1.Close()

	ctx := context.Background()

	// 1. Insert vector via coordinator on Node 1 (replicated to both nodes)
	v1 := []float32{1.0, 0.0, 0.0}
	err = coord1.CoordinatedInsert(ctx, &pb.InsertRequest{
		Id:      "vec-1",
		Vector:  v1,
		Payload: []byte("payload-1"),
	})
	if err != nil {
		t.Fatalf("coordinated insert failed: %v", err)
	}

	// Verify both Node 1 and Node 2 received the replicated vector
	if n, ok := s1.Get("vec-1"); !ok || n.ID != "vec-1" {
		t.Errorf("expected node 1 to hold vec-1")
	}
	if n, ok := s2.Get("vec-1"); !ok || n.ID != "vec-1" {
		t.Errorf("expected node 2 to hold replicated vec-1")
	}

	// 2. Insert distinct vector into Node 2 only (simulating sharding)
	v2 := []float32{0.0, 1.0, 0.0}
	_ = s2.Insert("vec-2-only-on-2", v2, []byte("payload-2"))

	// 3. Scatter-gather search from Node 1
	// Node 1 should fan out across both nodes and find vec-2 from Node 2!
	searchResp, err := coord1.CoordinatedSearch(ctx, &pb.SearchRequest{
		Vector: []float32{0.05, 0.95, 0.0},
		K:      5,
	})
	if err != nil {
		t.Fatalf("scatter-gather search failed: %v", err)
	}

	if len(searchResp) < 2 {
		t.Fatalf("expected at least 2 results gathered from cluster, got %d", len(searchResp))
	}

	// vec-2-only-on-2 should be the closest result
	if searchResp[0].Id != "vec-2-only-on-2" {
		t.Errorf("expected vec-2-only-on-2 to be top match, got %s", searchResp[0].Id)
	}

	// 4. Coordinated Get: Node 1 retrieves vector only residing on Node 2
	getResp, err := coord1.CoordinatedGet(ctx, &pb.GetRequest{Id: "vec-2-only-on-2"})
	if err != nil {
		t.Fatalf("coordinated get failed: %v", err)
	}
	if !getResp.Found || string(getResp.Payload) != "payload-2" {
		t.Errorf("expected payload-2 from coordinated get, got found=%v payload=%s", getResp.Found, string(getResp.Payload))
	}
}

func TestNodeFailureSurvival(t *testing.T) {
	s1, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store 1: %v", err)
	}
	defer s1.Close()
	addr1, stop1 := startTestShardServer(t, s1)
	defer stop1()

	s2, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store 2: %v", err)
	}
	defer s2.Close()
	addr2, stop2 := startTestShardServer(t, s2)

	r := ring.New(32)
	r.AddNode(ring.Node{ID: "node-1", GrpcAddr: addr1})
	r.AddNode(ring.Node{ID: "node-2", GrpcAddr: addr2})

	// Replicas = 2, WriteQuorum = 1 so system survives 1 node down
	coord := New(Config{
		NodeID:      "node-1",
		Replicas:    2,
		WriteQuorum: 1,
		Timeout:     500 * time.Millisecond,
	}, s1, r)
	defer coord.Close()

	ctx := context.Background()

	// Insert when both nodes alive
	_ = coord.CoordinatedInsert(ctx, &pb.InsertRequest{
		Id:     "alive-test",
		Vector: []float32{0.5, 0.5, 0.5},
	})

	// KILL Node 2!
	stop2()
	_ = r.RemoveNode("node-2")

	// Node 1 should still be able to insert with quorum 1
	err = coord.CoordinatedInsert(ctx, &pb.InsertRequest{
		Id:     "survived-write",
		Vector: []float32{0.1, 0.2, 0.3},
	})
	if err != nil {
		t.Fatalf("write failed after node failure: %v", err)
	}

	// Node 1 should still be able to search
	res, err := coord.CoordinatedSearch(ctx, &pb.SearchRequest{
		Vector: []float32{0.1, 0.2, 0.3},
		K:      5,
	})
	if err != nil {
		t.Fatalf("search failed after node failure: %v", err)
	}
	if len(res) == 0 {
		t.Fatalf("expected search results from surviving node, got 0")
	}
}
