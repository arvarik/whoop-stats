"use client";

import { useMemo } from "react";
import { Area, AreaChart, CartesianGrid, XAxis, YAxis, ResponsiveContainer, Tooltip, TooltipProps } from "recharts";
import { format } from "date-fns";

interface DataPoint {
  date: string;
  strain: number | null;
  recovery: number | null;
}

const CustomTooltip = ({ active, payload, label }: TooltipProps<number, string>) => {
  if (active && payload && payload.length) {
    return (
      <div className="rounded-xl border border-white/[0.08] bg-zinc-950/90 p-4 shadow-2xl backdrop-blur-xl">
        <p className="mb-3 text-sm font-medium text-zinc-400">{label}</p>
        <div className="flex flex-col gap-2">
          {payload.map((entry, index) => {
            const isStrain = entry.dataKey === "strain";
            return (
              <div key={index} className="flex items-center justify-between gap-6">
                <div className="flex items-center gap-2">
                  <div 
                    className="w-2 h-2 rounded-full" 
                    style={{ backgroundColor: entry.color }}
                  />
                  <span className="text-sm font-medium text-zinc-300 capitalize">{entry.dataKey}</span>
                </div>
                <span className="font-semibold text-zinc-50">
                  {entry.value !== null && entry.value !== undefined 
                    ? Number(entry.value).toFixed(isStrain ? 1 : 0) + (isStrain ? "" : "%") 
                    : "--"}
                </span>
              </div>
            );
          })}
        </div>
      </div>
    );
  }
  return null;
};

export function StrainRecoveryChart({ data }: { data: DataPoint[] }) {
  const chartConfig = {
    strain: { color: "#3b82f6" }, // blue-500
    recovery: { color: "#10b981" }, // emerald-500
  };

  const formattedData = useMemo(() => {
    return data.map(d => ({
      ...d,
      formattedDate: format(new Date(d.date), "MMM d"),
    }));
  }, [data]);

  return (
    <div className="w-full h-full min-h-[350px] flex flex-col relative group">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold tracking-tight text-white">30-Day Trends</h2>
          <p className="text-sm text-zinc-400">Strain vs Recovery</p>
        </div>
      </div>
      
      <div className="flex-1 w-full h-[300px]">
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={formattedData} margin={{ top: 10, right: 0, left: -20, bottom: 0 }}>
            <defs>
              <linearGradient id="colorStrain" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor={chartConfig.strain.color} stopOpacity={0.3} />
                <stop offset="95%" stopColor={chartConfig.strain.color} stopOpacity={0} />
              </linearGradient>
              <linearGradient id="colorRecovery" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor={chartConfig.recovery.color} stopOpacity={0.3} />
                <stop offset="95%" stopColor={chartConfig.recovery.color} stopOpacity={0} />
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.05)" vertical={false} />
            <XAxis 
              dataKey="formattedDate" 
              stroke="rgba(255,255,255,0.3)" 
              tick={{ fill: "rgba(255,255,255,0.4)", fontSize: 12, fontWeight: 500 }} 
              tickLine={false} 
              axisLine={false} 
              dy={10}
              minTickGap={30}
            />
            <YAxis 
              yAxisId="left" 
              stroke="rgba(255,255,255,0.3)" 
              tick={{ fill: "rgba(255,255,255,0.4)", fontSize: 12, fontWeight: 500 }} 
              tickLine={false} 
              axisLine={false}
              domain={[0, 21]} 
              dx={-10}
            />
            <YAxis 
              yAxisId="right" 
              orientation="right" 
              stroke="rgba(255,255,255,0.3)" 
              tick={{ fill: "rgba(255,255,255,0.4)", fontSize: 12, fontWeight: 500 }} 
              tickLine={false} 
              axisLine={false}
              domain={[0, 100]} 
              dx={10}
            />
            <Tooltip 
              content={<CustomTooltip />} 
              cursor={{ stroke: 'rgba(255,255,255,0.1)', strokeWidth: 1, strokeDasharray: '4 4' }}
            />
            <Area 
              yAxisId="left"
              type="monotone" 
              dataKey="strain" 
              stroke={chartConfig.strain.color} 
              fillOpacity={1} 
              fill="url(#colorStrain)" 
              connectNulls 
              strokeWidth={2}
              activeDot={{ r: 4, strokeWidth: 0, fill: chartConfig.strain.color }}
            />
            <Area 
              yAxisId="right"
              type="monotone" 
              dataKey="recovery" 
              stroke={chartConfig.recovery.color} 
              fillOpacity={1} 
              fill="url(#colorRecovery)" 
              connectNulls
              strokeWidth={2}
              activeDot={{ r: 4, strokeWidth: 0, fill: chartConfig.recovery.color }}
            />
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
