import { client } from "@/lib/api/client";
import { WorkoutFeed } from "@/components/workout-feed";
import type { ApiRecord } from "@/lib/types";

export const dynamic = "force-dynamic";

export default async function WorkoutsPage() {
  const workoutsRes = await client.GET("/api/v1/workouts", {
    params: { query: { cursor: new Date().toISOString() } },
  });

  const workouts = (workoutsRes.data as ApiRecord[]) || [];

  return (
    <div className="px-4 md:px-8 lg:px-10 py-6 md:py-8 max-w-7xl mx-auto space-y-6">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight text-text-primary">
          Workouts
        </h1>
        <p className="text-sm text-text-tertiary mt-0.5">
          {workouts.length} activities logged
        </p>
      </header>

      <WorkoutFeed workouts={workouts} />
    </div>
  );
}
