"use client";

import { useState, useMemo } from "react";
import { WorkoutCard } from "@/components/workout-card";
import { WorkoutDetail } from "@/components/workout-detail";
import { Dumbbell, SlidersHorizontal, X, Flame, Clock, ChevronUp, ChevronDown, Timer } from "lucide-react";
import { formatCalories, formatDuration, kjToCal } from "@/lib/format";
import { cn } from "@/lib/utils";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type AnyRecord = Record<string, any>;

const SPORT_ICONS: Record<string, { emoji: string; color: string }> = {
  running: { emoji: "🏃", color: "bg-blue-500/20 text-blue-400 border-blue-500/30" },
  walking: { emoji: "🚶", color: "bg-emerald-500/20 text-emerald-400 border-emerald-500/30" },
  weightlifting: { emoji: "🏋️", color: "bg-orange-500/20 text-orange-400 border-orange-500/30" },
  tennis: { emoji: "🎾", color: "bg-yellow-500/20 text-yellow-400 border-yellow-500/30" },
  "hiking-rucking": { emoji: "🥾", color: "bg-green-500/20 text-green-400 border-green-500/30" },
  activity: { emoji: "⚡", color: "bg-violet-500/20 text-violet-400 border-violet-500/30" },
  cycling: { emoji: "🚴", color: "bg-cyan-500/20 text-cyan-400 border-cyan-500/30" },
  swimming: { emoji: "🏊", color: "bg-sky-500/20 text-sky-400 border-sky-500/30" },
  yoga: { emoji: "🧘", color: "bg-pink-500/20 text-pink-400 border-pink-500/30" },
};

function getEffortLevel(strain: number): "easy" | "moderate" | "hard" | "max" {
  if (strain >= 14) return "max";
  if (strain >= 10) return "hard";
  if (strain >= 5) return "moderate";
  return "easy";
}

function getWorkoutDurationMs(w: AnyRecord): number {
  if (!w.end_time) return 0;
  return new Date(w.end_time).getTime() - new Date(w.start_time).getTime();
}

const EFFORT_LEVELS = [
  { key: "easy", label: "Easy", range: "< 5", color: "bg-emerald-500/20 text-emerald-400 border-emerald-500/30" },
  { key: "moderate", label: "Moderate", range: "5-10", color: "bg-yellow-500/20 text-yellow-400 border-yellow-500/30" },
  { key: "hard", label: "Hard", range: "10-14", color: "bg-orange-500/20 text-orange-400 border-orange-500/30" },
  { key: "max", label: "Max", range: "14+", color: "bg-rose-500/20 text-rose-400 border-rose-500/30" },
] as const;

const DURATION_FILTERS = [
  { key: "any", label: "Any" },
  { key: "short", label: "< 20m" },
  { key: "medium", label: "20-45m" },
  { key: "long", label: "45m+" },
] as const;

type SortKey = "date" | "strain" | "duration" | "calories";

interface WorkoutFeedProps {
  workouts: AnyRecord[];
}

