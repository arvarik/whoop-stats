"use client";

import { MetricCard } from "@/components/metric-card";
import { DetailPopup, DetailRow, useDetailPopup } from "@/components/detail-popup";
import { HeartPulse, Wind, Thermometer, Activity, TrendingUp, TrendingDown } from "lucide-react";

interface PanelData {
  hrv: number | null;
  rhr: number | null;
  spo2: number | null;
  skinTemp: number | null;
  recoveryScore: number | null;
  avg7dHRV: number | null;
  avg30dHRV: number | null;
  avg7dRHR: number | null;
  avg30dRHR: number | null;
  hrvStdDev: number | null;
  rhrStdDev: number | null;
  hrvMin: number | null;
  hrvMax: number | null;
  rhrMin: number | null;
  rhrMax: number | null;
  avgSpo2: number | null;
  minSpo2: number | null;
  avgSkinTemp: number | null;
  skinTempStdDev: number | null;
  skinTempDeviation: number | null;
  recoveryDelta: number | null;
  hrvDelta: number | null;
  rhrDelta: number | null;
  avg7dRecovery: number | null;
  avg30dRecovery: number | null;
  greenDays: number;
  yellowDays: number;
  redDays: number;
  totalDays: number;
}

function Delta({ value, unit, invertColor }: { value: number | null; unit: string; invertColor?: boolean }) {
  if (value === null) return null;
  // For RHR, down is good (invertColor=true)
  const isGood = invertColor ? value < 0 : value > 0;
  const color = Math.abs(value) < 1 ? "text-text-muted" : isGood ? "text-emerald-400" : "text-rose-400";
  const Icon = value > 0 ? TrendingUp : value < 0 ? TrendingDown : null;
  return (
    <span className={`flex items-center gap-0.5 text-[10px] ${color}`}>
      {Icon && <Icon className="w-3 h-3" />}
      {value > 0 ? "+" : ""}{value.toFixed(1)}{unit} from yesterday
    </span>
  );
}

