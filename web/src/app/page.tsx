import { Suspense } from "react";
import { client } from "@/lib/api/client";
import { MetricCard } from "@/components/metric-card";
import { StrainRecoveryChart } from "@/components/strain-recovery-chart";
import { SleepStagesBar } from "@/components/sleep-stages-bar";
import { RecentWorkouts } from "@/components/recent-workouts";
import { SyncButton } from "@/components/SyncButton";
import {
  Activity,
  Moon,
  HeartPulse,
  Flame,
  TrendingUp,
  Dumbbell,
} from "lucide-react";
import {
  formatDuration,
  formatCalories,
  getRecoveryColor,
  getRecoveryLabel,
} from "@/lib/format";
import type { ApiRecord } from "@/lib/types";

export const dynamic = "force-dynamic";

export default async function DashboardPage() {
  const [profileRes, cyclesRes, sleepsRes, workoutsRes, recoveriesRes] =
    await Promise.all([
      client.GET("/api/v1/user/profile"),
      client.GET("/api/v1/cycles", {
        params: { query: { cursor: new Date().toISOString() } },
      }),
      client.GET("/api/v1/sleeps", {
        params: { query: { cursor: new Date().toISOString() } },
      }),
      client.GET("/api/v1/workouts", {
        params: { query: { cursor: new Date().toISOString() } },
      }),
      client.GET("/api/v1/recoveries", {
        params: { query: { cursor: new Date().toISOString() } },
      }),
    ]);

  // Latest data
  const latestCycle = (cyclesRes.data as ApiRecord[])?.[0];
  const latestSleep = (sleepsRes.data as ApiRecord[])?.[0];
  const latestRecovery = (recoveriesRes.data as ApiRecord[])?.[0];
  const workouts = ((workoutsRes.data as ApiRecord[]) || []).slice(0, 4);
  const recoveries = ((recoveriesRes.data as ApiRecord[]) || []).slice(0, 7);

  // Strain
  const currentStrain = latestCycle?.strain
    ? Number(latestCycle.strain).toFixed(1)
    : "--";
  const kilojoules = latestCycle?.kilojoule
    ? formatCalories(Number(latestCycle.kilojoule))
    : "No data";

  // Recovery
  const recoveryScore = latestRecovery?.recovery_score
    ? Math.round(Number(latestRecovery.recovery_score))
    : null;
  const recoveryColorName = recoveryScore ? getRecoveryColor(recoveryScore) : "none";
  const recoverySubtext = recoveryScore ? getRecoveryLabel(recoveryScore) : "--";
  const hrv = latestRecovery?.hrv_rmssd_milli
    ? Number(latestRecovery.hrv_rmssd_milli).toFixed(0)
    : null;
  const rhr = latestRecovery?.resting_heart_rate
    ? Number(latestRecovery.resting_heart_rate).toFixed(0)
    : null;

  // Sleep
  const sleepPerf = latestSleep?.performance_score
    ? Math.round(Number(latestSleep.performance_score))
    : null;
  const lightMs = Number(latestSleep?.total_light_sleep_time_milli || 0);
  const remMs = Number(latestSleep?.total_rem_sleep_time_milli || 0);
  const deepMs = Number(latestSleep?.total_slow_wave_sleep_time_milli || 0);
  const awakeMs = Number(latestSleep?.total_awake_time_milli || 0);
  const totalSleepMs = lightMs + remMs + deepMs;

  // Build 30-day chart from raw cycles + recoveries + workouts
  const allCycles = (cyclesRes.data as ApiRecord[]) || [];
  const allRecoveries = (recoveriesRes.data as ApiRecord[]) || [];
  const allWorkouts = (workoutsRes.data as ApiRecord[]) || [];

  // Build a set of dates that had workouts
  const workoutDates = new Map<string, string>();
  allWorkouts.forEach((w: ApiRecord) => {
    const dateKey = new Date(w.start_time).toISOString().slice(0, 10);
    workoutDates.set(dateKey, w.sport_name || "Workout");
  });

  // Build recovery lookup by date
  const recoveryByDate = new Map<string, number>();
  allRecoveries.forEach((r: ApiRecord) => {
    if (r.recovery_score) {
      const dateKey = new Date(r.start_time).toISOString().slice(0, 10);
      recoveryByDate.set(dateKey, Number(r.recovery_score));
    }
  });

  // Build chart data from cycles (one per day)
  const chartData = allCycles
    .filter((c: ApiRecord) => c.strain || c.kilojoule)
    .map((c: ApiRecord) => {
      const dateKey = new Date(c.start_time).toISOString().slice(0, 10);
      return {
        date: c.start_time as string,
        strain: c.strain ? Number(c.strain) : null,
        recovery: recoveryByDate.get(dateKey) ?? null,
        hasWorkout: workoutDates.has(dateKey),
        workoutName: workoutDates.get(dateKey),
      };
    })
    .reverse();


  return (
    <div className="px-4 md:px-8 lg:px-10 py-6 md:py-8 max-w-7xl mx-auto space-y-6">
      {/* Header */}
      <header className="flex flex-col sm:flex-row sm:items-end justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight text-text-primary">
            Overview
          </h1>
          <p className="text-sm text-text-tertiary mt-0.5">
            Welcome back
            {profileRes.data?.first_name ? `, ${profileRes.data.first_name}` : ""}.
          </p>
        </div>
        <SyncButton />
      </header>

      {/* Hero metrics */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
        <MetricCard
          title="Day Strain"
          value={currentStrain}
          subtitle={
            <span className="flex items-center gap-1">
              <Flame className="w-3 h-3 text-orange-400" />
              {kilojoules}
            </span>
          }
          icon={<Activity className="w-4 h-4" />}
          accentColor="blue"
        />

        <MetricCard
          title="Recovery"
          value={recoveryScore ? `${recoveryScore}%` : "--%"}
          subtitle={
            <div className="space-y-1">
              <div>{recoverySubtext}</div>
              {(hrv || rhr) && (
                <div className="flex gap-3 text-xs text-text-muted">
                  {hrv && <span>HRV {hrv} ms</span>}
                  {rhr && <span>RHR {rhr} bpm</span>}
                </div>
              )}
            </div>
          }
          icon={<HeartPulse className="w-4 h-4" />}
          accentColor={recoveryColorName === "none" ? "none" : recoveryColorName}
        />

        <MetricCard
          title="Sleep Performance"
          value={sleepPerf ? `${sleepPerf}%` : "--%"}
          subtitle={totalSleepMs > 0 ? formatDuration(totalSleepMs) + " actual sleep" : "No data"}
          icon={<Moon className="w-4 h-4" />}
          accentColor="violet"
        >
          {totalSleepMs > 0 && (
            <SleepStagesBar light={lightMs} rem={remMs} deep={deepMs} awake={awakeMs} />
          )}
        </MetricCard>
      </div>

      {/* Weekly recovery strip */}
      {recoveries.length > 0 && (
        <div className="glass-card p-4">
          <h3 className="text-xs font-medium uppercase tracking-wider text-text-tertiary mb-3">
            7-Day Recovery
          </h3>
          <div className="flex items-center gap-1.5">
            {recoveries
              .slice()
              .reverse()
              .map((rec: ApiRecord, i: number) => {
                const score = rec.recovery_score
                  ? Math.round(Number(rec.recovery_score))
                  : null;
                const color = score
                  ? score >= 66
                    ? "bg-emerald-500"
                    : score >= 34
                      ? "bg-amber-500"
                      : "bg-rose-500"
                  : "bg-surface-2";
                return (
                  <div key={i} className="flex-1 flex flex-col items-center gap-1.5">
                    <div
                      className={`w-full h-8 rounded-md ${color} transition-all`}
                      style={{ opacity: score ? 0.3 + (score / 100) * 0.7 : 0.2 }}
                      title={score ? `${score}%` : "No data"}
                    />
                    <span className="text-[10px] text-text-muted">
                      {score ? `${score}%` : "--"}
                    </span>
                  </div>
                );
              })}
          </div>
        </div>
      )}

      {/* Trend chart */}
      <div className="glass-card p-5 md:p-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h2 className="text-sm font-semibold text-text-primary">30-Day Trends</h2>
            <p className="text-xs text-text-tertiary mt-0.5">Strain vs Recovery</p>
          </div>
          <TrendingUp className="w-4 h-4 text-text-muted" />
        </div>
        <Suspense
          fallback={
            <div className="w-full h-[300px] animate-pulse bg-surface-1/30 rounded-xl" />
          }
        >
          <StrainRecoveryChart data={chartData} />
        </Suspense>
      </div>

      {/* Recent workouts */}
      {workouts.length > 0 && (
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <h2 className="text-sm font-semibold text-text-primary flex items-center gap-2">
              <Dumbbell className="w-4 h-4 text-text-tertiary" />
              Recent Workouts
            </h2>
          </div>
          <RecentWorkouts workouts={workouts} />
        </div>
      )}
    </div>
  );
}
