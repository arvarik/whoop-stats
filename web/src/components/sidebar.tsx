"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState } from "react";
import { cn } from "@/lib/utils";
import {
  LayoutDashboard,
  HeartPulse,
  Moon,
  Flame,
  Dumbbell,
  ChevronsLeft,
  ChevronsRight,
} from "lucide-react";

const navItems = [
  { href: "/", label: "Overview", icon: LayoutDashboard },
  { href: "/recovery", label: "Recovery", icon: HeartPulse },
  { href: "/sleep", label: "Sleep", icon: Moon },
  { href: "/strain", label: "Strain", icon: Flame },
  { href: "/workouts", label: "Workouts", icon: Dumbbell },
];

export function Sidebar() {
  const pathname = usePathname();
  const [collapsed, setCollapsed] = useState(false);

  return (
    <aside
      className={cn(
        "hidden md:flex flex-col h-screen sticky top-0 border-r border-border-subtle bg-surface-0/50 backdrop-blur-xl transition-all duration-300 z-30",
        collapsed ? "w-[68px]" : "w-[220px]"
      )}
    >
      {/* Logo */}
      <div className="flex items-center gap-3 px-4 h-14 border-b border-border-subtle">
        <div className="w-8 h-8 rounded-lg bg-accent/20 flex items-center justify-center flex-shrink-0">
          <span className="text-accent font-bold text-sm">W</span>
        </div>
        {!collapsed && (
          <span className="text-sm font-semibold text-text-primary tracking-tight truncate">
            WHOOP Stats
          </span>
        )}
      </div>

      {/* Navigation */}
      <nav className="flex-1 py-3 px-2 space-y-0.5">
        {navItems.map((item) => {
          const isActive = pathname === item.href;
          return (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                "flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-all duration-200 group",
                isActive
                  ? "bg-accent-muted text-text-primary"
                  : "text-text-secondary hover:text-text-primary hover:bg-surface-1"
              )}
            >
              <item.icon
                className={cn(
                  "w-[18px] h-[18px] flex-shrink-0 transition-colors",
                  isActive ? "text-accent" : "text-text-tertiary group-hover:text-text-secondary"
                )}
              />
              {!collapsed && <span className="truncate">{item.label}</span>}
            </Link>
          );
        })}
      </nav>

      {/* Collapse toggle */}
      <button
        onClick={() => setCollapsed(!collapsed)}
        className="flex items-center justify-center h-10 mx-2 mb-3 rounded-lg text-text-tertiary hover:text-text-secondary hover:bg-surface-1 transition-colors"
      >
        {collapsed ? <ChevronsRight className="w-4 h-4" /> : <ChevronsLeft className="w-4 h-4" />}
      </button>
    </aside>
  );
}
