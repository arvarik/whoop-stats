"use client";

import { MetricCard } from "@/components/metric-card";
import { DetailPopup, DetailRow, useDetailPopup } from "@/components/detail-popup";
import { Flame, Zap, Heart, Activity, Dumbbell, Timer } from "lucide-react";
import { formatCalories, formatDuration, kjToCal } from "@/lib/format";

interface StrainPanelData {
  strain: number | null;
  kj: number | null;
  avgHR: number | null;
  maxHR: number | null;
  weekStrain: number;
  weekKJ: number;
  avgDailyStrain: number | null;
  peakStrain: number | null;
  totalCal: number;
  totalDays: number;
  workoutCount: number;
  highStrainDays: number;
  // Extended
  avgDailyCal: number | null;
  avgWorkoutStrain: number | null;
  avgWorkoutDurationMs: number | null;
  totalWorkoutDurationMs: number;
  sportBreakdown: { sport: string; count: number; avgStrain: number; totalCal: number }[];
  strainDelta: number | null;
  weekAvgDailyStrain: number | null;
}

export function StrainPanels({ data: d }: { data: StrainPanelData }) {
  const { popup, open, close } = useDetailPopup();

  return (
    <>
      {/* Hero stats */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <MetricCard
          title="Today's Strain"
          value={d.strain ? d.strain.toFixed(1) : "--"}
          subtitle={d.kj ? formatCalories(d.kj) : undefined}
          icon={<Activity className="w-4 h-4" />}
          accentColor="blue"
          onClick={() => open("strain")}
        />
        <MetricCard
          title="7-Day Total"
          value={d.weekStrain.toFixed(1)}
          subtitle={formatCalories(d.weekKJ)}
          icon={<Flame className="w-4 h-4" />}
          accentColor="yellow"
          onClick={() => open("weekly")}
        />
        <MetricCard
          title="Avg Heart Rate"
          value={d.avgHR ? `${d.avgHR} bpm` : "--"}
          subtitle="Today's average"
          icon={<Heart className="w-4 h-4" />}
          accentColor="red"
          onClick={() => open("hr")}
        />
        <MetricCard
          title="Max Heart Rate"
          value={d.maxHR ? `${d.maxHR} bpm` : "--"}
          subtitle="Today's peak"
          icon={<Zap className="w-4 h-4" />}
          accentColor="green"
          onClick={() => open("hr")}
        />
      </div>

      {/* Derived metrics */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 mt-3">
        <MetricCard
          title="Avg Daily Strain"
          value={d.avgDailyStrain ? d.avgDailyStrain.toFixed(1) : "--"}
          subtitle={`${d.totalDays}-day average`}
          onClick={() => open("averages")}
        />
        <MetricCard
          title="Peak Strain"
          value={d.peakStrain ? d.peakStrain.toFixed(1) : "--"}
          subtitle="Highest day"
          accentColor="blue"
          onClick={() => open("averages")}
        />
        <MetricCard
          title="Total Calories"
          value={d.totalCal.toLocaleString()}
          subtitle={`${d.totalDays} days tracked`}
          icon={<Flame className="w-4 h-4" />}
          onClick={() => open("calories")}
        />
        <MetricCard
          title="High Strain Days"
          value={d.highStrainDays}
          subtitle={`≥14.0 strain (${d.workoutCount} workouts)`}
          icon={<Dumbbell className="w-4 h-4" />}
          onClick={() => open("sports")}
        />
      </div>

      {/* Today Strain Detail */}
      {popup === "strain" && (
        <DetailPopup title="Strain Score" onClose={close}>
          <p className="text-xs text-text-tertiary mb-4">
            Strain measures the cardiovascular load your body accumulated throughout the day. Scale ranges from 0 to 21, with 21 being maximal effort.
          </p>
          <DetailRow label="Current Strain" value={d.strain ? d.strain.toFixed(1) : "--"} />
          <DetailRow label="Calories Burned" value={d.kj ? formatCalories(d.kj) : "--"} />
          {d.strainDelta != null && (
            <DetailRow label="vs Yesterday" value={`${d.strainDelta > 0 ? "+" : ""}${d.strainDelta.toFixed(1)}`} />
          )}
          <DetailRow label="Average HR" value={d.avgHR ? `${d.avgHR} bpm` : "--"} />
          <DetailRow label="Max HR" value={d.maxHR ? `${d.maxHR} bpm` : "--"} />
          <div className="mt-4 p-3 rounded-lg bg-surface-1/30">
            <p className="text-xs text-text-secondary">
              {d.strain && d.strain >= 14
                ? "High strain day — consider a lighter day tomorrow for recovery."
                : d.strain && d.strain >= 10
                  ? "Moderate strain — good training stimulus."
                  : d.strain && d.strain >= 5
                    ? "Light activity day — good for active recovery."
                    : "Rest day — focus on sleep and nutrition."}
            </p>
          </div>
        </DetailPopup>
      )}

      {/* Weekly Detail */}
      {popup === "weekly" && (
        <DetailPopup title="7-Day Summary" onClose={close}>
          <DetailRow label="Total Strain" value={d.weekStrain.toFixed(1)} />
          <DetailRow label="Avg Daily Strain" value={d.weekAvgDailyStrain ? d.weekAvgDailyStrain.toFixed(1) : "--"} />
          <DetailRow label="Total Calories" value={`${kjToCal(d.weekKJ).toLocaleString()} Cal`} />
          <DetailRow label="Avg Daily Calories" value={d.weekKJ ? `${Math.round(kjToCal(d.weekKJ) / 7).toLocaleString()} Cal` : "--"} />
        </DetailPopup>
      )}

      {/* HR Detail */}
      {popup === "hr" && (
        <DetailPopup title="Heart Rate Analysis" onClose={close}>
          <p className="text-xs text-text-tertiary mb-4">
            Heart rate data from today&apos;s cycle. Higher average HR during activity indicates greater cardiovascular load.
          </p>
          <DetailRow label="Average HR" value={d.avgHR ? `${d.avgHR} bpm` : "--"} />
          <DetailRow label="Max HR" value={d.maxHR ? `${d.maxHR} bpm` : "--"} />
          <DetailRow label="Today's Strain" value={d.strain ? d.strain.toFixed(1) : "--"} />
          <DetailRow label="Calories" value={d.kj ? formatCalories(d.kj) : "--"} />
        </DetailPopup>
      )}

      {/* Averages Detail */}
      {popup === "averages" && (
        <DetailPopup title="Strain Averages" onClose={close}>
          <DetailRow label="Daily Average" value={d.avgDailyStrain ? d.avgDailyStrain.toFixed(1) : "--"} />
          <DetailRow label="Peak Strain" value={d.peakStrain ? d.peakStrain.toFixed(1) : "--"} hint="Highest single-day strain" />
          <DetailRow label="High Strain Days" value={`${d.highStrainDays} (${d.totalDays ? Math.round(d.highStrainDays / d.totalDays * 100) : 0}%)`} hint="Days ≥ 14.0" />
          <DetailRow label="Total Workouts" value={d.workoutCount} />
          {d.avgWorkoutStrain != null && (
            <DetailRow label="Avg Workout Strain" value={d.avgWorkoutStrain.toFixed(1)} />
          )}
          {d.avgWorkoutDurationMs != null && (
            <DetailRow label="Avg Workout Duration" value={formatDuration(d.avgWorkoutDurationMs)} />
          )}
          <DetailRow label="Total Workout Time" value={d.totalWorkoutDurationMs > 0 ? formatDuration(d.totalWorkoutDurationMs) : "--"} />
        </DetailPopup>
      )}

      {/* Calories Detail */}
      {popup === "calories" && (
        <DetailPopup title="Calorie Analysis" onClose={close}>
          <DetailRow label="Total Calories" value={`${d.totalCal.toLocaleString()} Cal`} />
          <DetailRow label="Daily Average" value={d.avgDailyCal ? `${Math.round(d.avgDailyCal).toLocaleString()} Cal` : "--"} />
          <DetailRow label="Days Tracked" value={d.totalDays} />
          <DetailRow label="7-Day Total" value={`${kjToCal(d.weekKJ).toLocaleString()} Cal`} />
        </DetailPopup>
      )}

      {/* Sport Breakdown Detail */}
      {popup === "sports" && d.sportBreakdown.length > 0 && (
        <DetailPopup title="Activity Breakdown" onClose={close}>
          <p className="text-xs text-text-tertiary mb-4">
            Performance by sport type across your dataset.
          </p>
          {d.sportBreakdown.map((s) => (
            <div key={s.sport} className="flex items-center justify-between py-2 border-b border-border-subtle/30 last:border-0">
              <div>
                <span className="text-sm text-text-secondary capitalize">{s.sport}</span>
                <p className="text-[10px] text-text-muted">{s.count} sessions · {s.totalCal.toLocaleString()} Cal</p>
              </div>
              <span className="text-sm font-semibold text-text-primary">{s.avgStrain.toFixed(1)} avg</span>
            </div>
          ))}
          <DetailRow label="Total Workouts" value={d.workoutCount} />
          <DetailRow label="Total Time" value={d.totalWorkoutDurationMs > 0 ? formatDuration(d.totalWorkoutDurationMs) : "--"} />
        </DetailPopup>
      )}
    </>
  );
}
