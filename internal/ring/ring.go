package ring

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
	"sync"
)

var (
	ErrEmptyRing    = errors.New("hash ring is empty")
	ErrNodeNotFound = errors.New("node not found in ring")
)

// Node represents a physical server in the cluster.
type Node struct {
	ID       string `json:"id"`
	GrpcAddr string `json:"grpc_addr"`
	HttpAddr string `json:"http_addr"`
}

type vnode struct {
	hash uint64
	node Node
}

// Ring implements a consistent hash ring with virtual nodes.
type Ring struct {
	mu     sync.RWMutex
	vnodes int             // number of virtual nodes per physical node
	nodes  map[string]Node // physical node ID -> Node
	ring   []vnode         // sorted list of virtual node hashes
}

// New creates a new Consistent Hash Ring.
// vnodesPerNode defines how many virtual points on the ring each node gets.
// A good default is 64 or 128.
func New(vnodesPerNode int) *Ring {
	if vnodesPerNode <= 0 {
		vnodesPerNode = 128
	}
	return &Ring{
		vnodes: vnodesPerNode,
		nodes:  make(map[string]Node),
		ring:   make([]vnode, 0),
	}
}

func hashKey(key string) uint64 {
	h := sha256.Sum256([]byte(key))
	return binary.BigEndian.Uint64(h[:8])
}

// AddNode registers a physical node and places its virtual nodes on the ring.
func (r *Ring) AddNode(node Node) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// If node already exists, remove its old vnodes first
	if _, exists := r.nodes[node.ID]; exists {
		r.removeNodeLocked(node.ID)
	}

	r.nodes[node.ID] = node

	for i := 0; i < r.vnodes; i++ {
		vkey := fmt.Sprintf("%s#vnode-%d", node.ID, i)
		h := hashKey(vkey)
		r.ring = append(r.ring, vnode{hash: h, node: node})
	}

	// Keep virtual nodes sorted clockwise by hash
	sort.Slice(r.ring, func(i, j int) bool {
		return r.ring[i].hash < r.ring[j].hash
	})
}

// RemoveNode removes a physical node and all its virtual nodes from the ring.
func (r *Ring) RemoveNode(nodeID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.nodes[nodeID]; !exists {
		return ErrNodeNotFound
	}

	r.removeNodeLocked(nodeID)
	return nil
}

func (r *Ring) removeNodeLocked(nodeID string) {
	delete(r.nodes, nodeID)

	filtered := make([]vnode, 0, len(r.ring))
	for _, vn := range r.ring {
		if vn.node.ID != nodeID {
			filtered = append(filtered, vn)
		}
	}
	r.ring = filtered
}

// GetNode returns the primary owner node for a given key.
func (r *Ring) GetNode(key string) (Node, error) {
	nodes, err := r.GetNodes(key, 1)
	if err != nil {
		return Node{}, err
	}
	return nodes[0], nil
}

// GetNodes returns 'count' distinct physical nodes for a given key.
// The first node is the Primary, and subsequent nodes are Replicas.
// If the ring has fewer distinct physical nodes than count, it returns all available nodes.
func (r *Ring) GetNodes(key string, count int) ([]Node, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.ring) == 0 {
		return nil, ErrEmptyRing
	}

	if count <= 0 {
		count = 1
	}

	totalNodes := len(r.nodes)
	if count > totalNodes {
		count = totalNodes
	}

	h := hashKey(key)

	// Binary search clockwise for first vnode with hash >= h
	idx := sort.Search(len(r.ring), func(i int) bool {
		return r.ring[i].hash >= h
	})
	if idx >= len(r.ring) {
		idx = 0 // wrap around
	}

	result := make([]Node, 0, count)
	seen := make(map[string]bool, count)

	for len(result) < count && len(seen) < totalNodes {
		curr := r.ring[idx].node
		if !seen[curr.ID] {
			seen[curr.ID] = true
			result = append(result, curr)
		}
		idx = (idx + 1) % len(r.ring)
	}

	return result, nil
}

// GetAllNodes returns a slice of all distinct physical nodes currently in the ring.
func (r *Ring) GetAllNodes() []Node {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]Node, 0, len(r.nodes))
	for _, n := range r.nodes {
		out = append(out, n)
	}
	return out
}

// NodeCount returns the number of physical nodes in the ring.
func (r *Ring) NodeCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.nodes)
}
