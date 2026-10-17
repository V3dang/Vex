package cluster

import (
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/v3dang/vex/internal/ring"
)

func getFreePort(t *testing.T) int {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to get free port: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func TestClusterJoinAndLeave(t *testing.T) {
	p1 := getFreePort(t)
	p2 := getFreePort(t)

	r1 := ring.New(32)
	c1, err := New(Config{
		NodeID:    "node-1",
		BindAddr:  "127.0.0.1",
		BindPort:  p1,
		GrpcAddr:  "127.0.0.1:7071",
		LogOutput: io.Discard,
	}, r1)
	if err != nil {
		t.Fatalf("failed to create c1: %v", err)
	}
	defer c1.Leave(1 * time.Second)

	if r1.NodeCount() != 1 {
		t.Fatalf("expected 1 node in r1, got %d", r1.NodeCount())
	}

	r2 := ring.New(32)
	c2, err := New(Config{
		NodeID:    "node-2",
		BindAddr:  "127.0.0.1",
		BindPort:  p2,
		GrpcAddr:  "127.0.0.1:7072",
		LogOutput: io.Discard,
	}, r2)
	if err != nil {
		t.Fatalf("failed to create c2: %v", err)
	}
	defer c2.Leave(1 * time.Second)

	// Node 2 joins Node 1
	nJoined, err := c2.Join([]string{fmt.Sprintf("127.0.0.1:%d", p1)})
	if err != nil {
		t.Fatalf("join failed: %v", err)
	}
	if nJoined < 1 {
		t.Fatalf("expected at least 1 joined node, got %d", nJoined)
	}

	// Wait briefly for gossip dissemination
	deadline := time.Now().Add(3 * time.Second)
	synced := false
	for time.Now().Before(deadline) {
		if r1.NodeCount() == 2 && r2.NodeCount() == 2 {
			synced = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if !synced {
		t.Fatalf("nodes did not sync ring membership in time: r1 has %d, r2 has %d", r1.NodeCount(), r2.NodeCount())
	}

	// Verify node metadata was passed accurately into the ring
	n1, err := r1.GetNode("some-test-vector-key")
	if err != nil {
		t.Fatalf("failed to resolve key: %v", err)
	}
	if n1.GrpcAddr != "127.0.0.1:7071" && n1.GrpcAddr != "127.0.0.1:7072" {
		t.Fatalf("expected valid grpc addr, got %s", n1.GrpcAddr)
	}

	// Node 2 leaves
	if err := c2.Leave(500 * time.Millisecond); err != nil {
		t.Fatalf("c2 leave error: %v", err)
	}

	// Wait for r1 to detect leave
	leftSynced := false
	deadline = time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if r1.NodeCount() == 1 {
			leftSynced = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if !leftSynced {
		t.Fatalf("r1 did not remove left node in time: node count is %d", r1.NodeCount())
	}
}
