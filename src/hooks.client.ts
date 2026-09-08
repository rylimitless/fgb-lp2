import { redirect } from "@sveltejs/kit";
import type { HandleFetch } from "@sveltejs/kit";

// Best-effort trace report when the CLIENT-side layer detects a 401 and
// bounces to /login. Mirrors what hooks.server.ts reports from the SSR layer.
// Fire-and-forget so it never blocks the redirect.
function reportClientTrace(
  event: string,
  pathname: string,
  detail: Record<string, unknown>,
): void {
  try {
    fetch("/api/_debug/auth-event", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({
        source: "browser",
        event,
        path: pathname,
        detail,
      }),
      keepalive: true,
    }).catch(() => {
      /* ignore */
    });
  } catch {
    /* ignore */
  }
}

export const handleFetch: HandleFetch = async ({ event, request, fetch }) => {
  const response = await fetch(request);

  // A 401 from a load-function fetch means the session cookie wasn't accepted.
  // Before we bounce the user to /login, record the event + which request
  // triggered it, so the trace shows which client load revealed the dead
  // session. (Component-level fetches are NOT routed through handleFetch, so
  // this only catches load fetches — but that's the meaningful set, since a
  // 401 there is exactly what triggers the SvelteKit redirect.)
  if (
    response.status === 401 &&
    !event.url.pathname.startsWith("/login") &&
    !event.url.pathname.startsWith("/api/_debug")
  ) {
    reportClientTrace("client_redirect", event.url.pathname, {
      request_url:
        typeof request.url === "string" ? request.url : String(request.url),
    });
    throw redirect(307, "/login");
  }

  return response;
};
