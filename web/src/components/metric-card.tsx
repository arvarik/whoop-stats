"use client";

import { ReactNode } from "react";
import { cn } from "@/lib/utils";

interface MetricCardProps {
  title: string;
  value: string | number;
  subtitle?: ReactNode;
  icon?: ReactNode;
  accentColor?: "green" | "yellow" | "red" | "blue" | "violet" | "none";
  className?: string;
  children?: ReactNode;
  onClick?: () => void;
}

const accentMap: Record<string, { glow: string; border: string }> = {
  green: { glow: "bg-emerald-500", border: "hover:border-emerald-500/20" },
  yellow: { glow: "bg-amber-500", border: "hover:border-amber-500/20" },
  red: { glow: "bg-rose-500", border: "hover:border-rose-500/20" },
  blue: { glow: "bg-blue-500", border: "hover:border-blue-500/20" },
  violet: { glow: "bg-violet-500", border: "hover:border-violet-500/20" },
  none: { glow: "", border: "" },
};

export function MetricCard({
  title,
  value,
  subtitle,
  icon,
  accentColor = "none",
  className,
  children,
  onClick,
}: MetricCardProps) {
  const accent = accentMap[accentColor];

  return (
    <div
      onClick={onClick}
      className={cn(
        "group relative overflow-hidden rounded-2xl border border-border-subtle bg-surface-0/60 p-5 backdrop-blur-xl transition-all duration-300",
        accent.border,
        onClick && "cursor-pointer",
        className
      )}
    >
      {/* Corner glow */}
      {accentColor !== "none" && (
        <div
          className={cn(
            "absolute -top-12 -right-12 w-24 h-24 rounded-full blur-[40px] opacity-30 transition-opacity duration-500 group-hover:opacity-50",
            accent.glow
          )}
        />
      )}

      <div className="relative z-10 flex flex-col h-full">
        <div className="flex items-center justify-between">
          <h3 className="text-xs font-medium uppercase tracking-wider text-text-tertiary">
            {title}
          </h3>
          {icon && (
            <div className="text-text-muted group-hover:text-text-tertiary transition-colors">
              {icon}
            </div>
          )}
        </div>

        <div className="mt-3 flex items-baseline gap-2">
          <span className="text-3xl font-semibold tracking-tight text-text-primary">
            {value}
          </span>
        </div>

        {subtitle && (
          <div className="mt-1.5 text-sm text-text-tertiary">{subtitle}</div>
        )}

        {children && (
          <div className="mt-4 flex-1 flex flex-col justify-end">{children}</div>
        )}
      </div>
    </div>
  );
}
