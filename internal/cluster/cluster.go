package cluster

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/hashicorp/memberlist"
	"github.com/v3dang/vex/internal/ring"
)

// NodeMeta holds cluster-level communication addresses gossiped to peers.
type NodeMeta struct {
	NodeID   string `json:"node_id"`
	GrpcAddr string `json:"grpc_addr"`
	HttpAddr string `json:"http_addr"`
}

// Config specifies gossip and membership configuration.
type Config struct {
	NodeID        string
	BindAddr      string
	BindPort      int
	AdvertiseAddr string
	AdvertisePort int
	GrpcAddr      string
	HttpAddr      string
	LogOutput     io.Writer
}

// Cluster manages decentralized node discovery and gossip membership.
type Cluster struct {
	cfg        Config
	ring       *ring.Ring
	memberlist *memberlist.Memberlist
	mu         sync.Mutex
	shutdown   bool
}

type delegate struct {
	meta []byte
}

func (d *delegate) NodeMeta(limit int) []byte {
	return d.meta
}

func (d *delegate) NotifyMsg([]byte) {}

func (d *delegate) GetBroadcasts(overhead, limit int) [][]byte {
	return nil
}

func (d *delegate) LocalState(join bool) []byte {
	return nil
}

func (d *delegate) MergeRemoteState(buf []byte, join bool) {}

type eventDelegate struct {
	ring *ring.Ring
}

func parseMeta(node *memberlist.Node) ring.Node {
	var meta NodeMeta
	if len(node.Meta) > 0 {
		if err := json.Unmarshal(node.Meta, &meta); err == nil && meta.NodeID != "" {
			return ring.Node{
				ID:       meta.NodeID,
				GrpcAddr: meta.GrpcAddr,
				HttpAddr: meta.HttpAddr,
			}
		}
	}
	return ring.Node{
		ID:       node.Name,
		GrpcAddr: fmt.Sprintf("%s:%d", node.Addr.String(), node.Port),
	}
}

func (e *eventDelegate) NotifyJoin(node *memberlist.Node) {
	rNode := parseMeta(node)
	log.Printf("[cluster] node joined gossip: id=%s grpc=%s addr=%s:%d", rNode.ID, rNode.GrpcAddr, node.Addr, node.Port)
	e.ring.AddNode(rNode)
}

func (e *eventDelegate) NotifyLeave(node *memberlist.Node) {
	rNode := parseMeta(node)
	log.Printf("[cluster] node left gossip: id=%s", rNode.ID)
	_ = e.ring.RemoveNode(rNode.ID)
}

func (e *eventDelegate) NotifyUpdate(node *memberlist.Node) {
	rNode := parseMeta(node)
	log.Printf("[cluster] node updated: id=%s grpc=%s", rNode.ID, rNode.GrpcAddr)
	e.ring.AddNode(rNode)
}

// New creates and starts a new gossip cluster node.
func New(cfg Config, r *ring.Ring) (*Cluster, error) {
	metaBytes, err := json.Marshal(NodeMeta{
		NodeID:   cfg.NodeID,
		GrpcAddr: cfg.GrpcAddr,
		HttpAddr: cfg.HttpAddr,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal node meta: %w", err)
	}

	mConfig := memberlist.DefaultLANConfig()
	if cfg.BindAddr == "127.0.0.1" || cfg.BindAddr == "localhost" {
		mConfig = memberlist.DefaultLocalConfig()
	}

	mConfig.Name = cfg.NodeID
	if cfg.BindAddr != "" {
		mConfig.BindAddr = cfg.BindAddr
	}
	if cfg.BindPort != 0 {
		mConfig.BindPort = cfg.BindPort
	}
	if cfg.AdvertiseAddr != "" {
		mConfig.AdvertiseAddr = cfg.AdvertiseAddr
	}
	if cfg.AdvertisePort != 0 {
		mConfig.AdvertisePort = cfg.AdvertisePort
	}
	if cfg.LogOutput != nil {
		mConfig.LogOutput = cfg.LogOutput
	}

	mConfig.Delegate = &delegate{meta: metaBytes}
	mConfig.Events = &eventDelegate{ring: r}

	ml, err := memberlist.Create(mConfig)
	if err != nil {
		return nil, fmt.Errorf("create memberlist: %w", err)
	}

	// Register self on the local hash ring
	r.AddNode(ring.Node{
		ID:       cfg.NodeID,
		GrpcAddr: cfg.GrpcAddr,
		HttpAddr: cfg.HttpAddr,
	})

	return &Cluster{
		cfg:        cfg,
		ring:       r,
		memberlist: ml,
	}, nil
}

// Join connects this node to one or more known peer addresses in the cluster.
func (c *Cluster) Join(peers []string) (int, error) {
	if len(peers) == 0 {
		return 0, nil
	}
	return c.memberlist.Join(peers)
}

// Leave gracefully departs from the gossip cluster.
func (c *Cluster) Leave(timeout time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.shutdown {
		return nil
	}
	c.shutdown = true

	if err := c.memberlist.Leave(timeout); err != nil {
		_ = c.memberlist.Shutdown()
		return err
	}
	return c.memberlist.Shutdown()
}

// Shutdown immediately stops the memberlist cluster without graceful leave.
func (c *Cluster) Shutdown() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.shutdown {
		return nil
	}
	c.shutdown = true
	return c.memberlist.Shutdown()
}

// Members returns the list of active memberlist nodes.
func (c *Cluster) Members() []*memberlist.Node {
	return c.memberlist.Members()
}

// Ring returns the synchronized consistent hash ring.
func (c *Cluster) Ring() *ring.Ring {
	return c.ring
}

// LocalNode returns this node's memberlist representation.
func (c *Cluster) LocalNode() *memberlist.Node {
	return c.memberlist.LocalNode()
}

// ParseHostPort helper parses "host:port" or ":port" into host and int port.
func ParseHostPort(addr string, defaultPort int) (string, int, error) {
	if addr == "" {
		return "0.0.0.0", defaultPort, nil
	}
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0, err
	}
	if host == "" {
		host = "0.0.0.0"
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", 0, err
	}
	return host, port, nil
}
