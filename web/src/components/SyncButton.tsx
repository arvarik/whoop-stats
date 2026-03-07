"use client";

import { useTransition } from "react";
import { RefreshCw } from "lucide-react";
import { syncWhoopData } from "@/app/actions";
import { toast } from "sonner";
import { cn } from "@/lib/utils";

export function SyncButton() {
  const [isPending, startTransition] = useTransition();

  const handleSync = () => {
    startTransition(async () => {
      try {
        await syncWhoopData();
        toast.success("Sync triggered successfully", {
          description: "Your data is being refreshed in the background.",
        });
      } catch (err: unknown) {
        toast.error("Sync failed", {
          description: err instanceof Error ? err.message : String(err),
        });
      }
    });
  };

  return (
    <button
      onClick={handleSync}
      disabled={isPending}
      className={cn(
        "group flex items-center justify-center gap-2 rounded-full px-4 py-2 text-sm font-medium tracking-tight transition-all duration-300",
        "border border-white/[0.08] bg-zinc-900/50 text-zinc-300 backdrop-blur-md",
        "hover:border-white/[0.15] hover:bg-zinc-800 hover:text-white",
        "active:scale-95 disabled:pointer-events-none disabled:opacity-50"
      )}
    >
      <RefreshCw className={cn("w-4 h-4 text-zinc-400 group-hover:text-zinc-300", isPending && "animate-spin")} />
      {isPending ? "Syncing..." : "Sync"}
    </button>
  );
}
