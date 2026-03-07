"use client";

import dynamic from "next/dynamic";
import { Skeleton } from "@/components/ui/skeleton";

// Safely dynamically load the map without SSR
const GlobeMap = dynamic(() => import("@/components/GlobeMap"), {
  ssr: false,
  loading: () => <Skeleton className="w-full h-full min-h-[300px] rounded-xl bg-white/5" />,
});

export default function GlobeMapWrapper() {
  return <GlobeMap />;
}
