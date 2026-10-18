#!/usr/bin/env python3
import sys
import os
import time
import subprocess
import signal

# Add parent directory to sys.path so we can import vex
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "python")))
import vex

def build_vex_binary(root_dir):
    vex_bin = os.path.join(root_dir, "vex")
    if not os.path.exists(vex_bin):
        print("[Setup] Building vex binary...")
        subprocess.check_call(["go", "build", "-o", "vex", "main.go"], cwd=root_dir)
    return vex_bin

def main():
    root_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
    vex_bin = build_vex_binary(root_dir)

    temp_data = f"/tmp/vex-py-demo-{os.getpid()}"
    os.makedirs(f"{temp_data}/n1", exist_ok=True)
    os.makedirs(f"{temp_data}/n2", exist_ok=True)
    os.makedirs(f"{temp_data}/n3", exist_ok=True)

    processes = []

    def cleanup():
        print("\n[Teardown] Stopping cluster nodes...")
        for p in processes:
            try:
                p.terminate()
                p.wait(timeout=2)
            except Exception:
                p.kill()
        subprocess.call(["rm", "-rf", temp_data])

    try:
        print("\n=======================================================")
        print("    VEX (Vector Exchange) - Distributed Python Demo    ")
        print("=======================================================\n")

        # 1. Start 3 VEX Nodes
        print("[1/5] Starting 3-node distributed VEX cluster...")
        p1 = subprocess.Popen(
            [vex_bin, "-node-id", "node-1", "-grpc-addr", "127.0.0.1:7071", "-http-addr", "127.0.0.1:8081",
             "-gossip-addr", "127.0.0.1:7946", "-data-dir", f"{temp_data}/n1", "-replicas", "2", "-quorum", "1"],
            stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL
        )
        processes.append(p1)
        time.sleep(1)

        p2 = subprocess.Popen(
            [vex_bin, "-node-id", "node-2", "-grpc-addr", "127.0.0.1:7072", "-http-addr", "127.0.0.1:8082",
             "-gossip-addr", "127.0.0.1:7947", "-join", "127.0.0.1:7946", "-data-dir", f"{temp_data}/n2",
             "-replicas", "2", "-quorum", "1"],
            stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL
        )
        processes.append(p2)

        p3 = subprocess.Popen(
            [vex_bin, "-node-id", "node-3", "-grpc-addr", "127.0.0.1:7073", "-http-addr", "127.0.0.1:8083",
             "-gossip-addr", "127.0.0.1:7948", "-join", "127.0.0.1:7946", "-data-dir", f"{temp_data}/n3",
             "-replicas", "2", "-quorum", "1"],
            stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL
        )
        processes.append(p3)

        print("Waiting for gossip peer convergence...")
        time.sleep(2.5)

        # 2. Connect client with multi-node endpoints
        print("[2/5] Connecting vex.Client to cluster endpoints...")
        client = vex.Client(endpoints=[
            "http://127.0.0.1:8081",
            "http://127.0.0.1:8082",
            "http://127.0.0.1:8083"
        ])

        cluster_info = client.get_cluster_info()
        print(f"Connected! Cluster size: {cluster_info.get('cluster_size')} active nodes.")
        for node in cluster_info.get("nodes", []):
            print(f"  - Node {node['id']} (gRPC: {node['grpc_addr']}, HTTP: {node['http_addr']})")

        # 3. Create Collection and Ingest Documents
        print("\n[3/5] Ingesting documents into 'tech_knowledge' collection...")
        collection = client.get_or_create_collection("tech_knowledge")

        documents = [
            "Consistent hashing and HNSW enable decentralized vector databases to scale horizontally.",
            "Golang concurrency primitives like goroutines and channels make building high-throughput systems straightforward.",
            "SWIM gossip protocol allows distributed nodes to detect failures and discover peers without a coordinator.",
            "Transformer models use self-attention to generate high-dimensional embeddings for semantic search."
        ]

        # Normalized 3D vector embeddings representing semantic spaces
        embeddings = [
            [0.92, 0.25, 0.05],  # Distributed vector DBs
            [0.10, 0.95, 0.15],  # Golang concurrency
            [0.85, 0.35, 0.10],  # Gossip & distributed systems
            [0.05, 0.20, 0.95],  # Transformer models
        ]

        metadatas = [
            {"category": "database", "author": "VexTeam"},
            {"category": "language", "author": "GoTeam"},
            {"category": "distributed", "author": "SWIM"},
            {"category": "ai", "author": "MLTeam"},
        ]

        ids = ["doc_vex", "doc_go", "doc_swim", "doc_ai"]

        collection.add(
            ids=ids,
            documents=documents,
            embeddings=embeddings,
            metadatas=metadatas
        )
        print(f"Successfully inserted {len(documents)} documents (sharded & replicated across cluster).")

        # 4. Perform Semantic Search (Scatter-Gather)
        print("\n[4/5] Performing semantic search query...")
        query_vector = [0.90, 0.28, 0.08] # Looking for distributed vector systems
        print(f"Query Vector: {query_vector}")

        results = collection.query(
            query_embeddings=query_vector,
            n_results=2
        )

        print("\nTop Matches (Scatter-Gather retrieved & merged):")
        for i in range(len(results["ids"])):
            print(f"  [{i+1}] ID: {results['ids'][i]}")
            print(f"      Distance : {results['distances'][i]:.4f}")
            print(f"      Category : {results['metadatas'][i].get('category')}")
            print(f"      Document : \"{results['documents'][i]}\"")

        # 5. Fault Tolerance Test: Kill Node 1
        print("\n[5/5] Testing Node Failure Survival...")
        print("Terminating Node 1 (PID %d)..." % p1.pid)
        p1.send_signal(signal.SIGKILL)
        p1.wait()
        processes.remove(p1)
        time.sleep(1.5)

        print("Searching again via surviving nodes (client auto-fails over)...")
        failover_results = collection.query(
            query_embeddings=query_vector,
            n_results=2
        )

        print("\nQuery results after Node 1 failure:")
        for i in range(len(failover_results["ids"])):
            print(f"  [{i+1}] ID: {failover_results['ids'][i]} (Distance: {failover_results['distances'][i]:.4f})")
            print(f"      Document : \"{failover_results['documents'][i]}\"")

        print("\n=======================================================")
        print("    DEMO SUCCESSFUL: Distributed VEX + Python works!   ")
        print("=======================================================\n")

    finally:
        cleanup()

if __name__ == "__main__":
    main()
