"use client";

import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  TooltipProps,
} from "recharts";
import { formatShortDate } from "@/lib/format";
import { useMemo, useState } from "react";
import { cn } from "@/lib/utils";

interface TrendDataPoint {
  date: string;
  value: number | null;
}

interface TrendChartProps {
  data: TrendDataPoint[];
  color: string;
  gradientId: string;
  unit?: string;
  domain?: [number, number];
  height?: number;
}

function ChartTooltip({ active, payload, label, unit }: TooltipProps<number, string> & { unit?: string }) {
  if (!active || !payload?.length) return null;
  return (
    <div className="rounded-lg border border-border-subtle bg-surface-0/95 px-3 py-2 shadow-xl backdrop-blur-xl">
      <p className="text-[10px] text-text-muted mb-1">{label}</p>
      <p className="text-sm font-semibold text-text-primary">
        {payload[0].value != null ? `${Number(payload[0].value).toFixed(1)}${unit || ""}` : "--"}
      </p>
    </div>
  );
}

export function TrendChart({
  data,
  color,
  gradientId,
  unit = "",
  domain,
  height = 200,
}: TrendChartProps) {
  const formattedData = useMemo(
    () => data.map((d) => ({ ...d, label: formatShortDate(d.date) })),
    [data]
  );

  if (!formattedData.length) {
    return (
      <div
        className="flex items-center justify-center text-text-muted text-xs"
        style={{ height }}
      >
        No data
      </div>
    );
  }

  return (
    <div style={{ height }} className="w-full">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={formattedData} margin={{ top: 4, right: 4, left: -24, bottom: 0 }}>
          <defs>
            <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={color} stopOpacity={0.2} />
              <stop offset="95%" stopColor={color} stopOpacity={0} />
            </linearGradient>
          </defs>
          <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.03)" vertical={false} />
          <XAxis
            dataKey="label"
            stroke="transparent"
            tick={{ fill: "var(--color-text-muted)", fontSize: 10 }}
            tickLine={false}
            axisLine={false}
            minTickGap={40}
          />
          <YAxis
            stroke="transparent"
            tick={{ fill: "var(--color-text-muted)", fontSize: 10 }}
            tickLine={false}
            axisLine={false}
            domain={domain || ["auto", "auto"]}
            width={35}
          />
          <Tooltip content={<ChartTooltip unit={unit} />} cursor={{ stroke: "rgba(255,255,255,0.06)" }} />
          <Area
            type="monotone"
            dataKey="value"
            stroke={color}
            fill={`url(#${gradientId})`}
            strokeWidth={2}
            connectNulls
            activeDot={{ r: 3, strokeWidth: 0, fill: color }}
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}

/** Wrapper with period toggle (7d / 30d) */
export function TrendChartWithToggle({
  data,
  color,
  gradientId,
  unit,
  domain,
  height,
}: TrendChartProps) {
  const [period, setPeriod] = useState<7 | 30>(30);
  const filtered = data.slice(-period);

  return (
    <div>
      <div className="flex gap-1 mb-3">
        {([7, 30] as const).map((p) => (
          <button
            key={p}
            onClick={() => setPeriod(p)}
            className={cn(
              "px-2.5 py-1 rounded-md text-[11px] font-medium transition-colors",
              period === p
                ? "bg-accent-muted text-accent"
                : "text-text-muted hover:text-text-secondary"
            )}
          >
            {p}d
          </button>
        ))}
      </div>
      <TrendChart
        data={filtered}
        color={color}
        gradientId={gradientId}
        unit={unit}
        domain={domain}
        height={height}
      />
    </div>
  );
}
