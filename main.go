package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	nethttp "net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/v3dang/vex/internal/cluster"
	"github.com/v3dang/vex/internal/coordinator"
	vexhttp "github.com/v3dang/vex/internal/http"
	"github.com/v3dang/vex/internal/ring"
	"github.com/v3dang/vex/internal/server"
	"github.com/v3dang/vex/internal/store"
	pb "github.com/v3dang/vex/pb"
	"google.golang.org/grpc"
)

func main() {
	var (
		nodeID     = flag.String("node-id", "", "Unique node identifier (default: hostname)")
		grpcAddr   = flag.String("grpc-addr", "127.0.0.1:7070", "Address to listen for gRPC requests")
		httpAddr   = flag.String("http-addr", "127.0.0.1:8080", "Address to listen for HTTP REST requests")
		gossipAddr = flag.String("gossip-addr", "127.0.0.1:7946", "Address to bind for gossip cluster discovery")
		joinPeers  = flag.String("join", "", "Comma-separated list of peer gossip addresses to join")
		dataDir    = flag.String("data-dir", "", "Directory for storing WAL and snapshots")
		replicas   = flag.Int("replicas", 2, "Number of replicas per vector shard")
		quorum     = flag.Int("quorum", 1, "Write quorum count")
		vnodes     = flag.Int("vnodes", 64, "Virtual nodes per physical node on hash ring")
	)
	flag.Parse()

	// Default node ID to hostname if omitted
	if *nodeID == "" {
		host, err := os.Hostname()
		if err != nil {
			*nodeID = "node-1"
		} else {
			*nodeID = host
		}
	}

	// Default data directory
	if *dataDir == "" {
		home, _ := os.UserHomeDir()
		*dataDir = filepath.Join(home, ".vex", *nodeID)
	}

	log.Printf("[vex] Starting node '%s' (data: %s)", *nodeID, *dataDir)

	// 1. Storage Engine
	s, err := store.Open(*dataDir)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer s.Close()

	// 2. Consistent Hash Ring
	hashRing := ring.New(*vnodes)

	// 3. Coordinator
	coord := coordinator.New(coordinator.Config{
		NodeID:      *nodeID,
		Replicas:    *replicas,
		WriteQuorum: *quorum,
		Timeout:     5 * time.Second,
	}, s, hashRing)
	defer coord.Close()

	// 4. Decentralized Gossip Discovery
	gossipHost, gossipPort, err := cluster.ParseHostPort(*gossipAddr, 7946)
	if err != nil {
		log.Fatalf("parse gossip addr: %v", err)
	}

	clust, err := cluster.New(cluster.Config{
		NodeID:        *nodeID,
		BindAddr:      gossipHost,
		BindPort:      gossipPort,
		AdvertiseAddr: gossipHost,
		AdvertisePort: gossipPort,
		GrpcAddr:      *grpcAddr,
		HttpAddr:      *httpAddr,
	}, hashRing)
	if err != nil {
		log.Fatalf("start cluster gossip: %v", err)
	}

	// Join peer nodes if specified
	if *joinPeers != "" {
		peers := strings.Split(*joinPeers, ",")
		for i := range peers {
			peers[i] = strings.TrimSpace(peers[i])
		}
		nJoined, err := clust.Join(peers)
		if err != nil {
			log.Printf("[cluster] warning: error joining peers %v: %v", peers, err)
		} else {
			log.Printf("[cluster] successfully contacted %d seed peer(s)", nJoined)
		}
	}

	// 5. gRPC Server
	lis, err := net.Listen("tcp", *grpcAddr)
	if err != nil {
		log.Fatalf("listen gRPC %s: %v", *grpcAddr, err)
	}

	grpcServer := grpc.NewServer()
	vexServer := server.New(*nodeID, s, hashRing, coord)
	pb.RegisterVectorStoreServer(grpcServer, vexServer)

	go func() {
		log.Printf("[gRPC] listening on %s", *grpcAddr)
		if err := grpcServer.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			log.Printf("[gRPC] error: %v", err)
		}
	}()

	// 6. HTTP REST Gateway
	httpServer := &nethttp.Server{
		Addr:    *httpAddr,
		Handler: vexhttp.NewServer(*nodeID, s, hashRing, coord).Handler(),
	}

	go func() {
		log.Printf("[HTTP] REST API listening on http://%s", *httpAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != nethttp.ErrServerClosed {
			log.Printf("[HTTP] error: %v", err)
		}
	}()

	fmt.Printf(`
╔═══════════════════════════════════════════════════════╗
║   VEX - Distributed Vector Exchange (Running)         ║
╠═══════════════════════════════════════════════════════╣
║ Node ID    : %-41s║
║ gRPC Addr  : %-41s║
║ HTTP API   : %-41s║
║ Gossip Addr: %-41s║
║ Replicas   : %-41d║
║ Quorum     : %-41d║
╚═══════════════════════════════════════════════════════╝
`, *nodeID, *grpcAddr, *httpAddr, *gossipAddr, *replicas, *quorum)

	// Graceful Shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	log.Println("[vex] Shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = httpServer.Shutdown(shutdownCtx)
	_ = clust.Leave(2 * time.Second)
	grpcServer.GracefulStop()
	_ = coord.Close()
	_ = s.Close()

	log.Println("[vex] Node stopped cleanly.")
}
