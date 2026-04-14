"use client";

/**
 * Global loading state shown while page data is being fetched.
 * Uses skeleton UI matching the dashboard layout.
 */
export default function Loading() {
  return (
    <div className="px-4 md:px-8 lg:px-10 py-6 md:py-8 max-w-7xl mx-auto space-y-6 animate-pulse">
      {/* Header skeleton */}
      <div className="space-y-2">
        <div className="h-7 w-32 bg-surface-1 rounded-lg" />
        <div className="h-4 w-48 bg-surface-1/50 rounded-md" />
      </div>

      {/* Metric cards skeleton */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
        {[1, 2, 3].map((i) => (
          <div key={i} className="glass-card p-5 space-y-3">
            <div className="h-4 w-20 bg-surface-2 rounded" />
            <div className="h-8 w-16 bg-surface-2 rounded" />
            <div className="h-3 w-28 bg-surface-1 rounded" />
          </div>
        ))}
      </div>

      {/* Chart skeleton */}
      <div className="glass-card p-5">
        <div className="h-4 w-24 bg-surface-2 rounded mb-4" />
        <div className="h-[300px] bg-surface-1/30 rounded-xl" />
      </div>
    </div>
  );
}
