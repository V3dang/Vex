package ring

import (
	"fmt"
	"testing"
)

func TestEmptyRing(t *testing.T) {
	r := New(64)
	_, err := r.GetNode("vec-1")
	if err != ErrEmptyRing {
		t.Fatalf("expected ErrEmptyRing, got %v", err)
	}
}

func TestSingleNode(t *testing.T) {
	r := New(64)
	node := Node{ID: "node-1", GrpcAddr: "127.0.0.1:7070"}
	r.AddNode(node)

	got, err := r.GetNode("any-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "node-1" {
		t.Fatalf("expected node-1, got %s", got.ID)
	}

	nodes, err := r.GetNodes("any-key", 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nodes) != 1 || nodes[0].ID != "node-1" {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
}

func TestMultiNodeReplication(t *testing.T) {
	r := New(64)
	r.AddNode(Node{ID: "node-1", GrpcAddr: "127.0.0.1:7071"})
	r.AddNode(Node{ID: "node-2", GrpcAddr: "127.0.0.1:7072"})
	r.AddNode(Node{ID: "node-3", GrpcAddr: "127.0.0.1:7073"})

	if r.NodeCount() != 3 {
		t.Fatalf("expected 3 nodes, got %d", r.NodeCount())
	}

	// Request 2 replicas (primary + 1 replica)
	replicas, err := r.GetNodes("vector-abc", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(replicas) != 2 {
		t.Fatalf("expected 2 replicas, got %d", len(replicas))
	}
	if replicas[0].ID == replicas[1].ID {
		t.Fatalf("replicas must be distinct physical nodes, got %s and %s", replicas[0].ID, replicas[1].ID)
	}

	// Request 5 replicas (should cap at 3 distinct nodes)
	all, err := r.GetNodes("vector-abc", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(all))
	}
}

func TestNodeRemovalAndReassignment(t *testing.T) {
	r := New(64)
	r.AddNode(Node{ID: "node-1", GrpcAddr: "127.0.0.1:7071"})
	r.AddNode(Node{ID: "node-2", GrpcAddr: "127.0.0.1:7072"})

	primaryBefore, err := r.GetNode("test-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Remove the primary node
	err = r.RemoveNode(primaryBefore.ID)
	if err != nil {
		t.Fatalf("failed to remove node: %v", err)
	}

	if r.NodeCount() != 1 {
		t.Fatalf("expected 1 node remaining, got %d", r.NodeCount())
	}

	// The remaining node must now own the key
	primaryAfter, err := r.GetNode("test-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if primaryAfter.ID == primaryBefore.ID {
		t.Fatalf("key was not reassigned after removing %s", primaryBefore.ID)
	}
}

func TestKeyDistributionBalance(t *testing.T) {
	r := New(64)
	nodeCount := 3
	for i := 1; i <= nodeCount; i++ {
		r.AddNode(Node{
			ID:       fmt.Sprintf("node-%d", i),
			GrpcAddr: fmt.Sprintf("127.0.0.1:707%d", i),
		})
	}

	counts := make(map[string]int)
	totalKeys := 3000
	for i := 0; i < totalKeys; i++ {
		key := fmt.Sprintf("vec-%d", i)
		n, err := r.GetNode(key)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		counts[n.ID]++
	}

	// Each node should hold roughly 1/3 of the keys (+/- reasonable standard deviation)
	for id, count := range counts {
		t.Logf("Node %s holds %d keys (%.1f%%)", id, count, float64(count)/float64(totalKeys)*100)
		if count < 600 || count > 1400 {
			t.Errorf("Node %s has unbalanced key distribution: %d keys", id, count)
		}
	}
}
