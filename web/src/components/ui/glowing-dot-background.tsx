"use client";

import React from "react";
import { DottedGlowBackground } from "@/components/ui/dotted-glow-background";
import { cn } from "@/lib/utils";

interface GlowingDotBackgroundProps {
  children?: React.ReactNode;
  className?: string;
}

export function GlowingDotBackground({
  children,
  className,
}: GlowingDotBackgroundProps) {
  return (
    <div
      className={cn(
        "relative w-full min-h-screen bg-[#fafafa] text-zinc-900 overflow-hidden",
        className
      )}
    >
      {/* Official Aceternity Dotted Glow Background Component */}
      <DottedGlowBackground
        className="pointer-events-none opacity-45 [mask-image:radial-gradient(ellipse_at_center,black_40%,transparent_90%)]"
        opacity={0.8}
        gap={12}
        radius={1.6}
        color="rgba(163, 163, 163, 0.95)"
        glowColor="rgba(99, 102, 241, 0.85)"
        colorLightVar="--color-neutral-500"
        glowColorLightVar="--color-indigo-600"
        backgroundOpacity={0}
        speedMin={0.3}
        speedMax={1.4}
        speedScale={1}
      />

      {/* Ambient Gradient Glows */}
      <div className="absolute top-0 left-1/2 -translate-x-1/2 w-[850px] h-[450px] bg-gradient-to-b from-indigo-200/30 via-violet-100/20 to-transparent blur-[120px] pointer-events-none -z-10" />
      <div className="absolute top-[800px] right-0 w-[500px] h-[400px] bg-emerald-100/20 blur-[100px] pointer-events-none -z-10" />
      <div className="absolute top-[1400px] left-0 w-[500px] h-[400px] bg-sky-100/25 blur-[100px] pointer-events-none -z-10" />

      {/* Content Layer */}
      <div className="relative z-10">{children}</div>
    </div>
  );
}
