import { client } from "@/lib/api/client";
import { RecoveryGauge } from "@/components/recovery-gauge";
import { TrendChartWithToggle } from "@/components/trend-chart";
import { getRecoveryLabel, formatFullDate } from "@/lib/format";
import { RecoveryPanels } from "@/components/recovery-panels";
import { computeAvg, computeStdDev } from "@/lib/stats";

export const dynamic = "force-dynamic";


// eslint-disable-next-line @typescript-eslint/no-explicit-any
type ApiRecord = Record<string, any>;


export default async function RecoveryPage() {
  const recoveriesRes = await client.GET("/api/v1/recoveries", {
    params: { query: { cursor: new Date().toISOString() } },
  });

  const recoveries = ((recoveriesRes.data as ApiRecord[]) || []);
  const latest = recoveries[0];

  const recoveryScore = latest?.recovery_score
    ? Math.round(Number(latest.recovery_score))
    : null;
  const hrv = latest?.hrv_rmssd_milli ? Number(latest.hrv_rmssd_milli) : null;
  const rhr = latest?.resting_heart_rate ? Number(latest.resting_heart_rate) : null;
  const spo2 = latest?.spo2_percentage ? Number(latest.spo2_percentage) : null;
  const skinTemp = latest?.skin_temp_celsius ? Number(latest.skin_temp_celsius) : null;

  // Compute arrays for derived stats
  const hrvValues = recoveries.filter((r: ApiRecord) => r.hrv_rmssd_milli).map((r: ApiRecord) => Number(r.hrv_rmssd_milli));
  const rhrValues = recoveries.filter((r: ApiRecord) => r.resting_heart_rate).map((r: ApiRecord) => Number(r.resting_heart_rate));
  const recoveryScores = recoveries.filter((r: ApiRecord) => r.recovery_score).map((r: ApiRecord) => Number(r.recovery_score));
  const spo2Values = recoveries.filter((r: ApiRecord) => r.spo2_percentage).map((r: ApiRecord) => Number(r.spo2_percentage));
  const skinTempValues = recoveries.filter((r: ApiRecord) => r.skin_temp_celsius).map((r: ApiRecord) => Number(r.skin_temp_celsius));

  // Averages
  const avg7dRecovery = computeAvg(recoveryScores.slice(0, 7));
  const avg30dRecovery = computeAvg(recoveryScores);
  const avg7dHRV = computeAvg(hrvValues.slice(0, 7));
  const avg30dHRV = computeAvg(hrvValues);
  const avg7dRHR = computeAvg(rhrValues.slice(0, 7));
  const avg30dRHR = computeAvg(rhrValues);

  // Standard deviations (variability)
  const hrvStdDev = computeStdDev(hrvValues.slice(0, 30));
  const rhrStdDev = computeStdDev(rhrValues.slice(0, 30));

  // Min/Max ranges
  const hrvMin = hrvValues.length ? Math.min(...hrvValues) : null;
  const hrvMax = hrvValues.length ? Math.max(...hrvValues) : null;
  const rhrMin = rhrValues.length ? Math.min(...rhrValues) : null;
  const rhrMax = rhrValues.length ? Math.max(...rhrValues) : null;

  // SpO2 stats
  const avgSpo2 = computeAvg(spo2Values);
  const minSpo2 = spo2Values.length ? Math.min(...spo2Values) : null;

  // Skin temp stats
  const avgSkinTemp = computeAvg(skinTempValues);
  const skinTempStdDev = computeStdDev(skinTempValues);
  const skinTempDeviation = skinTemp && avgSkinTemp ? skinTemp - avgSkinTemp : null;

  // Day-over-day deltas
  const prevRecovery = recoveries[1]?.recovery_score ? Number(recoveries[1].recovery_score) : null;
  const recoveryDelta = recoveryScore && prevRecovery ? recoveryScore - Math.round(prevRecovery) : null;
  const prevHRV = hrvValues.length > 1 ? hrvValues[1] : null;
  const hrvDelta = hrv && prevHRV ? hrv - prevHRV : null;
  const prevRHR = rhrValues.length > 1 ? rhrValues[1] : null;
  const rhrDelta = rhr && prevRHR ? rhr - prevRHR : null;

  // Distribution: how many days in green/yellow/red
  const greenDays = recoveryScores.filter(s => s >= 66).length;
  const yellowDays = recoveryScores.filter(s => s >= 34 && s < 66).length;
  const redDays = recoveryScores.filter(s => s < 34).length;

  // Build trend data
  const hrvTrend = recoveries
    .filter((r: ApiRecord) => r.hrv_rmssd_milli)
    .map((r: ApiRecord) => ({ date: r.start_time as string, value: Number(r.hrv_rmssd_milli) }))
    .reverse();
  const rhrTrend = recoveries
    .filter((r: ApiRecord) => r.resting_heart_rate)
    .map((r: ApiRecord) => ({ date: r.start_time as string, value: Number(r.resting_heart_rate) }))
    .reverse();
  const recoveryTrend = recoveries
    .filter((r: ApiRecord) => r.recovery_score)
    .map((r: ApiRecord) => ({ date: r.start_time as string, value: Number(r.recovery_score) }))
    .reverse();

  // Package all data for the client panels
  const panelData = {
    hrv, rhr, spo2, skinTemp, recoveryScore,
    avg7dHRV, avg30dHRV, avg7dRHR, avg30dRHR,
    hrvStdDev, rhrStdDev,
    hrvMin, hrvMax, rhrMin, rhrMax,
    avgSpo2, minSpo2,
    avgSkinTemp, skinTempStdDev, skinTempDeviation,
    recoveryDelta, hrvDelta, rhrDelta,
    avg7dRecovery, avg30dRecovery,
    greenDays, yellowDays, redDays,
    totalDays: recoveryScores.length,
  };

  return (
    <div className="px-4 md:px-8 lg:px-10 py-6 md:py-8 max-w-7xl mx-auto space-y-6">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight text-text-primary">Recovery</h1>
        <p className="text-sm text-text-tertiary mt-0.5">Track your body&apos;s readiness to perform</p>
      </header>

      {/* Top section: Gauge + Stats */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <div className="glass-card p-6 flex flex-col items-center justify-center lg:col-span-1">
          <RecoveryGauge score={recoveryScore} size={200} />
          {recoveryScore && (
            <p className="text-sm text-text-secondary mt-2">{getRecoveryLabel(recoveryScore)}</p>
          )}
          {recoveryDelta != null && (
            <p className="text-xs text-text-muted mt-1">
              {recoveryDelta > 0 ? "+" : ""}{recoveryDelta}% from yesterday
            </p>
          )}
        </div>

        {/* Clickable vitals */}
        <div className="lg:col-span-2">
          <RecoveryPanels data={panelData} />
        </div>
      </div>

      {/* Recovery Distribution — own row */}
      <div className="glass-card p-5">
        <h3 className="text-xs font-medium uppercase tracking-wider text-text-tertiary mb-3">
          Recovery Distribution ({panelData.totalDays} days)
        </h3>
        <div className="flex items-center gap-1.5 h-4 rounded-full overflow-hidden">
          {greenDays > 0 && (
            <div className="h-full bg-emerald-500 rounded-full transition-all" style={{ flex: greenDays }} />
          )}
          {yellowDays > 0 && (
            <div className="h-full bg-amber-500 rounded-full transition-all" style={{ flex: yellowDays }} />
          )}
          {redDays > 0 && (
            <div className="h-full bg-rose-500 rounded-full transition-all" style={{ flex: redDays }} />
          )}
        </div>
        <div className="flex justify-between mt-3 text-xs">
          <span className="text-emerald-400 font-medium">{greenDays} green ({recoveryScores.length ? Math.round(greenDays / recoveryScores.length * 100) : 0}%)</span>
          <span className="text-amber-400 font-medium">{yellowDays} yellow ({recoveryScores.length ? Math.round(yellowDays / recoveryScores.length * 100) : 0}%)</span>
          <span className="text-rose-400 font-medium">{redDays} red ({recoveryScores.length ? Math.round(redDays / recoveryScores.length * 100) : 0}%)</span>
        </div>
      </div>

      {/* Trend charts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <div className="glass-card p-5">
          <h3 className="text-sm font-semibold text-text-primary mb-1">HRV Trend</h3>
          <p className="text-xs text-text-tertiary mb-3">RMSSD in milliseconds</p>
          <TrendChartWithToggle data={hrvTrend} color="var(--color-recovery-green)" gradientId="hrvGrad" unit=" ms" height={200} />
        </div>
        <div className="glass-card p-5">
          <h3 className="text-sm font-semibold text-text-primary mb-1">Resting Heart Rate</h3>
          <p className="text-xs text-text-tertiary mb-3">Beats per minute (lower is better)</p>
          <TrendChartWithToggle data={rhrTrend} color="var(--color-strain)" gradientId="rhrGrad" unit=" bpm" height={200} />
        </div>
      </div>

      {/* Recovery score trend */}
      <div className="glass-card p-5">
        <h3 className="text-sm font-semibold text-text-primary mb-1">Recovery Score</h3>
        <p className="text-xs text-text-tertiary mb-3">Daily recovery percentage</p>
        <TrendChartWithToggle data={recoveryTrend} color="var(--color-recovery-green)" gradientId="recGrad" unit="%" domain={[0, 100]} height={220} />
      </div>

      {/* Recovery history list */}
      {recoveries.length > 0 && (
        <div className="glass-card p-5">
          <h3 className="text-sm font-semibold text-text-primary mb-4">Recovery History</h3>
          <div className="space-y-1">
            {recoveries.slice(0, 14).map((rec: ApiRecord, i: number) => {
              const score = rec.recovery_score ? Math.round(Number(rec.recovery_score)) : null;
              const dotColor = score
                ? score >= 66 ? "bg-emerald-500" : score >= 34 ? "bg-amber-500" : "bg-rose-500"
                : "bg-surface-3";
              return (
                <div key={i} className="flex items-center gap-3 px-3 py-2.5 rounded-lg hover:bg-surface-1/50 transition-colors">
                  <div className={`w-2.5 h-2.5 rounded-full ${dotColor}`} />
                  <span className="text-sm text-text-secondary flex-1">{formatFullDate(rec.start_time)}</span>
                  <span className="text-sm font-medium text-text-primary">{score ? `${score}%` : "--"}</span>
                  <span className="text-xs text-text-muted w-16 text-right">{rec.hrv_rmssd_milli ? `${Number(rec.hrv_rmssd_milli).toFixed(0)} ms` : "--"}</span>
                  <span className="text-xs text-text-muted w-16 text-right">{rec.resting_heart_rate ? `${Number(rec.resting_heart_rate).toFixed(0)} bpm` : "--"}</span>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
