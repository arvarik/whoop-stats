"use client";

import { motion } from "framer-motion";

interface SleepStagesProps {
  light: number;
  rem: number;
  deep: number;
  awake: number;
}

export function SleepStagesBar({ light, rem, deep, awake }: SleepStagesProps) {
  const total = light + rem + deep + awake;
  
  if (total === 0) return <div className="text-sm text-zinc-600">No stage data available</div>;

  const getPercent = (val: number) => (val / total) * 100;
  
  const lightPct = getPercent(light);
  const remPct = getPercent(rem);
  const deepPct = getPercent(deep);
  const awakePct = getPercent(awake);

  // Format to hours/mins
  const formatTime = (ms: number) => {
    const hrs = Math.floor(ms / (1000 * 60 * 60));
    const mins = Math.floor((ms % (1000 * 60 * 60)) / (1000 * 60));
    if (hrs === 0) return `${mins}m`;
    return `${hrs}h ${mins}m`;
  };

  const stages = [
    { label: "Awake", pct: awakePct, ms: awake, color: "bg-zinc-400" },
    { label: "REM", pct: remPct, ms: rem, color: "bg-indigo-400" },
    { label: "Light", pct: lightPct, ms: light, color: "bg-blue-300" },
    { label: "SWS (Deep)", pct: deepPct, ms: deep, color: "bg-blue-600" },
  ];

  return (
    <div className="w-full space-y-3">
      <div className="flex items-center justify-between text-xs font-medium text-zinc-500 mb-1">
        <span>Sleep Breakdown</span>
      </div>
      <div className="flex w-full h-3 bg-zinc-800 rounded-full overflow-hidden gap-[1px]">
        {stages.map((stage) => (
          stage.pct > 0 && (
            <motion.div
              key={stage.label}
              initial={{ width: 0 }}
              animate={{ width: `${stage.pct}%` }}
              transition={{ duration: 1, ease: "easeOut" }}
              className={`h-full ${stage.color}`}
              title={`${stage.label}: ${formatTime(stage.ms)}`}
            />
          )
        ))}
      </div>
      <div className="flex justify-between mt-2 flex-wrap gap-2">
        {stages.map((stage) => (
          <div key={stage.label} className="flex items-center gap-1.5 text-xs text-zinc-400">
            <div className={`w-2 h-2 rounded-full ${stage.color}`} />
            <span>{formatTime(stage.ms)} <span className="text-zinc-600">{stage.label}</span></span>
          </div>
        ))}
      </div>
    </div>
  );
}