export function RecoveryPanels({ data: d }: { data: PanelData }) {
  const { popup, open, close } = useDetailPopup();

  return (
    <>
      <div className="grid grid-cols-2 gap-3">
        <MetricCard
          title="HRV"
          value={d.hrv ? `${d.hrv.toFixed(0)} ms` : "--"}
          subtitle={<Delta value={d.hrvDelta} unit=" ms" />}
          icon={<HeartPulse className="w-4 h-4" />}
          accentColor="green"
          onClick={() => open("hrv")}
        />
        <MetricCard
          title="Resting HR"
          value={d.rhr ? `${d.rhr.toFixed(0)} bpm` : "--"}
          subtitle={<Delta value={d.rhrDelta} unit=" bpm" invertColor />}
          icon={<Activity className="w-4 h-4" />}
          accentColor="blue"
          onClick={() => open("rhr")}
        />
        <MetricCard
          title="SpO2"
          value={d.spo2 ? `${d.spo2.toFixed(1)}%` : "--"}
          subtitle="Blood Oxygen"
          icon={<Wind className="w-4 h-4" />}
          accentColor="violet"
          onClick={() => open("spo2")}
        />
        <MetricCard
          title="Skin Temp"
          value={d.skinTemp ? `${d.skinTemp.toFixed(1)}°C` : "--"}
          subtitle={d.skinTempDeviation != null ? (
            <span className={Math.abs(d.skinTempDeviation) > 0.5 ? "text-amber-400 text-[10px]" : "text-text-muted text-[10px]"}>
              {d.skinTempDeviation > 0 ? "+" : ""}{d.skinTempDeviation.toFixed(1)}°C from baseline
            </span>
          ) : "Skin Temperature"}
          icon={<Thermometer className="w-4 h-4" />}
          accentColor="yellow"
          onClick={() => open("skinTemp")}
        />
      </div>

      {/* HRV Detail */}
      {popup === "hrv" && (
        <DetailPopup title="Heart Rate Variability" onClose={close}>
          <p className="text-xs text-text-tertiary mb-4">
            HRV measures the variation in time between heartbeats. Higher values generally indicate better cardiovascular fitness and recovery.
          </p>
          <DetailRow label="Current HRV" value={d.hrv ? `${d.hrv.toFixed(1)} ms` : "--"} />
          <DetailRow label="Yesterday's change" value={d.hrvDelta != null ? `${d.hrvDelta > 0 ? "+" : ""}${d.hrvDelta.toFixed(1)} ms` : "--"} />
          <DetailRow label="7-Day Average" value={d.avg7dHRV ? `${d.avg7dHRV.toFixed(1)} ms` : "--"} />
          <DetailRow label="30-Day Average" value={d.avg30dHRV ? `${d.avg30dHRV.toFixed(1)} ms` : "--"} hint="Your baseline" />
          <DetailRow label="Variability (σ)" value={d.hrvStdDev ? `±${d.hrvStdDev.toFixed(1)} ms` : "--"} hint="Standard deviation — lower = more consistent" />
          <DetailRow label="30-Day Range" value={d.hrvMin != null && d.hrvMax != null ? `${d.hrvMin.toFixed(0)} – ${d.hrvMax.toFixed(0)} ms` : "--"} />
          {d.hrv && d.avg30dHRV && (
            <div className="mt-4 p-3 rounded-lg bg-surface-1/30">
              <p className="text-xs text-text-secondary">
                {d.hrv > d.avg30dHRV * 1.1
                  ? "Your HRV is above your 30-day baseline — great recovery!"
                  : d.hrv < d.avg30dHRV * 0.9
                    ? "Your HRV is below baseline — consider lighter training."
                    : "Your HRV is within your normal range."}
              </p>
            </div>
          )}
        </DetailPopup>
      )}

      {/* RHR Detail */}
      {popup === "rhr" && (
        <DetailPopup title="Resting Heart Rate" onClose={close}>
          <p className="text-xs text-text-tertiary mb-4">
            A lower resting heart rate generally indicates better cardiovascular fitness. An elevated RHR may signal stress, illness, or overtraining.
          </p>
          <DetailRow label="Current RHR" value={d.rhr ? `${d.rhr.toFixed(0)} bpm` : "--"} />
          <DetailRow label="Yesterday's change" value={d.rhrDelta != null ? `${d.rhrDelta > 0 ? "+" : ""}${d.rhrDelta.toFixed(1)} bpm` : "--"} />
          <DetailRow label="7-Day Average" value={d.avg7dRHR ? `${d.avg7dRHR.toFixed(1)} bpm` : "--"} />
          <DetailRow label="30-Day Average" value={d.avg30dRHR ? `${d.avg30dRHR.toFixed(1)} bpm` : "--"} hint="Your baseline" />
          <DetailRow label="Variability (σ)" value={d.rhrStdDev ? `±${d.rhrStdDev.toFixed(1)} bpm` : "--"} />
          <DetailRow label="30-Day Range" value={d.rhrMin != null && d.rhrMax != null ? `${d.rhrMin.toFixed(0)} – ${d.rhrMax.toFixed(0)} bpm` : "--"} />
          {d.rhr && d.avg30dRHR && (
            <div className="mt-4 p-3 rounded-lg bg-surface-1/30">
              <p className="text-xs text-text-secondary">
                {d.rhr > d.avg30dRHR + 3
                  ? "Your RHR is elevated — you may be stressed, dehydrated, or fighting off illness."
                  : d.rhr < d.avg30dRHR - 3
                    ? "Your RHR is lower than baseline — excellent cardiovascular recovery!"
                    : "Your RHR is within your normal range."}
              </p>
            </div>
          )}
        </DetailPopup>
      )}

      {/* SpO2 Detail */}
      {popup === "spo2" && (
        <DetailPopup title="Blood Oxygen (SpO2)" onClose={close}>
          <p className="text-xs text-text-tertiary mb-4">
            SpO2 measures oxygen saturation in your blood. Normal levels are 95-100%. Values below 95% may indicate respiratory issues.
          </p>
          <DetailRow label="Current SpO2" value={d.spo2 ? `${d.spo2.toFixed(1)}%` : "--"} />
          <DetailRow label="Average SpO2" value={d.avgSpo2 ? `${d.avgSpo2.toFixed(1)}%` : "--"} />
          <DetailRow label="Minimum Recorded" value={d.minSpo2 ? `${d.minSpo2.toFixed(1)}%` : "--"} hint="Lowest in dataset" />
          {d.spo2 && (
            <div className="mt-4 p-3 rounded-lg bg-surface-1/30">
              <p className="text-xs text-text-secondary">
                {d.spo2 >= 97 ? "Excellent blood oxygen levels."
                  : d.spo2 >= 95 ? "Normal blood oxygen levels."
                    : "SpO2 below 95% — consider consulting a physician if persistent."}
              </p>
            </div>
          )}
        </DetailPopup>
      )}

      {/* Skin Temp Detail */}
      {popup === "skinTemp" && (
        <DetailPopup title="Skin Temperature" onClose={close}>
          <p className="text-xs text-text-tertiary mb-4">
            Skin temperature can indicate physiological stress, illness onset, or hormonal changes. Track deviations from your personal baseline.
          </p>
          <DetailRow label="Current" value={d.skinTemp ? `${d.skinTemp.toFixed(2)}°C` : "--"} />
          <DetailRow label="Personal Baseline" value={d.avgSkinTemp ? `${d.avgSkinTemp.toFixed(2)}°C` : "--"} />
          <DetailRow label="Deviation" value={d.skinTempDeviation != null ? `${d.skinTempDeviation > 0 ? "+" : ""}${d.skinTempDeviation.toFixed(2)}°C` : "--"} hint="From your baseline" />
          <DetailRow label="Variability (σ)" value={d.skinTempStdDev ? `±${d.skinTempStdDev.toFixed(2)}°C` : "--"} />
          {d.skinTempDeviation != null && (
            <div className="mt-4 p-3 rounded-lg bg-surface-1/30">
              <p className="text-xs text-text-secondary">
                {Math.abs(d.skinTempDeviation) > 0.5
                  ? `Your skin temperature is ${d.skinTempDeviation > 0 ? "elevated" : "lower"} — this could indicate ${d.skinTempDeviation > 0 ? "illness onset, stress, or hormonal changes" : "improved recovery or cooler sleeping environment"}.`
                  : "Your skin temperature is within normal range."}
              </p>
            </div>
          )}
        </DetailPopup>
      )}
    </>
  );
}
