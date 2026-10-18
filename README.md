# VEX (Vector Exchange)

**VEX** is a distributed, masterless vector database built in Go. It combines in-memory **HNSW** (Hierarchical Navigable Small World) approximate nearest neighbor search with **decentralized SWIM gossip peer discovery**, **consistent hash ring sharding**, and **scatter-gather parallel queries**.

---

## Key Architectural Principles

- **No Central Coordinator**: Nodes discover each other dynamically via the SWIM gossip protocol (`hashicorp/memberlist`). No ZooKeeper, etcd, or primary-replica coordinator required.
- **Data Sharded Across Nodes**: Data is partitioned across cluster nodes using a Consistent Hash Ring with 128 virtual nodes per server and 64-bit cryptographic SHA-256 distribution. No single node holds everything.
- **Fault-Tolerant Replication**: Configurable replication factor ($R \ge 2$) and write quorum ($W$). If a node crashes, surviving replicas continue serving reads and writes with zero data loss.
- **Scatter-Gather ANN Search**: Queries sent to any entrypoint node fan out in parallel to relevant shards across the cluster and merge into a global top-$K$ candidate list.
- **Real ANN Algorithm**: True HNSW graph indexing powered by `coder/hnsw` with exact Cosine distance metric calculation.
- **Dual Interface**: High-throughput gRPC service + JSON HTTP REST API.
- **Client Libraries**: Available for both **Python** (`vex-python`) and **Go** (`client/client.go`).

---

## Architecture Overview

```
                          [ Client Application ]
                        (Python, Go, or HTTP REST)
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

### How Routing Works (The IPFS / Dynamo Model)

When a client sends a request to any node (e.g. `http://localhost:8081`):
1. **The receiving node acts as the Coordinator**:
   - For **Writes**: The coordinator hashes the document ID against the Consistent Hash Ring. If the key belongs to Node 2 and Node 3, Node 1 forwards the write to Node 2 and Node 3 over internal gRPC without storing it locally.
   - For **Searches**: Because semantic similarity spans the entire vector space, the coordinator fans out `ShardSearch` RPCs in parallel across all cluster shards, collects candidate vectors with exact distances, and performs a K-way merge into a global top-$K$ result.

---

## Python Client (`vex-python`)

The Python client includes automatic multi-node peer discovery, client-side round-robin, failover, and Chroma-like collections.

### 1. Run the Python Demo
```bash
python3 examples/demo.py
```
This automated demo:
1. Compiles and starts a 3-node distributed VEX cluster.
2. Connects using `vex.Client` across the cluster.
3. Ingests documents with high-dimensional vector embeddings and metadata.
4. Performs scatter-gather semantic search.
5. Kills Node 1 (`kill -9`) and performs search again through surviving nodes to prove zero downtime and zero data loss.

### 2. Python Code Example
```python
import vex

# Connect to any seed nodes in the cluster (client auto-discovers peers)
client = vex.Client(endpoints=[
    "http://127.0.0.1:8081",
    "http://127.0.0.1:8082",
    "http://127.0.0.1:8083"
])

# Create or access a collection
collection = client.get_or_create_collection("articles")

# Add documents with embeddings and metadata
collection.add(
    ids=["doc-1", "doc-2"],
    documents=[
        "Consistent hashing enables masterless distributed vector databases.",
        "Transformer architectures power modern semantic embedding models."
    ],
    embeddings=[
        [0.92, 0.25, 0.05],
        [0.05, 0.20, 0.95]
    ],
    metadatas=[
        {"topic": "distributed-systems", "author": "Lamport"},
        {"topic": "ai", "author": "Vaswani"}
    ]
)

# Search by query embedding
results = collection.query(
    query_embeddings=[0.90, 0.28, 0.08],
    n_results=1
)

print("Top match:", results["documents"][0])
print("Distance :", results["distances"][0])
print("Metadata :", results["metadatas"][0])
```

---

## Local Launch & Development

### Compile Binary
```bash
go build -o vex main.go
```

### Launch a 3-Node Cluster on Localhost

**Terminal 1 (Seed Node):**
```bash
./vex -node-id node-1 \
      -grpc-addr 127.0.0.1:7071 \
      -http-addr 127.0.0.1:8081 \
      -gossip-addr 127.0.0.1:7946 \
      -data-dir ~/.vex/node1 \
      -replicas 2 -quorum 1
```

