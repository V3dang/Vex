package server

import (
	"context"

	"github.com/v3dang/vex/internal/coordinator"
	"github.com/v3dang/vex/internal/ring"
	"github.com/v3dang/vex/internal/store"
	pb "github.com/v3dang/vex/pb"
)

type VexServer struct {
	pb.UnimplementedVectorStoreServer
	nodeID      string
	store       *store.Store
	ring        *ring.Ring
	coordinator *coordinator.Coordinator
}

func New(nodeID string, s *store.Store, r *ring.Ring, c *coordinator.Coordinator) *VexServer {
	return &VexServer{
		nodeID:      nodeID,
		store:       s,
		ring:        r,
		coordinator: c,
	}
}

// Client-facing Coordinated RPCs

func (v *VexServer) Insert(ctx context.Context, req *pb.InsertRequest) (*pb.InsertResponse, error) {
	err := v.coordinator.CoordinatedInsert(ctx, req)
	return &pb.InsertResponse{Success: err == nil}, err
}

func (v *VexServer) Search(ctx context.Context, req *pb.SearchRequest) (*pb.SearchResponse, error) {
	results, err := v.coordinator.CoordinatedSearch(ctx, req)
	if err != nil {
		return nil, err
	}
	return &pb.SearchResponse{Results: results}, nil
}

func (v *VexServer) Delete(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	err := v.coordinator.CoordinatedDelete(ctx, req)
	return &pb.DeleteResponse{Success: err == nil}, err
}

func (v *VexServer) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	return v.coordinator.CoordinatedGet(ctx, req)
}

func (v *VexServer) Health(_ context.Context, _ *pb.HealthRequest) (*pb.HealthResponse, error) {
	var clusterCount int32 = 1
	if v.ring != nil {
		clusterCount = int32(v.ring.NodeCount())
	}
	return &pb.HealthResponse{
		Status:       "ok",
		VectorCount:  v.store.Count(),
		NodeId:       v.nodeID,
		ClusterNodes: clusterCount,
	}, nil
}

// Internal Shard-level RPCs

func (v *VexServer) ShardInsert(_ context.Context, req *pb.InsertRequest) (*pb.InsertResponse, error) {
	err := v.store.Insert(req.Id, req.Vector, req.Payload)
	return &pb.InsertResponse{Success: err == nil}, err
}

func (v *VexServer) ShardSearch(_ context.Context, req *pb.SearchRequest) (*pb.SearchResponse, error) {
	results, err := v.store.Search(req.Vector, int(req.K))
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
	return &pb.SearchResponse{Results: out}, nil
}

func (v *VexServer) ShardDelete(_ context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	err := v.store.Delete(req.Id)
	return &pb.DeleteResponse{Success: err == nil}, err
}

func (v *VexServer) ShardGet(_ context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	node, ok := v.store.Get(req.Id)
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
