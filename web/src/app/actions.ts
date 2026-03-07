"use server";

import { client } from "@/lib/api/client";
import { revalidatePath } from "next/cache";

export async function syncWhoopData() {
  const { data, error, response } = await client.POST("/api/v1/sync");

  if (error || !response.ok) {
    throw new Error(error?.error?.message || "Failed to trigger sync");
  }

  // Once synced successfully, we revalidate the dashboard path to refresh data
  revalidatePath("/");
  return data;
}
