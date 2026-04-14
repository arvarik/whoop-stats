import { client } from "@/lib/api/client";
import { SleepPanels } from "@/components/sleep-panels";
import { SleepStagesBar } from "@/components/sleep-stages-bar";
import { TrendChartWithToggle } from "@/components/trend-chart";
import { Moon } from "lucide-react";
import { formatDuration, formatFullDate } from "@/lib/format";
import { computeAvg } from "@/lib/stats";
import type { ApiRecord } from "@/lib/types";

export const dynamic = "force-dynamic";

export default async function SleepPage() {
  const sleepsRes = await client.GET("/api/v1/sleeps", {
    params: { query: { cursor: new Date().toISOString() } },
  });

  const sleeps = (sleepsRes.data as ApiRecord[]) || [];
  const latest = sleeps[0];

  const sleepPerf = latest?.performance_score ? Math.round(Number(latest.performance_score)) : null;
  const lightMs = Number(latest?.total_light_sleep_time_milli || 0);
  const remMs = Number(latest?.total_rem_sleep_time_milli || 0);
  const deepMs = Number(latest?.total_slow_wave_sleep_time_milli || 0);
  const awakeMs = Number(latest?.total_awake_time_milli || 0);
  const noDataMs = Number(latest?.total_no_data_time_milli || 0);
  const totalSleepMs = lightMs + remMs + deepMs;
  const totalInBedMs = Number(latest?.total_in_bed_time_milli || 0);
  const efficiency = latest?.sleep_efficiency_percentage ? Number(latest.sleep_efficiency_percentage) : null;
  const consistency = latest?.sleep_consistency_percentage ? Number(latest.sleep_consistency_percentage) : null;
  const respRate = latest?.respiratory_rate ? Number(latest.respiratory_rate) : null;
  const sleepDebtMs = latest?.sleep_debt_milli ? Number(latest.sleep_debt_milli) : null;
  const disturbances = latest?.disturbance_count ? Number(latest.disturbance_count) : null;
  const sleepCycles = latest?.sleep_cycle_count ? Number(latest.sleep_cycle_count) : null;
  const baselineMs = latest?.baseline_milli ? Number(latest.baseline_milli) : null;
  const needFromStrainMs = latest?.need_from_recent_strain_milli ? Number(latest.need_from_recent_strain_milli) : null;
  const needFromNapMs = latest?.need_from_recent_nap_milli ? Number(latest.need_from_recent_nap_milli) : null;
  let napCount = 0;
  const perfValues: number[] = [];
  const effValues: number[] = [];
  const durations: number[] = [];
  const deepPcts: number[] = [];
  const remPcts: number[] = [];
  const perfTrend: { date: string; value: number }[] = [];
  const durationTrend: { date: string; value: number }[] = [];

  for (const s of sleeps) {
    if (s.nap) napCount++;

    const l = Number(s.total_light_sleep_time_milli || 0);
    const r = Number(s.total_rem_sleep_time_milli || 0);
    const d = Number(s.total_slow_wave_sleep_time_milli || 0);
    const total = l + r + d;

    if (s.performance_score) {
      const perf = Number(s.performance_score);
      perfValues.push(perf);
      perfTrend.push({ date: s.start_time as string, value: perf });
    }

    if (s.sleep_efficiency_percentage) {
      effValues.push(Number(s.sleep_efficiency_percentage));
    }

    if (total > 0) {
      durations.push(total);
      deepPcts.push((d / total) * 100);
      remPcts.push((r / total) * 100);
      durationTrend.push({ date: s.start_time as string, value: total / (1000 * 60 * 60) });
    }
  }

  // Reverse trends to show chronological order
  perfTrend.reverse();
  durationTrend.reverse();

  // Compute averages
  const avg7dPerf = computeAvg(perfValues.slice(0, 7));
  const avg30dPerf = computeAvg(perfValues);
  const avg7dEfficiency = computeAvg(effValues.slice(0, 7));
  const avg30dEfficiency = computeAvg(effValues);
  const avgDurationMs = computeAvg(durations);
  const avgDeepPct = computeAvg(deepPcts);
  const avgRemPct = computeAvg(remPcts);

  // Day-over-day
  const prevPerf = sleeps[1]?.performance_score ? Math.round(Number(sleeps[1].performance_score)) : null;
  const perfDelta = sleepPerf && prevPerf ? sleepPerf - prevPerf : null;

  const panelData = {
    sleepPerf, efficiency, consistency, respRate,
    totalSleepMs, totalInBedMs, sleepDebtMs, disturbances,
    sleepCycles, baselineMs, needFromStrainMs, needFromNapMs, napCount,
    lightMs, remMs, deepMs, awakeMs, noDataMs,
    avg7dPerf, avg30dPerf, avg7dEfficiency, avg30dEfficiency,
    avgDurationMs, avgDeepPct, avgRemPct, perfDelta,
  };

  return (
    <div className="px-4 md:px-8 lg:px-10 py-6 md:py-8 max-w-7xl mx-auto space-y-6">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight text-text-primary">Sleep</h1>
        <p className="text-sm text-text-tertiary mt-0.5">Analyze your sleep quality and patterns</p>
      </header>

      {/* Clickable hero stats */}
      <SleepPanels data={panelData} />

      {/* Sleep stages for last night */}
      {totalSleepMs > 0 && (
        <div className="glass-card p-5">
          <h3 className="text-sm font-semibold text-text-primary mb-3">Last Night&apos;s Stages</h3>
          <SleepStagesBar light={lightMs} rem={remMs} deep={deepMs} awake={awakeMs} />
          <div className="flex gap-4 mt-4 text-xs text-text-tertiary">
            {sleepDebtMs != null && sleepDebtMs > 0 && (
              <span>Sleep debt: {formatDuration(Math.abs(sleepDebtMs))}</span>
            )}
            {disturbances != null && (
              <span>{disturbances} disturbance{disturbances !== 1 ? "s" : ""}</span>
            )}
            {sleepCycles != null && (
              <span>{sleepCycles} sleep cycle{sleepCycles !== 1 ? "s" : ""}</span>
            )}
          </div>
        </div>
      )}

      {/* Trends */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <div className="glass-card p-5">
          <h3 className="text-sm font-semibold text-text-primary mb-1">Sleep Performance</h3>
          <p className="text-xs text-text-tertiary mb-3">Percentage score</p>
          <TrendChartWithToggle data={perfTrend} color="var(--color-sleep)" gradientId="sleepPerfGrad" unit="%" domain={[0, 100]} height={200} />
        </div>
        <div className="glass-card p-5">
          <h3 className="text-sm font-semibold text-text-primary mb-1">Sleep Duration</h3>
          <p className="text-xs text-text-tertiary mb-3">Hours of actual sleep</p>
          <TrendChartWithToggle data={durationTrend} color="#38bdf8" gradientId="sleepDurGrad" unit="h" height={200} />
        </div>
      </div>

      {/* Sleep history */}
      {sleeps.length > 0 && (
        <div className="glass-card p-5">
          <h3 className="text-sm font-semibold text-text-primary mb-4">Sleep History</h3>
          <div className="space-y-1">
            {sleeps.slice(0, 14).map((s: ApiRecord, i: number) => {
              const perf = s.performance_score ? Math.round(Number(s.performance_score)) : null;
              const dur = Number(s.total_light_sleep_time_milli || 0) + Number(s.total_rem_sleep_time_milli || 0) + Number(s.total_slow_wave_sleep_time_milli || 0);
              return (
                <div key={i} className="flex items-center gap-3 px-3 py-2.5 rounded-lg hover:bg-surface-1/50 transition-colors">
                  <Moon className="w-3.5 h-3.5 text-sleep" />
                  <span className="text-sm text-text-secondary flex-1">{formatFullDate(s.start_time)}</span>
                  <span className="text-sm font-medium text-text-primary">{perf ? `${perf}%` : "--"}</span>
                  <span className="text-xs text-text-muted w-16 text-right">{dur > 0 ? formatDuration(dur) : "--"}</span>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
