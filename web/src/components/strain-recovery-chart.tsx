"use client";

import { useMemo } from "react";
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  TooltipProps,
  ReferenceDot,
} from "recharts";
import { formatShortDate } from "@/lib/format";

interface DataPoint {
  date: string;
  strain: number | null;
  recovery: number | null;
  hasWorkout?: boolean;
  workoutName?: string;
}

function CustomTooltip({ active, payload, label }: TooltipProps<number, string>) {
  if (!active || !payload?.length) return null;
  // Check if point has workout
  const dataPayload = payload[0]?.payload;
  return (
    <div className="rounded-xl border border-border-subtle bg-surface-0/95 p-3 shadow-2xl backdrop-blur-xl">
      <p className="mb-2 text-xs font-medium text-text-tertiary">{label}</p>
      {payload.map((entry, i) => {
        const isStrain = entry.dataKey === "strain";
        return (
          <div key={i} className="flex items-center justify-between gap-4">
            <div className="flex items-center gap-1.5">
              <div className="w-2 h-2 rounded-full" style={{ backgroundColor: entry.color }} />
              <span className="text-xs text-text-secondary capitalize">{String(entry.dataKey)}</span>
            </div>
            <span className="text-xs font-semibold text-text-primary">
              {entry.value != null
                ? Number(entry.value).toFixed(isStrain ? 1 : 0) + (isStrain ? "" : "%")
                : "--"}
            </span>
          </div>
        );
      })}
      {dataPayload?.hasWorkout && (
        <div className="mt-1.5 pt-1.5 border-t border-border-subtle flex items-center gap-1.5">
          <div className="w-2 h-2 rounded-full bg-orange-400" />
          <span className="text-[10px] text-orange-300">{dataPayload.workoutName || "Workout"}</span>
        </div>
      )}
    </div>
  );
}

export function StrainRecoveryChart({ data }: { data: DataPoint[] }) {
  const formattedData = useMemo(
    () => data.map((d) => ({ ...d, label: formatShortDate(d.date) })),
    [data]
  );

  if (!formattedData.length) {
    return (
      <div className="flex items-center justify-center h-[300px] text-text-muted text-sm">
        No trend data available yet
      </div>
    );
  }

  // Find workout markers
  const workoutPoints = formattedData
    .map((d, idx) => (d.hasWorkout ? { idx, label: d.label, strain: d.strain } : null))
    .filter(Boolean);

  return (
    <div className="w-full h-[300px]">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={formattedData} margin={{ top: 8, right: 8, left: -20, bottom: 0 }}>
          <defs>
            <linearGradient id="gradStrain" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="var(--color-strain)" stopOpacity={0.25} />
              <stop offset="95%" stopColor="var(--color-strain)" stopOpacity={0} />
            </linearGradient>
            <linearGradient id="gradRecovery" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="var(--color-recovery-green)" stopOpacity={0.25} />
              <stop offset="95%" stopColor="var(--color-recovery-green)" stopOpacity={0} />
            </linearGradient>
          </defs>
          <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.04)" vertical={false} />
          <XAxis
            dataKey="label"
            stroke="transparent"
            tick={{ fill: "var(--color-text-muted)", fontSize: 11 }}
            tickLine={false}
            axisLine={false}
            dy={8}
            minTickGap={40}
          />
          <YAxis
            yAxisId="left"
            stroke="transparent"
            tick={{ fill: "var(--color-text-muted)", fontSize: 11 }}
            tickLine={false}
            axisLine={false}
            domain={[0, 21]}
            width={30}
          />
          <YAxis
            yAxisId="right"
            orientation="right"
            stroke="transparent"
            tick={{ fill: "var(--color-text-muted)", fontSize: 11 }}
            tickLine={false}
            axisLine={false}
            domain={[0, 100]}
            width={30}
          />
          <Tooltip content={<CustomTooltip />} cursor={{ stroke: "rgba(255,255,255,0.06)", strokeWidth: 1 }} />
          <Area
            yAxisId="left"
            type="monotone"
            dataKey="strain"
            stroke="var(--color-strain)"
            fill="url(#gradStrain)"
            strokeWidth={2}
            connectNulls
            activeDot={{ r: 3, strokeWidth: 0, fill: "var(--color-strain)" }}
          />
          <Area
            yAxisId="right"
            type="monotone"
            dataKey="recovery"
            stroke="var(--color-recovery-green)"
            fill="url(#gradRecovery)"
            strokeWidth={2}
            connectNulls
            activeDot={{ r: 3, strokeWidth: 0, fill: "var(--color-recovery-green)" }}
          />
          {/* Workout markers */}
          {workoutPoints.map((wp) => (
            wp && (
              <ReferenceDot
                key={wp.idx}
                yAxisId="left"
                x={wp.label}
                y={wp.strain || 0}
                r={4}
                fill="#fb923c"
                stroke="#0f0f12"
                strokeWidth={2}
              />
            )
          ))}
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}
