import { ReactNode } from "react";
import { cn } from "@/lib/utils";

interface MetricCardProps {
  title: string;
  value: string | number;
  subtitle?: ReactNode;
  icon?: ReactNode;
  gradientColor?: "green" | "yellow" | "red" | "blue" | "indigo" | "none";
  className?: string;
  children?: ReactNode; // For mini charts or extra info
}

const gradientMap = {
  green: "from-emerald-500/20 via-emerald-500/5 to-transparent",
  yellow: "from-amber-500/20 via-amber-500/5 to-transparent",
  red: "from-rose-500/20 via-rose-500/5 to-transparent",
  blue: "from-blue-500/20 via-blue-500/5 to-transparent",
  indigo: "from-indigo-500/20 via-indigo-500/5 to-transparent",
  none: "",
};

export function MetricCard({ title, value, subtitle, icon, gradientColor = "none", className, children }: MetricCardProps) {
  const gradient = gradientMap[gradientColor];

  return (
    <div className={cn(
      "group relative overflow-hidden rounded-2xl border border-white/[0.08] bg-zinc-900/30 p-6 backdrop-blur-xl transition-all duration-300 hover:border-white/[0.15] hover:bg-zinc-900/50",
      className
    )}>
      {/* Background glow effect on hover */}
      {gradientColor !== "none" && (
        <div 
          className={cn(
            "absolute -inset-px opacity-0 transition-opacity duration-500 group-hover:opacity-100",
            "bg-gradient-to-br", gradient
          )} 
        />
      )}
      
      {/* Radial subtle base background */}
      {gradientColor !== "none" && (
         <div 
          className={cn(
            "absolute top-0 right-0 w-32 h-32 -translate-y-16 translate-x-16 rounded-full blur-[50px] opacity-40 transition-opacity duration-500 group-hover:opacity-70",
            gradientColor === "green" ? "bg-emerald-500" :
            gradientColor === "yellow" ? "bg-amber-500" :
            gradientColor === "red" ? "bg-rose-500" :
            gradientColor === "blue" ? "bg-blue-500" :
            gradientColor === "indigo" ? "bg-indigo-500" : ""
          )} 
         />
      )}

      <div className="relative z-10 flex flex-col h-full">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-medium text-zinc-400 tracking-tight">{title}</h3>
          {icon && <div className="text-zinc-500 group-hover:text-zinc-300 transition-colors">{icon}</div>}
        </div>
        
        <div className="mt-4 flex items-baseline gap-2">
          <span className="text-4xl font-semibold tracking-tighter text-zinc-50">{value}</span>
        </div>

        {subtitle && (
          <div className="mt-2 text-sm font-medium text-zinc-500">
            {subtitle}
          </div>
        )}

        {children && (
          <div className="mt-6 flex-1 flex flex-col justify-end">
            {children}
          </div>
        )}
      </div>
    </div>
  );
}

export function getRecoveryColorStr(score: number): "green" | "yellow" | "red" {
  if (score >= 66) return "green";
  if (score >= 34) return "yellow";
  return "red";
}
