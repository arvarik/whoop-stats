"use client";

import { useState } from "react";
import { WorkoutCard } from "@/components/workout-card";
import { WorkoutDetail } from "@/components/workout-detail";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type AnyRecord = Record<string, any>;

export function RecentWorkouts({ workouts }: { workouts: AnyRecord[] }) {
  const [detailWorkout, setDetailWorkout] = useState<AnyRecord | null>(null);

  return (
    <>
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        {workouts.map((w, i) => (
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

      {detailWorkout && (
        <WorkoutDetail workout={detailWorkout} onClose={() => setDetailWorkout(null)} />
      )}
    </>
  );
}
