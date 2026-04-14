"use client";

import { useEffect } from "react";

/**
 * Global error boundary for the app. Catches rendering errors and API failures,
 * displaying a user-friendly message with a retry button.
 */
export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    console.error("[WHOOP Stats] Page error:", error);
  }, [error]);

  return (
    <div className="px-4 md:px-8 lg:px-10 py-6 md:py-8 max-w-7xl mx-auto">
      <div className="glass-card p-8 text-center space-y-4">
        <div className="w-12 h-12 rounded-xl bg-rose-500/10 flex items-center justify-center mx-auto">
          <span className="text-rose-400 text-xl">!</span>
        </div>
        <h2 className="text-lg font-semibold text-text-primary">
          Something went wrong
        </h2>
        <p className="text-sm text-text-secondary max-w-md mx-auto">
          Failed to load data from the WHOOP Stats backend. Make sure the backend
          service is running and your environment variables are configured correctly.
        </p>
        <button
          onClick={reset}
          className="px-4 py-2 rounded-lg bg-accent text-white text-sm font-medium hover:bg-accent-hover transition-colors"
        >
          Try Again
        </button>
      </div>
    </div>
  );
}
