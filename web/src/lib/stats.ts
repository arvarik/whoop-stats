/**
 * Shared statistical helpers for deriving insights from WHOOP time-series data.
 */

/** Compute the arithmetic mean of a numeric array. Returns null for empty input. */
export function computeAvg(arr: number[]): number | null {
  if (!arr.length) return null;
  return arr.reduce((a, b) => a + b, 0) / arr.length;
}

/** Compute the population standard deviation. Returns null for fewer than 2 values. */
export function computeStdDev(arr: number[]): number | null {
  const avg = computeAvg(arr);
  if (avg === null || arr.length < 2) return null;
  const variance = arr.reduce((sum, v) => sum + (v - avg) ** 2, 0) / arr.length;
  return Math.sqrt(variance);
}

/** Compute the percentage change between two values. */
export function percentChange(current: number, previous: number): number | null {
  if (!previous) return null;
  return ((current - previous) / previous) * 100;
}
