"use client";

import React, { useState } from "react";
import {
  Terminal,
  ArrowRight,
  ShieldCheck,
  Cpu,
  Layers,
  Network,
  Sparkles,
  GitBranch,
  Copy,
  Check,
} from "lucide-react";
import { GlowingDotBackground } from "@/components/ui/glowing-dot-background";
import { DottedGlowBackground } from "@/components/ui/dotted-glow-background";
import { BentoCard } from "@/components/ui/bento-card";
import { CodeTabs } from "@/components/code-tabs";
import { BenchmarkSection } from "@/components/benchmark-section";

function GithubIcon({ className = "w-4 h-4" }: { className?: string }) {
  return (
    <svg className={className} fill="currentColor" viewBox="0 0 24 24" aria-hidden="true">
      <path
        fillRule="evenodd"
        d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.53 1.032 1.53 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z"
        clipRule="evenodd"
      />
    </svg>
  );
}

export default function Home() {
  const [copiedHero, setCopiedHero] = useState(false);

  const copyHeroCommand = () => {
    navigator.clipboard.writeText("git clone https://github.com/v3dang/vex.git");
    setCopiedHero(true);
    setTimeout(() => setCopiedHero(false), 2000);
  };

  return (
    <GlowingDotBackground>
      {/* Header */}
      <header className="sticky top-0 z-50 backdrop-blur-md bg-white/75 border-b border-zinc-200/80">
        <div className="max-w-6xl mx-auto px-6 h-16 flex items-center justify-between">
          <div className="flex items-center gap-3">
            {/* Minimalist Typographic Branding (No AI Graphic) */}
            <span className="font-mono font-black text-xl tracking-tighter text-zinc-950">
              VEX
            </span>
            <span className="text-[11px] font-mono text-zinc-500 border border-zinc-200 px-2 py-0.5 rounded-full bg-zinc-100/80">
              v0.1.0 • P2P
            </span>
          </div>

          <nav className="hidden md:flex items-center gap-7 text-xs font-medium text-zinc-600">
            <a href="#benchmarks" className="hover:text-zinc-950 transition">
              Benchmarks
            </a>
            <a href="#how-to-use" className="hover:text-zinc-950 transition">
              Integration & SDK
            </a>
            <a href="#architecture" className="hover:text-zinc-950 transition">
              Architecture
            </a>
            <a href="#deploy" className="hover:text-zinc-950 transition">
              Deploy
            </a>
          </nav>

          <div className="flex items-center gap-3">
            <a
              href="https://github.com/v3dang/vex"
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-1.5 text-xs font-medium bg-zinc-900 text-white px-3.5 py-1.5 rounded-lg hover:bg-zinc-800 transition shadow-sm"
            >
              <GithubIcon className="w-3.5 h-3.5" />
              <span>GitHub</span>
            </a>
          </div>
        </div>
      </header>

      {/* Hero Section */}
      <section className="max-w-5xl mx-auto px-6 pt-20 pb-16 text-center">
        <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full border border-zinc-300 bg-white shadow-[0_1px_2px_rgba(0,0,0,0.05)] text-zinc-800 text-xs font-medium mb-6">
          <Sparkles className="w-3.5 h-3.5 text-amber-500" />
          <span>Real HNSW • SWIM Gossip • Zero Master Node</span>
        </div>

        <h1 className="text-4xl sm:text-6xl font-black tracking-tight text-zinc-950 max-w-4xl mx-auto leading-[1.12]">
          Distributed Vector Exchange{" "}
          <span className="bg-gradient-to-r from-zinc-900 via-zinc-700 to-zinc-500 bg-clip-text text-transparent">
            Engineered for Real-World AI
          </span>
        </h1>

        <p className="mt-6 text-base sm:text-lg text-zinc-600 max-w-2xl mx-auto leading-relaxed">
          A decentralized vector database in Go. Data is sharded across nodes via a Consistent Hash Ring, discovered peer-to-peer with SWIM gossip, and queried via parallel scatter-gather HNSW search.
        </p>

        {/* Quick Launch Commands */}
        <div className="mt-8 flex flex-col sm:flex-row items-center justify-center gap-3">
          <div className="flex items-center gap-2.5 bg-white border border-zinc-300/80 px-4 py-2 rounded-xl font-mono text-xs text-zinc-800 shadow-sm">
            <Terminal className="w-3.5 h-3.5 text-zinc-500" />
            <span>git clone https://github.com/v3dang/vex.git</span>
            <button
              onClick={copyHeroCommand}
              className="ml-2 text-zinc-400 hover:text-zinc-700 transition"
              title="Copy"
            >
              {copiedHero ? (
                <Check className="w-3.5 h-3.5 text-emerald-600" />
              ) : (
                <Copy className="w-3.5 h-3.5" />
              )}
            </button>
          </div>

          <a
            href="#how-to-use"
            className="inline-flex items-center gap-2 text-xs font-semibold bg-zinc-900 text-white px-5 py-2.5 rounded-xl hover:bg-zinc-800 transition shadow-sm"
          >
            <span>Start Building</span>
            <ArrowRight className="w-3.5 h-3.5" />
          </a>
        </div>
      </section>

      {/* Real Benchmarks & Measured Impact Section */}
      <BenchmarkSection />

      {/* How to Use VEX (Python, Golang, GitHub) */}
      <section id="how-to-use" className="max-w-6xl mx-auto px-6 py-12">
        <div className="text-center mb-8">
          <div className="inline-flex items-center gap-1.5 text-xs font-mono uppercase tracking-widest text-zinc-500 mb-2">
            <GitBranch className="w-3.5 h-3.5 text-zinc-700" />
            Developer Integration
          </div>
          <h2 className="text-2xl sm:text-3xl font-extrabold text-zinc-950">
            How to Use VEX in Your Stack
          </h2>
          <p className="text-sm text-zinc-600 mt-2 max-w-xl mx-auto">
            Choose your language below to inspect complete, copy-pasteable code examples.
          </p>
        </div>

        <CodeTabs />
      </section>

      {/* Architecture Bento Grid */}
      <section id="architecture" className="max-w-6xl mx-auto px-6 py-12">
        <div className="text-center mb-10">
          <div className="inline-flex items-center gap-1.5 text-xs font-mono uppercase tracking-widest text-zinc-500 mb-2">
            <Layers className="w-3.5 h-3.5 text-indigo-600" />
            Core Architecture
          </div>
          <h2 className="text-2xl sm:text-3xl font-extrabold text-zinc-950">
            Engineered for Zero Data Loss
          </h2>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <BentoCard
            icon={<Network className="w-5 h-5" />}
            badge="P2P"
            title="SWIM Gossip Peer Discovery"
            description="Nodes communicate peer-to-peer using HashiCorp memberlist. Whenever a node boots with -join, it dynamically synchronizes cluster membership, heartbeat failure detection, and gRPC endpoints without any central coordinator."
          />

          <BentoCard
            icon={<Layers className="w-5 h-5" />}
            badge="128 Vnodes"
            title="Consistent Hash Ring Partitioning"
            description="Every vector ID is mapped to a 64-bit cryptographic SHA-256 position on the hash ring. Each physical node claims 128 virtual positions, preventing hot-spotting and guaranteeing uniform data sharding across all servers."
          />

          <BentoCard
            icon={<Cpu className="w-5 h-5" />}
            badge="Scatter-Gather"
            title="Parallel Scatter-Gather Search"
            description="Because vector similarity is semantic rather than ID-based, the receiving coordinator fans out concurrent ShardSearch RPCs across all shards, collects local top-K candidates, and executes a K-way min-heap merge."
          />

          <BentoCard
            icon={<ShieldCheck className="w-5 h-5" />}
            badge="R=2, W=1"
            title="Quorum Replication & Fault Tolerance"
            description="Every insert is replicated to the primary node and the next unique node clockwise along the ring. Writes succeed upon reaching write quorum, allowing the cluster to survive node failure with zero downtime."
          />
        </div>
      </section>

      {/* Official Aceternity DottedGlowBackground Showcase CTA */}
      <section id="deploy" className="max-w-6xl mx-auto px-6 py-16">
        <div className="relative mx-auto flex w-full max-w-6xl items-center justify-center rounded-3xl border border-zinc-200/90 bg-white/90 p-2 sm:p-4 backdrop-blur-sm shadow-sm overflow-hidden">
          <DottedGlowBackground
            className="pointer-events-none opacity-40 [mask-image:radial-gradient(ellipse_at_center,black_40%,transparent_90%)]"
            opacity={1}
            gap={10}
            radius={1.6}
            colorLightVar="--color-neutral-500"
            glowColorLightVar="--color-neutral-600"
            colorDarkVar="--color-neutral-500"
            glowColorDarkVar="--color-sky-800"
            backgroundOpacity={0}
            speedMin={0.3}
            speedMax={1.6}
            speedScale={1}
          />

          <div className="relative z-10 flex w-full flex-col items-center justify-between space-y-6 px-6 sm:px-10 py-12 text-center md:flex-row md:text-left">
            <div>
              <h2 className="text-center text-3xl font-extrabold tracking-tight text-neutral-900 sm:text-4xl md:text-left">
                Ready to build with{" "}
                <span className="font-mono text-zinc-950 font-black">VEX</span>?
              </h2>
              <p className="mt-3 max-w-lg text-center text-sm text-neutral-600 md:text-left leading-relaxed">
                A masterless, decentralized vector database built for high-performance AI, LLM, and RAG pipelines. Open-source and written in Go.
              </p>
            </div>

            <div className="flex flex-col gap-3 sm:flex-row">
              <a
                href="https://github.com/v3dang/vex"
                target="_blank"
                rel="noreferrer"
                className="inline-flex items-center justify-center rounded-xl border border-neutral-300 bg-white px-6 py-2.5 text-xs font-semibold text-neutral-800 shadow-sm transition-all duration-200 hover:bg-neutral-50 hover:shadow"
              >
                <GithubIcon className="w-4 h-4 mr-2" />
                GitHub Repository
              </a>
              <a
                href="#how-to-use"
                className="inline-flex items-center justify-center rounded-xl bg-zinc-950 px-6 py-2.5 text-xs font-semibold text-white shadow-sm transition-all duration-200 hover:bg-zinc-800 hover:shadow"
              >
                View Code Examples
              </a>
            </div>
          </div>
        </div>
      </section>

      {/* Minimalist Footer */}
      <footer className="border-t border-zinc-200 py-8 text-center text-xs text-zinc-500">
        <p>VEX — Vector Exchange. Built with Golang, HNSW & SWIM Gossip Protocol.</p>
        <p className="mt-1 font-mono text-[11px] text-zinc-400">
          Search: 49.4 µs • Single Node Throughput: 20,200+ QPS • 0% Data Loss
        </p>
      </footer>
    </GlowingDotBackground>
  );
}