export function WorkoutFeed({ workouts }: WorkoutFeedProps) {
  const [selectedSports, setSelectedSports] = useState<Set<string>>(new Set());
  const [selectedEffort, setSelectedEffort] = useState<Set<string>>(new Set());
  const [durationFilter, setDurationFilter] = useState<string>("any");
  const [sortBy, setSortBy] = useState<SortKey>("date");
  const [sortAsc, setSortAsc] = useState(false); // false = descending (default)
  const [detailWorkout, setDetailWorkout] = useState<AnyRecord | null>(null);

  const sportTypes = useMemo(() => {
    const counts: Record<string, number> = {};
    workouts.forEach((w) => {
      const sport = (w.sport_name || "activity").toLowerCase();
      counts[sport] = (counts[sport] || 0) + 1;
    });
    return Object.entries(counts).sort(([, a], [, b]) => b - a);
  }, [workouts]);

  const filtered = useMemo(() => {
    let result = [...workouts];

    if (selectedSports.size > 0) {
      result = result.filter((w) =>
        selectedSports.has((w.sport_name || "activity").toLowerCase())
      );
    }

    if (selectedEffort.size > 0) {
      result = result.filter((w) =>
        selectedEffort.has(getEffortLevel(Number(w.strain || 0)))
      );
    }

    if (durationFilter !== "any") {
      result = result.filter((w) => {
        const dMin = getWorkoutDurationMs(w) / (1000 * 60);
        if (durationFilter === "short") return dMin < 20;
        if (durationFilter === "medium") return dMin >= 20 && dMin < 45;
        if (durationFilter === "long") return dMin >= 45;
        return true;
      });
    }

    const dir = sortAsc ? 1 : -1;
    result.sort((a, b) => {
      let cmp = 0;
      if (sortBy === "strain") cmp = (Number(a.strain) || 0) - (Number(b.strain) || 0);
      else if (sortBy === "calories") cmp = (Number(a.kilojoule) || 0) - (Number(b.kilojoule) || 0);
      else if (sortBy === "duration") cmp = getWorkoutDurationMs(a) - getWorkoutDurationMs(b);
      else cmp = new Date(a.start_time).getTime() - new Date(b.start_time).getTime();
      return cmp * dir;
    });

    return result;
  }, [workouts, selectedSports, selectedEffort, durationFilter, sortBy, sortAsc]);

  const toggleSport = (sport: string) => {
    setSelectedSports((prev) => {
      const next = new Set(prev);
      if (next.has(sport)) next.delete(sport);
      else next.add(sport);
      return next;
    });
  };

  const toggleEffort = (level: string) => {
    setSelectedEffort((prev) => {
      const next = new Set(prev);
      if (next.has(level)) next.delete(level);
      else next.add(level);
      return next;
    });
  };

  const handleSort = (key: SortKey) => {
    if (sortBy === key) {
      setSortAsc((prev) => !prev);
    } else {
      setSortBy(key);
      setSortAsc(false);
    }
  };

  const hasFilters = selectedSports.size > 0 || selectedEffort.size > 0 || durationFilter !== "any";
  const totalStrain = filtered.reduce((acc, w) => acc + (Number(w.strain) || 0), 0);
  const totalCal = filtered.reduce((acc, w) => acc + kjToCal(Number(w.kilojoule || 0)), 0);
  const totalDurationMs = filtered.reduce((acc, w) => acc + getWorkoutDurationMs(w), 0);

  return (
    <div className="space-y-4">
      {/* Sport type icons */}
      <div className="glass-card p-4">
        <div className="flex items-center gap-2 mb-3">
          <SlidersHorizontal className="w-3.5 h-3.5 text-text-muted" />
          <span className="text-xs font-medium uppercase tracking-wider text-text-tertiary">
            Activity Type
          </span>
        </div>
        <div className="flex flex-wrap gap-2">
          {sportTypes.map(([sport, count]) => {
            const isActive = selectedSports.has(sport);
            const info = SPORT_ICONS[sport] || { emoji: "⚡", color: "bg-violet-500/20 text-violet-400 border-violet-500/30" };
            return (
              <button
                key={sport}
                onClick={() => toggleSport(sport)}
                className={cn(
                  "flex items-center gap-1.5 px-3 py-1.5 rounded-full border text-xs font-medium transition-all",
                  isActive
                    ? info.color
                    : "border-border-subtle text-text-muted hover:text-text-secondary hover:border-border-hover"
                )}
              >
                <span>{info.emoji}</span>
                <span className="capitalize">{sport.replace("-", " ")}</span>
                <span className="ml-0.5 opacity-60">×{count}</span>
              </button>
            );
          })}
        </div>
      </div>

      {/* Effort + Duration + Sort row */}
      <div className="flex flex-wrap gap-3">
        {/* Effort filter */}
        <div className="flex items-center gap-1.5">
          <span className="text-[10px] font-medium uppercase tracking-wider text-text-muted mr-1">Effort</span>
          {EFFORT_LEVELS.map((lvl) => {
            const isActive = selectedEffort.has(lvl.key);
            return (
              <button
                key={lvl.key}
                onClick={() => toggleEffort(lvl.key)}
                className={cn(
                  "px-2.5 py-1 rounded-md border text-[11px] font-medium transition-all",
                  isActive
                    ? lvl.color
                    : "border-border-subtle text-text-muted hover:text-text-secondary"
                )}
                title={`Strain ${lvl.range}`}
              >
                {lvl.label}
              </button>
            );
          })}
        </div>

        {/* Duration filter */}
        <div className="flex items-center gap-1.5">
          <Clock className="w-3 h-3 text-text-muted" />
          {DURATION_FILTERS.map((d) => (
            <button
              key={d.key}
              onClick={() => setDurationFilter(d.key)}
              className={cn(
                "px-2.5 py-1 rounded-md text-[11px] font-medium transition-colors",
                durationFilter === d.key
                  ? "bg-accent-muted text-accent"
                  : "text-text-muted hover:text-text-secondary"
              )}
            >
              {d.label}
            </button>
          ))}
        </div>

        {/* Sort with direction arrows */}
        <div className="flex items-center gap-1.5 ml-auto">
          <span className="text-[10px] font-medium uppercase tracking-wider text-text-muted mr-1">Sort</span>
          {(["date", "strain", "duration", "calories"] as const).map((s) => {
            const isActive = sortBy === s;
            return (
              <button
                key={s}
                onClick={() => handleSort(s)}
                className={cn(
                  "flex items-center gap-0.5 px-2.5 py-1 rounded-md text-[11px] font-medium transition-colors capitalize",
                  isActive
                    ? "bg-accent-muted text-accent"
                    : "text-text-muted hover:text-text-secondary"
                )}
              >
                {s}
                {isActive && (
                  sortAsc
                    ? <ChevronUp className="w-3 h-3" />
                    : <ChevronDown className="w-3 h-3" />
                )}
              </button>
            );
          })}
        </div>
      </div>

      {/* Summary with total duration */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 text-xs text-text-tertiary flex-wrap">
          <span className="font-medium text-text-secondary">{filtered.length}</span>
          <span>workouts</span>
          <span className="text-text-muted">·</span>
          <span className="flex items-center gap-1"><Flame className="w-3 h-3" /> {totalStrain.toFixed(1)} strain</span>
          <span className="text-text-muted">·</span>
          <span>{totalCal.toLocaleString()} Cal</span>
          <span className="text-text-muted">·</span>
          <span className="flex items-center gap-1"><Timer className="w-3 h-3" /> {formatDuration(totalDurationMs)}</span>
        </div>
        {hasFilters && (
          <button
            onClick={() => {
              setSelectedSports(new Set());
              setSelectedEffort(new Set());
              setDurationFilter("any");
            }}
            className="flex items-center gap-1 text-[11px] text-text-muted hover:text-text-secondary transition-colors"
          >
            <X className="w-3 h-3" />
            Clear filters
          </button>
        )}
      </div>

      {/* Workout grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
        {filtered.map((w, i) => (
          <div key={i} onClick={() => setDetailWorkout(w)} className="cursor-pointer">
            <WorkoutCard
              sportName={w.sport_name || "Activity"}
              strain={Number(w.strain || 0)}
              kilojoule={Number(w.kilojoule || 0)}
              startTime={w.start_time}
              endTime={w.end_time}
              averageHeartRate={w.average_heart_rate ? Number(w.average_heart_rate) : undefined}
              maxHeartRate={w.max_heart_rate ? Number(w.max_heart_rate) : undefined}
              zones={[
                Number(w.zone_zero_milli || 0),
                Number(w.zone_one_milli || 0),
                Number(w.zone_two_milli || 0),
                Number(w.zone_three_milli || 0),
                Number(w.zone_four_milli || 0),
                Number(w.zone_five_milli || 0),
              ]}
            />
          </div>
        ))}
      </div>

      {filtered.length === 0 && (
        <div className="glass-card p-12 flex flex-col items-center justify-center text-center">
          <Dumbbell className="w-10 h-10 text-text-muted mb-3" />
          <h3 className="text-sm font-semibold text-text-primary mb-1">No matching workouts</h3>
          <p className="text-xs text-text-tertiary">Try adjusting your filters</p>
        </div>
      )}

      {detailWorkout && (
        <WorkoutDetail workout={detailWorkout} onClose={() => setDetailWorkout(null)} />
      )}
    </div>
  );
}
