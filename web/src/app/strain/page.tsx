import { client } from "@/lib/api/client";
import { StrainPanels } from "@/components/strain-panels";
import { TrendChartWithToggle } from "@/components/trend-chart";
import { Flame } from "lucide-react";
import { formatCalories, formatFullDate, kjToCal } from "@/lib/format";
import { computeAvg } from "@/lib/stats";

export const dynamic = "force-dynamic";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type ApiRecord = Record<string, any>;

export default async function StrainPage() {
  const [cyclesRes, workoutsRes] = await Promise.all([
    client.GET("/api/v1/cycles", {
      params: { query: { cursor: new Date().toISOString() } },
    }),
    client.GET("/api/v1/workouts", {
      params: { query: { cursor: new Date().toISOString() } },
    }),
  ]);

  const cycles = (cyclesRes.data as ApiRecord[]) || [];
  const workouts = (workoutsRes.data as ApiRecord[]) || [];
  const latest = cycles[0];

  const strain = latest?.strain ? Number(latest.strain) : null;
  const kj = latest?.kilojoule ? Number(latest.kilojoule) : null;
  const avgHR = latest?.average_heart_rate ? Number(latest.average_heart_rate) : null;
  const maxHR = latest?.max_heart_rate ? Number(latest.max_heart_rate) : null;

  // 7-day
  const weekCycles = cycles.slice(0, 7);
  const weekStrain = weekCycles.reduce((acc: number, c: ApiRecord) => acc + (Number(c.strain) || 0), 0);
  const weekKJ = weekCycles.reduce((acc: number, c: ApiRecord) => acc + (Number(c.kilojoule) || 0), 0);
  const weekAvgDailyStrain = computeAvg(weekCycles.map((c: ApiRecord) => Number(c.strain || 0)));

  // All-time derived
  const allStrains = cycles.filter((c: ApiRecord) => c.strain).map((c: ApiRecord) => Number(c.strain));
  const avgDailyStrain = computeAvg(allStrains);
  const peakStrain = allStrains.length ? Math.max(...allStrains) : null;
  const totalCal = cycles.reduce((acc: number, c: ApiRecord) => acc + kjToCal(Number(c.kilojoule || 0)), 0);
  const avgDailyCal = computeAvg(cycles.map((c: ApiRecord) => kjToCal(Number(c.kilojoule || 0))));
  const highStrainDays = allStrains.filter(s => s >= 14).length;

  // Day-over-day
  const prevStrain = cycles[1]?.strain ? Number(cycles[1].strain) : null;
  const strainDelta = strain && prevStrain ? strain - prevStrain : null;

  // Workout stats
  const workoutStrains = workouts.filter((w: ApiRecord) => w.strain).map((w: ApiRecord) => Number(w.strain));
  const avgWorkoutStrain = computeAvg(workoutStrains);
  const workoutDurations = workouts.map((w: ApiRecord) => {
    if (!w.end_time) return 0;
    return new Date(w.end_time).getTime() - new Date(w.start_time).getTime();
  }).filter(d => d > 0);
  const avgWorkoutDurationMs = computeAvg(workoutDurations);
  const totalWorkoutDurationMs = workoutDurations.reduce((a, b) => a + b, 0);

  // Sport breakdown
  const sportMap: Record<string, { count: number; totalStrain: number; totalKJ: number }> = {};
  workouts.forEach((w: ApiRecord) => {
    const sport = (w.sport_name || "activity").toLowerCase();
    if (!sportMap[sport]) sportMap[sport] = { count: 0, totalStrain: 0, totalKJ: 0 };
    sportMap[sport].count++;
    sportMap[sport].totalStrain += Number(w.strain || 0);
    sportMap[sport].totalKJ += Number(w.kilojoule || 0);
  });
  const sportBreakdown = Object.entries(sportMap)
    .map(([sport, data]) => ({
      sport,
      count: data.count,
      avgStrain: data.totalStrain / data.count,
      totalCal: Math.round(kjToCal(data.totalKJ)),
    }))
    .sort((a, b) => b.count - a.count);

  const panelData = {
    strain, kj, avgHR, maxHR,
    weekStrain, weekKJ, weekAvgDailyStrain,
    avgDailyStrain, peakStrain, totalCal,
    totalDays: cycles.length, workoutCount: workouts.length,
    highStrainDays, avgDailyCal,
    avgWorkoutStrain, avgWorkoutDurationMs, totalWorkoutDurationMs,
    sportBreakdown, strainDelta,
  };

  // Trends
  const dailyStrainTrend = cycles
    .filter((c: ApiRecord) => c.strain)
    .map((c: ApiRecord) => ({ date: c.start_time as string, value: Number(c.strain) }))
    .reverse();
  const calorieTrend = cycles
    .filter((c: ApiRecord) => c.kilojoule)
    .map((c: ApiRecord) => ({ date: c.start_time as string, value: kjToCal(Number(c.kilojoule)) }))
    .reverse();

  return (
    <div className="px-4 md:px-8 lg:px-10 py-6 md:py-8 max-w-7xl mx-auto space-y-6">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight text-text-primary">Strain</h1>
        <p className="text-sm text-text-tertiary mt-0.5">Monitor your daily cardiovascular load</p>
      </header>

      {/* Clickable panels */}
      <StrainPanels data={panelData} />

      {/* Trend charts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <div className="glass-card p-5">
          <h3 className="text-sm font-semibold text-text-primary mb-1">Daily Strain</h3>
          <p className="text-xs text-text-tertiary mb-3">Per-cycle strain score</p>
          <TrendChartWithToggle data={dailyStrainTrend} color="var(--color-strain)" gradientId="strainDailyGrad" domain={[0, 21]} height={220} />
        </div>
        <div className="glass-card p-5">
          <h3 className="text-sm font-semibold text-text-primary mb-1">Daily Calories</h3>
          <p className="text-xs text-text-tertiary mb-3">Converted from kilojoules</p>
          <TrendChartWithToggle data={calorieTrend} color="#f97316" gradientId="calTrendGrad" unit=" Cal" height={220} />
        </div>
      </div>

      {/* Cycle history */}
      {cycles.length > 0 && (
        <div className="glass-card p-5">
          <h3 className="text-sm font-semibold text-text-primary mb-4">Cycle History</h3>
          <div className="space-y-1">
            {cycles.slice(0, 14).map((c: ApiRecord, i: number) => {
              const s = c.strain ? Number(c.strain) : null;
              return (
                <div key={i} className="flex items-center gap-3 px-3 py-2.5 rounded-lg hover:bg-surface-1/50 transition-colors">
                  <Flame className="w-3.5 h-3.5 text-strain" />
                  <span className="text-sm text-text-secondary flex-1">{formatFullDate(c.start_time)}</span>
                  <span className="text-sm font-medium text-text-primary">{s ? s.toFixed(1) : "--"}</span>
                  <span className="text-xs text-text-muted w-20 text-right">{c.kilojoule ? formatCalories(Number(c.kilojoule)) : "--"}</span>
                  <span className="text-xs text-text-muted w-16 text-right">{c.average_heart_rate ? `${c.average_heart_rate} bpm` : "--"}</span>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
