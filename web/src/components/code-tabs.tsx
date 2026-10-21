"use client";

import React, { useState } from "react";
import { Check, Copy } from "lucide-react";

export function CodeTabs() {
  const [activeTab, setActiveTab] = useState<"python" | "golang" | "github">("python");
  const [copied, setCopied] = useState(false);

  const snippets = {
    python: `# 1. Install vex-python directly from GitHub
pip install git+https://github.com/v3dang/vex.git#subdirectory=python

# 2. Ingest & query in your Python RAG / AI pipeline
import vex

# Connect to any seed nodes (auto-discovers peers via gossip)
client = vex.Client(endpoints=[
    "http://localhost:8081",
    "http://localhost:8082",
    "http://localhost:8083"
])

# Create an isolated collection
collection = client.get_or_create_collection("rag_knowledge")

# Ingest documents with embeddings and metadata
collection.add(
    ids=["doc-1", "doc-2"],
    documents=[
        "Consistent hashing and HNSW enable decentralized vector databases.",
        "SWIM gossip protocol allows distributed nodes to detect failures."
    ],
    embeddings=[
        [0.92, 0.25, 0.05],
        [0.10, 0.95, 0.15]
    ],
    metadatas=[
        {"topic": "distributed-systems", "author": "VexTeam"},
        {"topic": "consensus", "author": "SWIM"}
    ]
)

# Parallel scatter-gather semantic search
results = collection.query(
    query_embeddings=[0.90, 0.28, 0.08],
    n_results=1
)

print("Top match :", results["documents"][0])
print("Distance  :", results["distances"][0])`,

    golang: `// 1. Install Go client package
go get github.com/v3dang/vex

// 2. High-performance gRPC client with connection pooling
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/v3dang/vex/client"
)

func main() {
    // Connect to cluster endpoints (automatic round-robin & pooling)
    c, err := client.New([]string{"127.0.0.1:7071", "127.0.0.1:7072"})
    if err != nil {
        log.Fatalf("connect: %v", err)
    }
    defer c.Close()

    ctx := context.Background()

    // Replicate vector write across hash ring (R=2, Quorum=1)
    err = c.Insert(ctx, "vec-1", []float32{0.92, 0.25, 0.05}, []byte("payload"))
    if err != nil {
        log.Fatalf("insert: %v", err)
    }

    // Parallel scatter-gather ANN search across all cluster shards
    results, err := c.Search(ctx, []float32{0.90, 0.28, 0.08}, 5, 20)
    if err != nil {
        log.Fatalf("search: %v", err)
    }

    for _, r := range results {
        fmt.Printf("ID: %s, Distance: %f\\n", r.Id, r.Distance)
    }
}`,

    github: `# 1. Clone repository from GitHub
git clone https://github.com/v3dang/vex.git
cd vex

# 2. Build single binary in Go
go build -o vex main.go

# 3. Option A: Spin up 3-node cluster with Docker Compose
docker compose up -d

# Option B: Run 3 nodes on localhost
./vex -node-id node-1 -grpc-addr 127.0.0.1:7071 -http-addr 127.0.0.1:8081 -gossip-addr 127.0.0.1:7946 -data-dir ./data/n1 &
./vex -node-id node-2 -grpc-addr 127.0.0.1:7072 -http-addr 127.0.0.1:8082 -gossip-addr 127.0.0.1:7947 -join 127.0.0.1:7946 -data-dir ./data/n2 &
./vex -node-id node-3 -grpc-addr 127.0.0.1:7073 -http-addr 127.0.0.1:8083 -gossip-addr 127.0.0.1:7948 -join 127.0.0.1:7946 -data-dir ./data/n3 &

# 4. Verify cluster health via HTTP REST
curl http://localhost:8081/cluster`,
  };

  const copyCode = () => {
    navigator.clipboard.writeText(snippets[activeTab]);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="w-full">
      {/* Tab Switcher */}
      <div className="flex justify-center mb-6">
        <div className="inline-flex p-1 rounded-xl bg-zinc-200/70 border border-zinc-300/80 shadow-inner">
          <button
            onClick={() => setActiveTab("python")}
            className={`px-4 sm:px-5 py-2 rounded-lg text-xs font-semibold transition ${
              activeTab === "python"
                ? "bg-white text-zinc-950 shadow-sm"
                : "text-zinc-600 hover:text-zinc-950"
            }`}
          >
            Python (AI & RAG)
          </button>
          <button
            onClick={() => setActiveTab("golang")}
            className={`px-4 sm:px-5 py-2 rounded-lg text-xs font-semibold transition ${
              activeTab === "golang"
                ? "bg-white text-zinc-950 shadow-sm"
                : "text-zinc-600 hover:text-zinc-950"
            }`}
          >
            Golang SDK
          </button>
          <button
            onClick={() => setActiveTab("github")}
            className={`px-4 sm:px-5 py-2 rounded-lg text-xs font-semibold transition ${
              activeTab === "github"
                ? "bg-white text-zinc-950 shadow-sm"
                : "text-zinc-600 hover:text-zinc-950"
            }`}
          >
            GitHub Install
          </button>
        </div>
      </div>

      {/* Code Container */}
      <div className="rounded-2xl border border-zinc-800 bg-[#090b10] text-zinc-100 shadow-xl overflow-hidden">
        <div className="flex items-center justify-between px-5 py-3 border-b border-zinc-800 bg-zinc-950/70">
          <div className="flex items-center gap-2">
            <span className="w-3 h-3 rounded-full bg-rose-500/80" />
            <span className="w-3 h-3 rounded-full bg-amber-500/80" />
            <span className="w-3 h-3 rounded-full bg-emerald-500/80" />
            <span className="ml-3 text-xs font-mono text-zinc-400">
              {activeTab === "python" ? "main.py" : activeTab === "golang" ? "main.go" : "terminal.sh"}
            </span>
          </div>

          <button
            onClick={copyCode}
            className="flex items-center gap-1.5 text-xs text-zinc-400 hover:text-white transition px-2.5 py-1 rounded bg-zinc-900 border border-zinc-800"
          >
            {copied ? (
              <>
                <Check className="w-3.5 h-3.5 text-emerald-400" />
                <span className="text-emerald-400">Copied</span>
              </>
            ) : (
              <>
                <Copy className="w-3.5 h-3.5" />
                <span>Copy</span>
              </>
            )}
          </button>
        </div>

        <pre className="p-6 text-xs sm:text-sm font-mono overflow-x-auto text-zinc-300 leading-relaxed">
          <code>{snippets[activeTab]}</code>
        </pre>
      </div>
    </div>
  );
}