**Terminal 2 (Node 2 joining Node 1):**
```bash
./vex -node-id node-2 \
      -grpc-addr 127.0.0.1:7072 \
      -http-addr 127.0.0.1:8082 \
      -gossip-addr 127.0.0.1:7947 \
      -join 127.0.0.1:7946 \
      -data-dir ~/.vex/node2 \
      -replicas 2 -quorum 1
```

**Terminal 3 (Node 3 joining Node 1):**
```bash
./vex -node-id node-3 \
      -grpc-addr 127.0.0.1:7073 \
      -http-addr 127.0.0.1:8083 \
      -gossip-addr 127.0.0.1:7948 \
      -join 127.0.0.1:7946 \
      -data-dir ~/.vex/node3 \
      -replicas 2 -quorum 1
```

---

## Production Deployment

### Option 1: Docker Compose (Turnkey Multi-Node Cluster)

Launch a 3-node cluster with dedicated persistent storage volumes:
```bash
docker compose up -d
```

Check cluster status:
```bash
curl http://localhost:8081/cluster
```

### Option 2: Multi-Host Bare Metal / Cloud VMs (AWS EC2, GCP, DigitalOcean)

On each host, run VEX pointing to the host's private IP:

**On Server 1 (`10.0.0.1`):**
```bash
vex -node-id node-1 \
    -grpc-addr 0.0.0.0:7070 \
    -http-addr 0.0.0.0:8080 \
    -gossip-addr 0.0.0.0:7946 \
    -data-dir /var/lib/vex \
    -replicas 3 -quorum 2
```

**On Server 2 (`10.0.0.2`):**
```bash
vex -node-id node-2 \
    -grpc-addr 0.0.0.0:7070 \
    -http-addr 0.0.0.0:8080 \
    -gossip-addr 0.0.0.0:7946 \
    -join 10.0.0.1:7946 \
    -data-dir /var/lib/vex \
    -replicas 3 -quorum 2
```

**On Server 3 (`10.0.0.3`):**
```bash
vex -node-id node-3 \
    -grpc-addr 0.0.0.0:7070 \
    -http-addr 0.0.0.0:8080 \
    -gossip-addr 0.0.0.0:7946 \
    -join 10.0.0.1:7946 \
    -data-dir /var/lib/vex \
    -replicas 3 -quorum 2
```

### Option 3: Kubernetes StatefulSet

Deploy VEX as a `StatefulSet` with:
- `volumeClaimTemplates` mount persistent SSD volumes to `/data`.
- A headless Service (`vex-internal`) enables DNS discovery: `vex-0.vex-internal`, `vex-1.vex-internal`.
- Pods join `vex-0.vex-internal:7946` as the seed address.

---

## HTTP REST API Reference

### Insert Vector
`POST /vectors`
```bash
curl -X POST http://localhost:8081/vectors \
  -H "Content-Type: application/json" \
  -d '{
    "id": "doc-1",
    "vector": [0.1, 0.2, 0.3],
    "payload": {"title": "Distributed Vector Systems"}
  }'
```

### Search Vectors (Scatter-Gather)
`POST /vectors/search`
```bash
curl -X POST http://localhost:8081/vectors/search \
  -H "Content-Type: application/json" \
  -d '{
    "vector": [0.1, 0.2, 0.3],
    "k": 5
  }'
```

### Get Vector by ID
`GET /vectors?id=doc-1`
```bash
curl http://localhost:8081/vectors?id=doc-1
```

### Delete Vector
`DELETE /vectors?id=doc-1`
```bash
curl -X DELETE http://localhost:8081/vectors?id=doc-1
```

### Cluster Status & Health
```bash
curl http://localhost:8081/cluster
curl http://localhost:8081/health
```

---

## Go Client SDK

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/v3dang/vex/client"
)

func main() {
	c, err := client.New([]string{"127.0.0.1:7071", "127.0.0.1:7072"})
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer c.Close()

	ctx := context.Background()

	// Insert
	_ = c.Insert(ctx, "article-1", []float32{0.25, 0.75, 0.1}, []byte(`{"title":"Go Vectors"}`))

	// Search
	results, err := c.Search(ctx, []float32{0.2, 0.7, 0.1}, 5, 20)
	if err != nil {
		log.Fatalf("search: %v", err)
	}

	for _, r := range results {
		fmt.Printf("ID: %s, Distance: %f\n", r.Id, r.Distance)
	}
}
```

---

## Running the Test Suite

Run unit and cluster integration tests across all packages:

```bash
go test -v ./...
```
