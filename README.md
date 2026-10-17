# VEX (Vector Exchange)

**VEX** is a distributed, masterless vector database built in Go. It combines in-memory **HNSW** (Hierarchical Navigable Small World) graph search with **decentralized SWIM gossip discovery**, **consistent hash sharding**, and **scatter-gather query execution**.

---

## Key Capabilities

- **Decentralized Node Discovery**: Peer-to-peer discovery using the SWIM gossip protocol (`hashicorp/memberlist`). No ZooKeeper, etcd, or central coordinator required.
- **Consistent Hash Partitioning**: Data is sharded across nodes via a consistent hash ring with virtual nodes (128 vnodes per server) using cryptographic SHA-256 distribution.
- **Fault-Tolerant Replication**: Configurable replication factor ($R \ge 2$) and quorum writes ($W$). If a node fails, surviving replicas continue serving queries with zero data loss.
- **Scatter-Gather ANN Search**: Queries sent to any node fan out in parallel to relevant shards across the cluster and merge into a global top-$K$ candidate list.
- **Real ANN Algorithm**: True HNSW graph indexing using [`github.com/coder/hnsw`](https://github.com/coder/hnsw) with exact cosine distance metric calculation.
- **Dual Network Interface**: High-performance gRPC server + embedded HTTP REST API (`/vectors`, `/vectors/search`, `/cluster`, `/health`).
- **Persistence**: Append-only binary Write-Ahead Log (WAL) + Gob snapshotting.

---

## Architecture

```
                          [ Client Application ]
                        (via gRPC, HTTP, or Go SDK)
                                     │
                                     ▼
                      ┌─────────────────────────────┐
                      │  Any VEX Node (Coordinator)  │
                      └──────────────┬──────────────┘
                                     │
         ┌───────────────────────────┼───────────────────────────┐
         │ (Writes: Hash ID)         │ (Searches: Scatter-Gather)│
         ▼                           ▼                           ▼
┌──────────────────┐       ┌──────────────────┐       ┌──────────────────┐
│      Node 1      │◄─────►│      Node 2      │◄─────►│      Node 3      │
│  [Shard A Prim]  │ Gossip│  [Shard B Prim]  │ Gossip│  [Shard C Prim]  │
│  [Shard C Repl]  │ (SWIM)│  [Shard A Repl]  │ (SWIM)│  [Shard B Repl]  │
│   HNSW + WAL     │       │   HNSW + WAL     │       │   HNSW + WAL     │
└──────────────────┘       └──────────────────┘       └──────────────────┘
```

---

## Quickstart: 3-Node Cluster Demo

Run the automated 3-node cluster demo:

```bash
./scripts/demo.sh
```

This script:
1. Builds the `vex` binary.
2. Boots Node 1 (seed), Node 2, and Node 3.
3. Verifies peer discovery and ring convergence.
4. Inserts vectors through Node 1.
5. Executes a scatter-gather ANN query through Node 3.
6. Kills Node 1 (`kill -9`) and performs a search through Node 2 to prove fault-tolerant data survival.

---

## Running a Node

### Compile
```bash
go build -o vex main.go
```

### Start Seed Node (Node 1)
```bash
./vex -node-id node-1 \
      -grpc-addr 127.0.0.1:7071 \
      -http-addr 127.0.0.1:8081 \
      -gossip-addr 127.0.0.1:7946 \
      -data-dir ~/.vex/node1 \
      -replicas 2 -quorum 1
```

### Start Peer Node (Node 2, joining Node 1)
```bash
./vex -node-id node-2 \
      -grpc-addr 127.0.0.1:7072 \
      -http-addr 127.0.0.1:8082 \
      -gossip-addr 127.0.0.1:7947 \
      -join 127.0.0.1:7946 \
      -data-dir ~/.vex/node2 \
      -replicas 2 -quorum 1
```

---

## HTTP REST API Examples

### 1. Insert Vector
```bash
curl -X POST http://127.0.0.1:8081/vectors \
  -H "Content-Type: application/json" \
  -d '{
    "id": "doc-1",
    "vector": [0.1, 0.2, 0.3],
    "payload": {"title": "Distributed Vector Systems", "tags": ["go", "database"]}
  }'
```

### 2. Search Vectors (Scatter-Gather)
```bash
curl -X POST http://127.0.0.1:8082/vectors/search \
  -H "Content-Type: application/json" \
  -d '{
    "vector": [0.1, 0.2, 0.3],
    "k": 5
  }'
```

### 3. Get Vector by ID
```bash
curl http://127.0.0.1:8081/vectors?id=doc-1
```

### 4. Delete Vector
```bash
curl -X DELETE http://127.0.0.1:8081/vectors?id=doc-1
```

### 5. Cluster Status
```bash
curl http://127.0.0.1:8081/cluster
```

---

## Using the Go Client SDK

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/v3dang/vex/client"
)

func main() {
	// Connect to any available nodes in the cluster
	c, err := client.New([]string{"127.0.0.1:7071", "127.0.0.1:7072"})
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer c.Close()

	ctx := context.Background()

	// Insert
	err = c.Insert(ctx, "article-1", []float32{0.25, 0.75, 0.1}, []byte(`{"title":"Go Vectors"}`))
	if err != nil {
		log.Fatalf("insert: %v", err)
	}

	// Search
	results, err := c.Search(ctx, []float32{0.2, 0.7, 0.1}, 5, 20)
	if err != nil {
		log.Fatalf("search: %v", err)
	}

	for _, r := range results {
		fmt.Printf("ID: %s, Distance: %f, Payload: %s\n", r.Id, r.Distance, string(r.Payload))
	}
}
```

---

## Running Tests

Run the complete test suite (unit tests + multi-node cluster tests + failover tests):

```bash
go test -v ./...
```
