/**
 * Backend API client for server-side data fetching.
 *
 * Uses openapi-fetch with auto-generated types from the Go backend's OpenAPI spec.
 * JWTs are generated server-side using the shared encryption key — this is secure
 * because all API calls happen in React Server Components or Server Actions.
 */
import createClient from "openapi-fetch";
import type { paths } from "./schema";
import { SignJWT } from "jose";

// --- Required environment variables (fail fast if missing) ---
const backendUrl = process.env.NEXT_PUBLIC_API_URL;
if (!backendUrl) {
  throw new Error(
    "NEXT_PUBLIC_API_URL is required. Set it to your backend URL (e.g. http://localhost:8080)"
  );
}

const encryptionKey = process.env.WHOOP_STATS_ENCRYPTION_KEY;
if (!encryptionKey) {
  throw new Error(
    "WHOOP_STATS_ENCRYPTION_KEY is required. Must match the backend's ENCRYPTION_KEY."
  );
}

const whoopUserId = process.env.WHOOP_STATS_WHOOP_USER_ID;
if (!whoopUserId) {
  throw new Error(
    "WHOOP_STATS_WHOOP_USER_ID is required. Set it to your WHOOP user ID."
  );
}

// --- JWT token generation & caching ---
let cachedToken: string | null = null;
let tokenExpiry = 0;

async function generateToken(): Promise<string> {
  const secret = new TextEncoder().encode(encryptionKey);
  return new SignJWT({ whoop_user_id: whoopUserId })
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
export const client = createClient<paths>({
  baseUrl: backendUrl,
  fetch: async (input: RequestInfo | URL, init?: RequestInit) => {
    const token = await getToken();
    const headers = new Headers(init?.headers);
    headers.set("Authorization", `Bearer ${token}`);
    return fetch(input, { ...init, headers });
  },
});
