"use client";

import { MetricCard } from "@/components/metric-card";
import { DetailPopup, DetailRow, useDetailPopup } from "@/components/detail-popup";
import { BedDouble, Clock, Brain, Moon } from "lucide-react";
import { formatDuration } from "@/lib/format";

interface SleepPanelData {
  sleepPerf: number | null;
  efficiency: number | null;
  consistency: number | null;
  respRate: number | null;
  totalSleepMs: number;
  totalInBedMs: number;
  sleepDebtMs: number | null;
  disturbances: number | null;
  sleepCycles: number | null;
  baselineMs: number | null;
  needFromStrainMs: number | null;
  needFromNapMs: number | null;
  napCount: number;
  lightMs: number;
  remMs: number;
  deepMs: number;
  awakeMs: number;
  noDataMs: number;
  // Derived averages
  avg7dPerf: number | null;
  avg30dPerf: number | null;
  avg7dEfficiency: number | null;
  avg30dEfficiency: number | null;
  avgDurationMs: number | null;
  avgDeepPct: number | null;
  avgRemPct: number | null;
  perfDelta: number | null;
}

function fmtDur(ms: number): string {
  return ms > 0 ? formatDuration(ms) : "--";
}

export function SleepPanels({ data: d }: { data: SleepPanelData }) {
  const { popup, open, close } = useDetailPopup();

  const deepPct = d.totalSleepMs > 0 ? (d.deepMs / d.totalSleepMs * 100) : 0;
  const remPct = d.totalSleepMs > 0 ? (d.remMs / d.totalSleepMs * 100) : 0;
  const lightPct = d.totalSleepMs > 0 ? (d.lightMs / d.totalSleepMs * 100) : 0;

  // Sleep need: baseline + strain need - nap credit
  const sleepNeedMs = (d.baselineMs || 0) + (d.needFromStrainMs || 0) - (d.needFromNapMs || 0);
  const sleepDebt = d.sleepDebtMs ? d.sleepDebtMs : null;
  const overUnder = sleepNeedMs > 0 && d.totalSleepMs > 0
    ? d.totalSleepMs - sleepNeedMs
    : null;

  return (
    <>
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <MetricCard
          title="Performance"
          value={d.sleepPerf ? `${d.sleepPerf}%` : "--%"}
          subtitle={d.perfDelta != null ? (
            <span className={`text-[10px] ${d.perfDelta > 0 ? "text-emerald-400" : d.perfDelta < 0 ? "text-rose-400" : "text-text-muted"}`}>
              {d.perfDelta > 0 ? "+" : ""}{d.perfDelta}% from yesterday
            </span>
          ) : d.totalSleepMs > 0 ? fmtDur(d.totalSleepMs) + " total" : undefined}
          icon={<Moon className="w-4 h-4" />}
          accentColor="violet"
          onClick={() => open("performance")}
        />
        <MetricCard
          title="Efficiency"
          value={d.efficiency ? `${d.efficiency.toFixed(0)}%` : "--"}
          subtitle="Time asleep vs in bed"
          icon={<BedDouble className="w-4 h-4" />}
          accentColor="blue"
          onClick={() => open("efficiency")}
        />
        <MetricCard
          title="Consistency"
          value={d.consistency ? `${d.consistency.toFixed(0)}%` : "--"}
          subtitle="Schedule regularity"
          icon={<Clock className="w-4 h-4" />}
          accentColor="green"
          onClick={() => open("stages")}
        />
        <MetricCard
          title="Resp Rate"
          value={d.respRate ? `${d.respRate.toFixed(1)}` : "--"}
          subtitle="Breaths per minute"
          icon={<Brain className="w-4 h-4" />}
          accentColor="yellow"
          onClick={() => open("resp")}
        />
      </div>

      {/* Performance Detail */}
      {popup === "performance" && (
        <DetailPopup title="Sleep Performance" onClose={close}>
          <p className="text-xs text-text-tertiary mb-4">
            Sleep performance measures how well your actual sleep met your body&apos;s sleep need, including recovery from recent strain.
          </p>
          <DetailRow label="Performance Score" value={d.sleepPerf ? `${d.sleepPerf}%` : "--"} />
          <DetailRow label="7-Day Avg Performance" value={d.avg7dPerf ? `${d.avg7dPerf.toFixed(0)}%` : "--"} />
          <DetailRow label="30-Day Avg Performance" value={d.avg30dPerf ? `${d.avg30dPerf.toFixed(0)}%` : "--"} />
          <DetailRow label="Total Sleep" value={fmtDur(d.totalSleepMs)} />
          <DetailRow label="Time in Bed" value={fmtDur(d.totalInBedMs)} />
          <DetailRow label="Sleep Need" value={sleepNeedMs > 0 ? fmtDur(sleepNeedMs) : "--"} hint="Baseline + strain need - nap credit" />
          {d.baselineMs && d.baselineMs > 0 && (
            <DetailRow label="Baseline Need" value={fmtDur(d.baselineMs)} hint="Your body's base sleep requirement" />
          )}
          {d.needFromStrainMs && d.needFromStrainMs > 0 && (
            <DetailRow label="Added from Strain" value={`+${fmtDur(d.needFromStrainMs)}`} hint="Extra sleep needed from recent activity" />
          )}
          {overUnder != null && (
            <div className="mt-4 p-3 rounded-lg bg-surface-1/30">
              <p className="text-xs text-text-secondary">
                {overUnder > 0
                  ? `You slept ${fmtDur(overUnder)} more than your sleep need — great recovery!`
                  : overUnder < 0
                    ? `You slept ${fmtDur(Math.abs(overUnder))} less than your sleep need.`
                    : "You met your exact sleep need."}
              </p>
            </div>
          )}
        </DetailPopup>
      )}

      {/* Efficiency Detail */}
      {popup === "efficiency" && (
        <DetailPopup title="Sleep Efficiency" onClose={close}>
          <p className="text-xs text-text-tertiary mb-4">
            Efficiency measures the percentage of time you were actually asleep while in bed. Higher is better — aim for 85%+.
          </p>
          <DetailRow label="Efficiency" value={d.efficiency ? `${d.efficiency.toFixed(1)}%` : "--"} />
          <DetailRow label="7-Day Avg" value={d.avg7dEfficiency ? `${d.avg7dEfficiency.toFixed(1)}%` : "--"} />
          <DetailRow label="30-Day Avg" value={d.avg30dEfficiency ? `${d.avg30dEfficiency.toFixed(1)}%` : "--"} />
          <DetailRow label="Time Awake in Bed" value={fmtDur(d.awakeMs)} />
          <DetailRow label="Disturbances" value={d.disturbances ?? "--"} hint="Number of times you woke up" />
          <DetailRow label="Sleep Cycles" value={d.sleepCycles ?? "--"} hint="Complete sleep cycles completed" />
          <DetailRow label="Sleep Debt" value={sleepDebt != null && sleepDebt > 0 ? fmtDur(sleepDebt) : "None"} hint="Accumulated sleep deficit" />
          {d.napCount > 0 && (
            <DetailRow label="Naps" value={d.napCount.toLocaleString()} hint="Naps in data range" />
          )}
        </DetailPopup>
      )}

      {/* Stages Detail */}
      {popup === "stages" && (
        <DetailPopup title="Sleep Stages Breakdown" onClose={close}>
          <p className="text-xs text-text-tertiary mb-4">
            Your sleep is composed of light, REM, and deep sleep stages. Deep and REM sleep are critical for physical recovery and memory consolidation.
          </p>
          <DetailRow label="Deep Sleep" value={`${fmtDur(d.deepMs)} (${deepPct.toFixed(0)}%)`} hint="Ideal: 15-20% — physical recovery & growth hormone" />
          <DetailRow label="REM Sleep" value={`${fmtDur(d.remMs)} (${remPct.toFixed(0)}%)`} hint="Ideal: 20-25% — memory, learning, emotional processing" />
          <DetailRow label="Light Sleep" value={`${fmtDur(d.lightMs)} (${lightPct.toFixed(0)}%)`} hint="Transition sleep — typically 50-60%" />
          <DetailRow label="Awake Time" value={fmtDur(d.awakeMs)} />
          {d.noDataMs > 0 && <DetailRow label="No Data" value={fmtDur(d.noDataMs)} />}
          <DetailRow label="Avg Duration" value={d.avgDurationMs ? fmtDur(d.avgDurationMs) : "--"} hint="Average across dataset" />
          {d.avgDeepPct != null && <DetailRow label="Avg Deep %" value={`${d.avgDeepPct.toFixed(0)}%`} />}
          {d.avgRemPct != null && <DetailRow label="Avg REM %" value={`${d.avgRemPct.toFixed(0)}%`} />}
          <DetailRow label="Sleep Consistency" value={d.consistency ? `${d.consistency.toFixed(0)}%` : "--"} hint="How regular your sleep schedule is" />
        </DetailPopup>
      )}

      {/* Respiratory Rate Detail */}
      {popup === "resp" && (
        <DetailPopup title="Respiratory Rate" onClose={close}>
          <p className="text-xs text-text-tertiary mb-4">
            Normal respiratory rate during sleep is 12-20 breaths/min. Changes can indicate illness, altitude effects, or training adaptations.
          </p>
          <DetailRow label="Current" value={d.respRate ? `${d.respRate.toFixed(1)} bpm` : "--"} />
          {d.respRate && (
            <div className="mt-4 p-3 rounded-lg bg-surface-1/30">
              <p className="text-xs text-text-secondary">
                {d.respRate >= 12 && d.respRate <= 20
                  ? "Your respiratory rate is within the normal range."
                  : d.respRate < 12
                    ? "Your respiratory rate is below normal — this is unusual."
                    : "Your respiratory rate is elevated — this could indicate illness or altitude."}
              </p>
            </div>
          )}
        </DetailPopup>
      )}
    </>
  );
}
