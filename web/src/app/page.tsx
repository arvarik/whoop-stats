import { Suspense } from "react";
import { client } from "@/lib/api/client";
import { MetricCard, getRecoveryColorStr } from "@/components/MetricCard";
import { StrainRecoveryChart } from "@/components/StrainRecoveryChart";
import { SyncButton } from "@/components/SyncButton";
import { SleepStagesBar } from "@/components/SleepStagesBar";
import GlobeMapWrapper from "@/components/GlobeMapWrapper";
import { Activity, Moon, HeartPulse, Flame } from "lucide-react";

export const revalidate = 300; // revalidate every 5 mins
export const dynamic = "force-dynamic";

export default async function DashboardPage() {
  // Fetch initial data securely on the server
  const [profileRes, cyclesRes, sleepsRes, insightsRes] = await Promise.all([
    client.GET("/api/v1/user/profile"),
    client.GET("/api/v1/cycles", { params: { query: { cursor: new Date().toISOString() } } }),
    client.GET("/api/v1/sleeps", { params: { query: { cursor: new Date().toISOString() } } }),
    client.GET("/api/v1/insights"),
  ]);

  // Extract relevant metrics
  const latestCycle = cyclesRes.data?.[0];
  const latestSleep = sleepsRes.data?.[0];

  // Strain Data
  const currentStrain = latestCycle?.strain ? Number(latestCycle.strain).toFixed(1) : "--";
  const kilojoules = latestCycle?.kilojoule ? `${Math.round(Number(latestCycle.kilojoule)).toLocaleString()} kJ burned` : "No energy data";

  // Recovery Data (From insights for score, HRV/RHR from cycle if available, but since we didn't fetch recoveries directly, let's use what we have)
  const insightsData = insightsRes.data as { strain?: Record<string, unknown>[], recovery?: Record<string, unknown>[] } | undefined;
  const latestRecoveryInsight = insightsData?.recovery?.[(insightsData.recovery.length || 1) - 1];
  const recoveryScore = latestRecoveryInsight?.avg_recovery ? Math.round(Number(latestRecoveryInsight.avg_recovery)) : "--";
  const recoveryColor = typeof recoveryScore === "number" ? getRecoveryColorStr(recoveryScore) : "none";
  const recoverySubtext = typeof recoveryScore === "number" 
    ? (recoveryScore >= 66 ? "Primed to perform" : recoveryScore >= 34 ? "Moderate readiness" : "Take it easy")
    : "--";

  // Sleep Data
  const sleepPerformance = latestSleep?.performance_score ? Math.round(Number(latestSleep.performance_score)) : "--";
  let sleepDurationText = "No duration data";
  const lightMs = Number(latestSleep?.total_light_sleep_time_milli || 0);
  const remMs = Number(latestSleep?.total_rem_sleep_time_milli || 0);
  const deepMs = Number(latestSleep?.total_slow_wave_sleep_time_milli || 0);
  const awakeMs = Number(latestSleep?.total_awake_time_milli || 0);
  
  if (lightMs || remMs || deepMs) {
    const totalSleepMs = lightMs + deepMs + remMs;
    const hours = Math.floor(totalSleepMs / (1000 * 60 * 60));
    const mins = Math.floor((totalSleepMs % (1000 * 60 * 60)) / (1000 * 60));
    sleepDurationText = `${hours}h ${mins}m actual sleep`;
  }

  // Format chart data
  const chartData = insightsData?.strain?.map((s: Record<string, unknown>, idx: number) => ({
    date: s.bucket as string,
    strain: s.avg_strain ? Number(s.avg_strain) : null,
    recovery: insightsData.recovery?.[idx]?.avg_recovery ? Number(insightsData.recovery[idx].avg_recovery) : null,
  })) || [];

  return (
    <div className="min-h-screen bg-[#09090b] text-zinc-50 font-sans selection:bg-indigo-500/30">
      {/* Subtle top ambient glow */}
      <div className="fixed top-0 left-0 right-0 h-[500px] w-full bg-[radial-gradient(ellipse_at_top,_var(--tw-gradient-stops))] from-indigo-900/10 via-[#09090b] to-[#09090b] -z-10 pointer-events-none" />

      <main className="mx-auto max-w-6xl p-6 md:p-10 space-y-12 pb-24">
        
        {/* Header */}
        <header className="flex flex-col md:flex-row md:items-end justify-between gap-6 pt-10">
          <div className="space-y-2">
            <h1 className="text-4xl font-semibold tracking-tighter text-zinc-100">
              Overview
            </h1>
            <p className="text-lg tracking-tight text-zinc-400">
              Welcome back{profileRes.data?.first_name ? `, ${profileRes.data.first_name}` : ""}.
            </p>
          </div>
          <div className="flex items-center gap-3">
            <SyncButton />
          </div>
        </header>

        {/* Top 3 Metrics */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <MetricCard
            title="Day Strain"
            value={currentStrain}
            subtitle={
              <div className="flex items-center gap-1.5 mt-1">
                <Flame className="w-3.5 h-3.5 text-orange-500" />
                <span>{kilojoules}</span>
              </div>
            }
            icon={<Activity className="w-4 h-4" />}
            gradientColor="blue"
          />

          <MetricCard
            title="Recovery"
            value={`${recoveryScore}%`}
            subtitle={recoverySubtext}
            icon={<HeartPulse className="w-4 h-4" />}
            gradientColor={recoveryColor}
          />

          <MetricCard
            title="Sleep Performance"
            value={`${sleepPerformance}%`}
            subtitle={sleepDurationText}
            icon={<Moon className="w-4 h-4" />}
            gradientColor="indigo"
          >
            <div className="mt-4">
              <SleepStagesBar light={lightMs} rem={remMs} deep={deepMs} awake={awakeMs} />
            </div>
          </MetricCard>
        </div>

        {/* Main Chart Section */}
        <div className="rounded-3xl border border-white/[0.08] bg-zinc-900/20 p-6 md:p-8 backdrop-blur-xl">
          <Suspense fallback={<div className="w-full h-[400px] animate-pulse bg-white/5 rounded-xl" />}>
            <StrainRecoveryChart data={chartData} />
          </Suspense>
        </div>

        {/* Bottom Section - Map */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          <div className="rounded-3xl border border-white/[0.08] bg-zinc-900/20 p-6 md:p-8 backdrop-blur-xl flex flex-col min-h-[400px]">
            <div className="mb-4">
              <h2 className="text-lg font-semibold tracking-tight text-white">Global Activities</h2>
              <p className="text-sm text-zinc-400">Recent workout locations</p>
            </div>
            <div className="flex-1 relative -mx-4 -mb-4 overflow-hidden rounded-b-xl">
               <Suspense fallback={<div className="w-full h-full animate-pulse bg-white/5" />}>
                 <GlobeMapWrapper />
               </Suspense>
            </div>
          </div>
          
          <div className="rounded-3xl border border-white/[0.08] bg-zinc-900/20 p-6 md:p-8 backdrop-blur-xl flex flex-col justify-center items-center text-center">
             <div className="w-16 h-16 rounded-full bg-zinc-800/50 flex items-center justify-center mb-4 border border-white/5">
                <Activity className="w-8 h-8 text-zinc-500" />
             </div>
             <h3 className="text-lg font-semibold tracking-tight text-white mb-2">Detailed Workout Zones</h3>
             <p className="text-sm text-zinc-400 max-w-sm">
               Connect more detailed workout data to visualize your time spent in Heart Rate Zones 1-5.
             </p>
          </div>
        </div>

      </main>
    </div>
  );
}
