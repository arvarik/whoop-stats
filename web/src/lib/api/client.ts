import createClient from "openapi-fetch";
import type { paths } from "./schema";

// Ensure this runs on the server side and securely talks to the backend
const backendUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

// For server-side fetching, we would use the WHOOP_OAUTH_TOKEN if we were forwarding it from the user's session
// In this case, our backend uses a JWT which contains whoop_user_id. For testing locally we can pass a dummy token or rely on backend config
const authToken = process.env.WHOOP_STATS_TOKEN || "test-token";

export const client = createClient<paths>({
  baseUrl: backendUrl,
  headers: {
    Authorization: `Bearer ${authToken}`,
  },
});
