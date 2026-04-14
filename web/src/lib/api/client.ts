/**
 * Backend API client for server-side data fetching.
 *
 * Uses openapi-fetch with auto-generated types from the Go backend's OpenAPI spec.
 * JWTs are generated server-side using the shared encryption key — this is secure
 * because all API calls happen in React Server Components or Server Actions.
 *
 * Environment variables are validated lazily (on first API call) rather than at
 * module evaluation time, because Next.js evaluates all imports during the build
 * step even for dynamic pages. The env vars are only available at runtime.
 */
import createClient from "openapi-fetch";
import type { paths } from "./schema";
import { SignJWT } from "jose";

// --- Lazy environment variable access ---
// Next.js evaluates module-level code during `next build` (collecting page data).
// Throwing at module scope causes CI builds to fail when env vars aren't set.
// Instead, we read env vars lazily on first use (runtime only).

function getRequiredEnv(name: string, hint: string): string {
  const value = process.env[name];
  if (!value) {
    throw new Error(`${name} is required. ${hint}`);
  }
  return value;
}

// --- JWT token generation & caching ---
let cachedToken: string | null = null;
let tokenExpiry = 0;

async function generateToken(): Promise<string> {
  const key = getRequiredEnv(
    "WHOOP_STATS_ENCRYPTION_KEY",
    "Must match the backend's ENCRYPTION_KEY."
  );
  const userId = getRequiredEnv(
    "WHOOP_STATS_WHOOP_USER_ID",
    "Set it to your WHOOP user ID."
  );

  const secret = new TextEncoder().encode(key);
  return new SignJWT({ whoop_user_id: userId })
    .setProtectedHeader({ alg: "HS256" })
    .setIssuedAt()
    .setExpirationTime("24h")
    .sign(secret);
}

async function getToken(): Promise<string> {
  const now = Date.now();
  // Refresh if expired or within 1 minute of expiry
  if (!cachedToken || now >= tokenExpiry - 60_000) {
    cachedToken = await generateToken();
    tokenExpiry = now + 23 * 60 * 60 * 1000; // ~23 hours
  }
  return cachedToken;
}

// --- OpenAPI client with auth header injection ---
// baseUrl is read lazily via the fetch wrapper so it doesn't throw during build.
export const client = createClient<paths>({
  baseUrl: process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080",
  fetch: async (input: RequestInfo | URL, init?: RequestInit) => {
    // Validate at runtime (first actual API call), not at build time
    getRequiredEnv(
      "NEXT_PUBLIC_API_URL",
      "Set it to your backend URL (e.g. http://localhost:8080)"
    );

    const token = await getToken();
    const headers = new Headers(init?.headers);
    headers.set("Authorization", `Bearer ${token}`);
    return fetch(input, { ...init, headers });
  },
});
