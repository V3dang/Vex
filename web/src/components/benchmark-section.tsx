"use client";

import React from "react";
import { Clock, Zap, Cpu, ShieldCheck, Check, BarChart3 } from "lucide-react";

export function BenchmarkSection() {
  return (
    <section id="benchmarks" className="max-w-6xl mx-auto px-6 py-12">
      <div className="text-center mb-10">
        <div className="inline-flex items-center gap-1.5 text-xs font-mono uppercase tracking-widest text-zinc-500 mb-2">
          <BarChart3 className="w-3.5 h-3.5 text-emerald-600" />
          Measured Performance
        </div>
        <h2 className="text-2xl sm:text-3xl font-extrabold text-zinc-950">
          Real Numbers. Real Measured Impact.
        </h2>
        <p className="text-sm text-zinc-600 mt-2 max-w-xl mx-auto">
          Benchmarked directly with Go&apos;s standard benchmarking engine (
          <code className="font-mono text-xs bg-zinc-200/80 px-1 py-0.5 rounded">
            go test -bench=. -benchmem
          </code>
          ) on 128-dimensional embeddings.
        </p>
      </div>

      {/* Benchmark Metric Cards Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5">
        {/* Metric 1 */}
        <div className="rounded-2xl border border-zinc-200/90 bg-white p-6 shadow-[0_2px_8px_rgba(0,0,0,0.03)] hover:shadow-md transition">
          <div className="flex items-center justify-between text-zinc-500 text-xs font-mono">
            <span>SEARCH LATENCY</span>
            <Clock className="w-4 h-4 text-emerald-600" />
          </div>
          <div className="mt-4 text-3xl sm:text-4xl font-extrabold tracking-tight text-zinc-950 font-mono">
            49.4 <span className="text-lg font-normal text-zinc-500">µs</span>
          </div>
          <p className="mt-2 text-xs text-zinc-600 leading-relaxed">
            Sub-millisecond approximate nearest neighbor retrieval per query via in-memory HNSW graph.
          </p>
          <div className="mt-4 pt-3 border-t border-zinc-100 flex items-center justify-between text-[11px] text-zinc-500 font-mono">
            <span>BenchmarkStoreSearch</span>
            <span className="text-emerald-700 font-bold">~0.049 ms</span>
          </div>
        </div>

        {/* Metric 2 */}
        <div className="rounded-2xl border border-zinc-200/90 bg-white p-6 shadow-[0_2px_8px_rgba(0,0,0,0.03)] hover:shadow-md transition">
          <div className="flex items-center justify-between text-zinc-500 text-xs font-mono">
            <span>SEARCH CAPACITY</span>
            <Zap className="w-4 h-4 text-amber-500" />
          </div>
          <div className="mt-4 text-3xl sm:text-4xl font-extrabold tracking-tight text-zinc-950 font-mono">
            20,200+ <span className="text-lg font-normal text-zinc-500">QPS</span>
          </div>
          <p className="mt-2 text-xs text-zinc-600 leading-relaxed">
            Concurrent search queries served per second per individual node with linear multi-core scaling.
          </p>
          <div className="mt-4 pt-3 border-t border-zinc-100 flex items-center justify-between text-[11px] text-zinc-500 font-mono">
            <span>Core Scaling</span>
            <span className="text-zinc-800 font-bold">Linear</span>
          </div>
        </div>

        {/* Metric 3 */}
        <div className="rounded-2xl border border-zinc-200/90 bg-white p-6 shadow-[0_2px_8px_rgba(0,0,0,0.03)] hover:shadow-md transition">
          <div className="flex items-center justify-between text-zinc-500 text-xs font-mono">
            <span>DURABLE INSERT</span>
            <Cpu className="w-4 h-4 text-sky-600" />
          </div>
          <div className="mt-4 text-3xl sm:text-4xl font-extrabold tracking-tight text-zinc-950 font-mono">
            139 <span className="text-lg font-normal text-zinc-500">µs</span>
          </div>
          <p className="mt-2 text-xs text-zinc-600 leading-relaxed">
            Binary length-prefixed WAL disk append + HNSW multi-layer graph index update.
          </p>
          <div className="mt-4 pt-3 border-t border-zinc-100 flex items-center justify-between text-[11px] text-zinc-500 font-mono">
            <span>Throughput</span>
            <span className="text-zinc-800 font-bold">7,100+ w/sec</span>
          </div>
        </div>

        {/* Metric 4 */}
        <div className="rounded-2xl border border-zinc-200/90 bg-white p-6 shadow-[0_2px_8px_rgba(0,0,0,0.03)] hover:shadow-md transition">
          <div className="flex items-center justify-between text-zinc-500 text-xs font-mono">
            <span>FAILOVER LOSS</span>
            <ShieldCheck className="w-4 h-4 text-violet-600" />
          </div>
          <div className="mt-4 text-3xl sm:text-4xl font-extrabold tracking-tight text-zinc-950 font-mono">
            0 <span className="text-lg font-normal text-zinc-500">%</span>
          </div>
          <p className="mt-2 text-xs text-zinc-600 leading-relaxed">
            Quorum replication (R=2, W=1). If any node crashes, traffic fails over seamlessly to replicas.
          </p>
          <div className="mt-4 pt-3 border-t border-zinc-100 flex items-center justify-between text-[11px] text-zinc-500 font-mono">
            <span>SWIM Detection</span>
            <span className="text-emerald-700 font-bold">&lt; 2.0 sec</span>
          </div>
        </div>
      </div>

      {/* Comparison Table */}
      <div className="mt-8 rounded-2xl border border-zinc-200/90 bg-white overflow-hidden shadow-sm">
        <div className="px-6 py-4 border-b border-zinc-100 flex items-center justify-between bg-zinc-50/60">
          <span className="text-xs font-bold text-zinc-900 tracking-wider font-mono">
            ARCHITECTURE COMPARISON
          </span>
          <span className="text-xs text-zinc-500">VEX vs Traditional Vector Stores</span>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs">
            <thead>
              <tr className="border-b border-zinc-100 text-zinc-400">
                <th className="py-3 px-6 font-medium">Dimension</th>
                <th className="py-3 px-6 font-bold text-zinc-900">VEX (Vector Exchange)</th>
                <th className="py-3 px-6 font-medium">Centralized Vector DB</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-100 text-zinc-700">
              <tr>
                <td className="py-3 px-6 font-medium text-zinc-900">Coordination</td>
                <td className="py-3 px-6 text-emerald-700 font-semibold flex items-center gap-1.5">
                  <Check className="w-3.5 h-3.5" /> Masterless (SWIM Gossip protocol)
                </td>
                <td className="py-3 px-6 text-zinc-500">Requires ZooKeeper / etcd / Primary leader</td>
              </tr>
              <tr>
                <td className="py-3 px-6 font-medium text-zinc-900">Sharding Mechanism</td>
                <td className="py-3 px-6 text-emerald-700 font-semibold flex items-center gap-1.5">
                  <Check className="w-3.5 h-3.5" /> Consistent Hash Ring (128 vnodes per node)
                </td>
                <td className="py-3 px-6 text-zinc-500">Centralized partition router table</td>
              </tr>
              <tr>
                <td className="py-3 px-6 font-medium text-zinc-900">Search Strategy</td>
                <td className="py-3 px-6 text-emerald-700 font-semibold flex items-center gap-1.5">
                  <Check className="w-3.5 h-3.5" /> Parallel Scatter-Gather + K-Way Heap Merge
                </td>
                <td className="py-3 px-6 text-zinc-500">Broadcast query coordinator</td>
              </tr>
              <tr>
                <td className="py-3 px-6 font-medium text-zinc-900">Single Point of Failure</td>
                <td className="py-3 px-6 text-emerald-700 font-semibold flex items-center gap-1.5">
                  <Check className="w-3.5 h-3.5" /> Zero SPOF (Any node acts as coordinator)
                </td>
                <td className="py-3 px-6 text-zinc-500">Coordinator or master failure halts cluster</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </section>
  );
}
