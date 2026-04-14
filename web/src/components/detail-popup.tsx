"use client";

import { useState, ReactNode } from "react";
import { X } from "lucide-react";

interface DetailPopupProps {
  title: string;
  children: ReactNode;
  onClose: () => void;
}

export function DetailPopup({ title, children, onClose }: DetailPopupProps) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4" onClick={onClose}>
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" />
      <div
        className="relative w-full max-w-lg max-h-[80vh] overflow-y-auto rounded-2xl border border-border-subtle bg-surface-0/95 backdrop-blur-xl p-6 shadow-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        <button
          onClick={onClose}
          className="absolute top-4 right-4 p-1 rounded-lg text-text-muted hover:text-text-primary hover:bg-surface-2/50 transition-colors"
        >
          <X className="w-4 h-4" />
        </button>
        <h3 className="text-lg font-semibold text-text-primary mb-4">{title}</h3>
        {children}
      </div>
    </div>
  );
}

/** Simple row for detail popups */
export function DetailRow({ label, value, hint }: { label: string; value: string | number; hint?: string }) {
  return (
    <div className="flex items-center justify-between py-2 border-b border-border-subtle/30 last:border-0">
      <div>
        <span className="text-sm text-text-secondary">{label}</span>
        {hint && <p className="text-[10px] text-text-muted mt-0.5">{hint}</p>}
      </div>
      <span className="text-sm font-semibold text-text-primary">{value}</span>
    </div>
  );
}

/** Hook to manage popup state */
export function useDetailPopup() {
  const [popup, setPopup] = useState<string | null>(null);
  return { popup, open: setPopup, close: () => setPopup(null) };
}
