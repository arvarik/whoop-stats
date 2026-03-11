"use client";

import { useMemo } from "react";

interface RecoveryGaugeProps {
  score: number | null;
  size?: number;
}

export function RecoveryGauge({ score, size = 180 }: RecoveryGaugeProps) {
  const { color, bgColor, strokeDasharray, strokeDashoffset } = useMemo(() => {
    const radius = (size - 20) / 2;
    const circumference = Math.PI * radius; // half circle
    const pct = score ? Math.min(score, 100) / 100 : 0;

    let c = "var(--color-text-muted)";
    let bg = "rgba(255,255,255,0.05)";
    if (score !== null) {
      if (score >= 66) {
        c = "var(--color-recovery-green)";
        bg = "rgba(16, 185, 129, 0.08)";
      } else if (score >= 34) {
        c = "var(--color-recovery-yellow)";
        bg = "rgba(245, 158, 11, 0.08)";
      } else {
        c = "var(--color-recovery-red)";
        bg = "rgba(239, 68, 68, 0.08)";
      }
    }

    return {
      color: c,
      bgColor: bg,
      strokeDasharray: `${circumference}`,
      strokeDashoffset: `${circumference * (1 - pct)}`,
    };
  }, [score, size]);

  const radius = (size - 20) / 2;
  const cx = size / 2;
  const cy = size / 2 + 10;

  return (
    <div className="flex flex-col items-center">
      <svg width={size} height={size * 0.65} viewBox={`0 0 ${size} ${size * 0.65}`}>
        {/* Background arc */}
        <path
          d={`M ${cx - radius} ${cy} A ${radius} ${radius} 0 0 1 ${cx + radius} ${cy}`}
          fill="none"
          stroke="rgba(255,255,255,0.06)"
          strokeWidth="10"
          strokeLinecap="round"
        />
        {/* Filled arc */}
        <path
          d={`M ${cx - radius} ${cy} A ${radius} ${radius} 0 0 1 ${cx + radius} ${cy}`}
          fill="none"
          stroke={color}
          strokeWidth="10"
          strokeLinecap="round"
          strokeDasharray={strokeDasharray}
          strokeDashoffset={strokeDashoffset}
          style={{
            transition: "stroke-dashoffset 1s ease-out, stroke 0.5s ease",
          }}
        />
      </svg>
      <div
        className="flex flex-col items-center -mt-8 px-6 py-3 rounded-2xl"
        style={{ backgroundColor: bgColor }}
      >
        <span className="text-4xl font-bold tracking-tight" style={{ color }}>
          {score !== null ? `${score}%` : "--%"}
        </span>
        <span className="text-xs text-text-tertiary mt-1 uppercase tracking-wider">Recovery</span>
      </div>
    </div>
  );
}
