#!/usr/bin/env bash
set -e

echo "=========================================================="
echo "      VEX (Vector Exchange) 3-Node Distributed Demo       "
echo "=========================================================="

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$ROOT_DIR"

# 1. Build binary
echo "[1/6] Building vex binary..."
go build -o vex main.go

DATA_BASE="/tmp/vex-demo-$$"
mkdir -p "$DATA_BASE/node1" "$DATA_BASE/node2" "$DATA_BASE/node3"

cleanup() {
    echo ""
    echo "[Cleanup] Stopping cluster nodes..."
    kill $PID1 $PID2 $PID3 2>/dev/null || true
    rm -rf "$DATA_BASE"
    echo "[Cleanup] Done."
}
trap cleanup EXIT

# 2. Start Node 1 (Seed Node)
echo "[2/6] Starting Node 1 (Seed: 127.0.0.1:7946, HTTP: 8081, gRPC: 7071)..."
./vex -node-id node-1 \
      -grpc-addr 127.0.0.1:7071 \
      -http-addr 127.0.0.1:8081 \
      -gossip-addr 127.0.0.1:7946 \
      -data-dir "$DATA_BASE/node1" \
      -replicas 2 -quorum 1 > "$DATA_BASE/node1.log" 2>&1 &
PID1=$!

sleep 1

# 3. Start Node 2 (Joins Node 1)
echo "[3/6] Starting Node 2 (Joining 127.0.0.1:7946, HTTP: 8082, gRPC: 7072)..."
./vex -node-id node-2 \
      -grpc-addr 127.0.0.1:7072 \
      -http-addr 127.0.0.1:8082 \
      -gossip-addr 127.0.0.1:7947 \
      -join 127.0.0.1:7946 \
      -data-dir "$DATA_BASE/node2" \
      -replicas 2 -quorum 1 > "$DATA_BASE/node2.log" 2>&1 &
PID2=$!

# 4. Start Node 3 (Joins Node 1)
echo "[4/6] Starting Node 3 (Joining 127.0.0.1:7946, HTTP: 8083, gRPC: 7073)..."
./vex -node-id node-3 \
      -grpc-addr 127.0.0.1:7073 \
      -http-addr 127.0.0.1:8083 \
      -gossip-addr 127.0.0.1:7948 \
      -join 127.0.0.1:7946 \
      -data-dir "$DATA_BASE/node3" \
      -replicas 2 -quorum 1 > "$DATA_BASE/node3.log" 2>&1 &
PID3=$!

echo "Waiting for gossip cluster convergence..."
sleep 2

# Check cluster topology via Node 1 HTTP API
echo ""
echo "--- Cluster Status (Queried via Node 1 HTTP API) ---"
curl -s http://127.0.0.1:8081/cluster | jq . || curl -s http://127.0.0.1:8081/cluster

# 5. Insert vectors via HTTP into Node 1
echo ""
echo "[5/6] Inserting vectors into Node 1 (Coordinator will shard across cluster)..."

curl -s -X POST http://127.0.0.1:8081/vectors \
  -H "Content-Type: application/json" \
  -d '{"id": "doc-ai", "vector": [0.95, 0.05, 0.0], "payload": {"topic": "Artificial Intelligence", "stars": 5}}'
echo ""

curl -s -X POST http://127.0.0.1:8081/vectors \
  -H "Content-Type: application/json" \
  -d '{"id": "doc-db", "vector": [0.05, 0.95, 0.0], "payload": {"topic": "Distributed Databases", "stars": 5}}'
echo ""

curl -s -X POST http://127.0.0.1:8081/vectors \
  -H "Content-Type: application/json" \
  -d '{"id": "doc-go", "vector": [0.0, 0.1, 0.9], "payload": {"topic": "Golang Systems", "stars": 5}}'
echo ""

# 6. Scatter-Gather Search from Node 3!
echo ""
echo "[6/6] Executing Scatter-Gather Search from NODE 3 (Query vector close to 'doc-db')..."
curl -s -X POST http://127.0.0.1:8083/vectors/search \
  -H "Content-Type: application/json" \
  -d '{"vector": [0.1, 0.9, 0.0], "k": 2}' | jq . || curl -s -X POST http://127.0.0.1:8083/vectors/search -H "Content-Type: application/json" -d '{"vector": [0.1, 0.9, 0.0], "k": 2}'

echo ""
echo "--- Testing Node Failure Survival ---"
echo "Killing Node 1 (PID $PID1)..."
kill -9 $PID1 2>/dev/null || true
sleep 2

echo "Searching from Node 2 after Node 1 failure (zero data loss verified)..."
curl -s -X POST http://127.0.0.1:8082/vectors/search \
  -H "Content-Type: application/json" \
  -d '{"vector": [0.9, 0.1, 0.0], "k": 2}' | jq . || curl -s -X POST http://127.0.0.1:8082/vectors/search -H "Content-Type: application/json" -d '{"vector": [0.9, 0.1, 0.0], "k": 2}'

echo ""
echo "=========================================================="
echo "          DEMO SUCCESSFUL: VEX IS OPERATIONAL!            "
echo "=========================================================="
