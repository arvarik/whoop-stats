"use client";

import { cn } from "@/lib/utils";
import { formatDuration } from "@/lib/format";
import { motion } from "framer-motion";

interface SleepStagesBarProps {
  light: number;
  rem: number;
  deep: number;
  awake: number;
}

const stages = [
  { key: "awake", label: "Awake", color: "bg-zinc-400" },
  { key: "rem", label: "REM", color: "bg-indigo-400" },
  { key: "light", label: "Light", color: "bg-blue-300" },
  { key: "deep", label: "Deep", color: "bg-blue-600" },
] as const;

export function SleepStagesBar({ light, rem, deep, awake }: SleepStagesBarProps) {
  const values = { awake, rem, light, deep };
  const total = awake + rem + light + deep;

  if (total === 0) return <div className="text-xs text-text-muted">No stage data</div>;

  return (
    <div className="space-y-2.5">
      <div className="flex w-full h-2.5 rounded-full overflow-hidden gap-[1px] bg-surface-2/50">
        {stages.map((stage) => {
          const pct = (values[stage.key] / total) * 100;
          if (pct < 0.5) return null;
          return (
            <motion.div
              key={stage.key}
              initial={{ width: 0 }}
              animate={{ width: `${pct}%` }}
              transition={{ duration: 0.8, ease: "easeOut", delay: 0.1 }}
              className={cn("h-full", stage.color)}
            />
          );
        })}
      </div>
      <div className="flex flex-wrap gap-x-4 gap-y-1">
        {stages.map((stage) => (
          <div key={stage.key} className="flex items-center gap-1.5 text-xs">
            <div className={cn("w-2 h-2 rounded-full", stage.color)} />
            <span className="text-text-secondary">{formatDuration(values[stage.key])}</span>
            <span className="text-text-muted">{stage.label}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
