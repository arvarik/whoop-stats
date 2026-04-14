"use client";

import { cn } from "@/lib/utils";
import { formatDuration, formatCalories, HR_ZONE_COLORS } from "@/lib/format";
import { Flame, Clock } from "lucide-react";

interface WorkoutCardProps {
  sportName: string;
  strain: number;
  kilojoule: number;
  startTime: string;
  endTime?: string;
  averageHeartRate?: number;
  maxHeartRate?: number;
  zones: number[]; // [zone0ms, zone1ms, zone2ms, zone3ms, zone4ms, zone5ms]
  className?: string;
}

export function WorkoutCard({
  sportName,
  strain,
  kilojoule,
  startTime,
  endTime,
  averageHeartRate,
  maxHeartRate,
  zones,
  className,
}: WorkoutCardProps) {
  const start = new Date(startTime);
  const end = endTime ? new Date(endTime) : null;
  const durationMs = end ? end.getTime() - start.getTime() : 0;
  const totalZoneMs = zones.reduce((acc, z) => acc + (z || 0), 0);

  return (
    <div
      className={cn(
        "group rounded-xl border border-border-subtle bg-surface-0/40 p-4 backdrop-blur-sm transition-all duration-200 hover:border-border-hover hover:bg-surface-1/40",
        className
      )}
    >
      <div className="flex items-start justify-between">
        <div>
          <h4 className="text-sm font-semibold text-text-primary">{sportName || "Activity"}</h4>
          <p className="text-xs text-text-tertiary mt-0.5">
            {start.toLocaleDateString("en-US", { weekday: "short", month: "short", day: "numeric" })}
          </p>
        </div>
        <div className="flex items-center gap-1 text-xs font-medium text-strain">
          <Flame className="w-3.5 h-3.5" />
          {strain ? strain.toFixed(1) : "--"}
        </div>
      </div>

      {/* Stats row */}
      <div className="mt-3 flex items-center gap-4 text-xs text-text-secondary">
        {durationMs > 0 && (
          <span className="flex items-center gap-1">
            <Clock className="w-3 h-3 text-text-muted" />
            {formatDuration(durationMs)}
          </span>
        )}
        <span>{formatCalories(kilojoule)}</span>
        {averageHeartRate ? <span>Avg {averageHeartRate} bpm</span> : null}
        {maxHeartRate ? <span>Max {maxHeartRate} bpm</span> : null}
      </div>

      {/* HR Zones mini bar */}
      {totalZoneMs > 0 && (
        <div className="mt-3 flex w-full h-1.5 rounded-full overflow-hidden gap-[1px]">
          {zones.map((z, i) => {
            const pct = (z / totalZoneMs) * 100;
            if (pct < 0.5) return null;
            return (
              <div
                key={i}
                className="h-full rounded-full transition-all"
                style={{
                  width: `${pct}%`,
                  backgroundColor: HR_ZONE_COLORS[i],
                }}
              />
            );
          })}
        </div>
      )}
    </div>
  );
}
