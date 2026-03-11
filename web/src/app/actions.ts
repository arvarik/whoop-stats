"use server";

import { client } from "@/lib/api/client";
import { revalidatePath } from "next/cache";

/**
 * Server action to trigger an ad-hoc data sync with the WHOOP API.
 * Revalidates all dashboard routes after a successful sync so the UI
 * reflects the latest data.
 */
export async function syncWhoopData() {
  const { data, error, response } = await client.POST("/api/v1/sync");

  if (error || !response.ok) {
    throw new Error(error?.error?.message || "Failed to trigger sync");
  }

  // Revalidate all dashboard routes to reflect fresh data
  revalidatePath("/");
  revalidatePath("/recovery");
  revalidatePath("/sleep");
  revalidatePath("/strain");
  revalidatePath("/workouts");

  return data;
}
