"use client";

import React from "react";
import { cn } from "@/lib/utils";

interface BentoCardProps {
  title: string;
  description: string;
  icon?: React.ReactNode;
  badge?: string;
  className?: string;
  children?: React.ReactNode;
}

export function BentoCard({
  title,
  description,
  icon,
  badge,
  className,
  children,
}: BentoCardProps) {
  return (
    <div
      className={cn(
        "group relative rounded-2xl border border-zinc-200/90 bg-white/80 p-6 sm:p-7 backdrop-blur-sm",
        "shadow-[0_2px_12px_rgba(0,0,0,0.03)] hover:shadow-[0_8px_24px_rgba(0,0,0,0.06)] hover:border-zinc-300",
        "transition-all duration-300 flex flex-col justify-between overflow-hidden",
        className
      )}
    >
      {/* Subtle top border spotlight accent on hover */}
      <div className="absolute top-0 left-0 right-0 h-[2px] bg-gradient-to-r from-transparent via-zinc-900 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500" />

      <div>
        <div className="flex items-center justify-between mb-4">
          {icon && (
            <div className="w-10 h-10 rounded-xl bg-zinc-100 border border-zinc-200 flex items-center justify-center text-zinc-900">
              {icon}
            </div>
          )}
          {badge && (
            <span className="text-[11px] font-mono font-medium px-2 py-0.5 rounded-md bg-zinc-100 text-zinc-600 border border-zinc-200">
              {badge}
            </span>
          )}
        </div>

        <h3 className="font-bold text-zinc-950 text-base tracking-tight">{title}</h3>
        <p className="text-xs sm:text-sm text-zinc-600 mt-2 leading-relaxed">{description}</p>
      </div>

      {children && <div className="mt-4 pt-4 border-t border-zinc-100">{children}</div>}
    </div>
  );
}
