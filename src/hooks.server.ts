import { redirect, type Handle, type RequestEvent } from "@sveltejs/kit";
import { apiFetch } from "$lib/server/api";

// Public routes need no session. /welcome is the immersive marketing landing.
const PUBLIC_ROUTES = [
  "/login",
  "/setup",
  "/welcome",
  "/certificates",
  "/forgot-password",
  "/reset-password",
];

type User = {
  id: number;
  email: string;
  name: string;
  role: string;
  roles: string[];
};

// Cookie prefix (first 6 chars) for the auth_trace log. Mirrors the backend's
// authtrace.Prefix so SSR and backend rows can be correlated on the same key.
function cookiePrefix(token: string | undefined): string {
  if (!token) return "none";
  return token.length <= 6 ? token : token.slice(0, 6);
}

// Report an auth event (why we are about to redirect to /login) into the
// backend's auth_trace table via the unauthenticated debug endpoint. Best-
// effort: never blocks or fails the request. Runs server-side (SSR) so it has
// the browser's actual cookie and reports the SSR layer's perspective.
function reportTrace(
  event: RequestEvent,
  traceEvent: string,
  detail: Record<string, unknown>,
  path: string,
): void {
  const token = event.cookies.get("session_token");
  const body = {
    source: "ssr",
    event: traceEvent,
    path,
    cookie_prefix: cookiePrefix(token),
    detail,
  };
  // Fire-and-forget. Use the internal backend URL via apiFetch so this works
  // even though the session may already be invalid.
  apiFetch(event, "/api/_debug/auth-event", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  }).catch(() => {
    /* ignore — tracing must never break the request */
  });
}

// Route-prefix -> roles allowed to view it. Mirrors the nav `roles` arrays in
// (app)/+layout.svelte so the sidebar and the server-side guard agree. An empty
// roles array (or no entry) means "any authenticated user".
//
// Order matters: longer/more-specific prefixes are listed first so a path like
// "/learning-path-management/builder" matches its parent rule before a shorter
// one. Matching uses startsWith on the first matching entry.
const ROLE_GATED_ROUTES: { prefix: string; roles: string[] }[] = [
  { prefix: "/content-studio", roles: ["content creator"] },
  { prefix: "/course-builder", roles: ["content creator"] },
  { prefix: "/ai-content-generator", roles: ["content creator"] },
  { prefix: "/learning-path-management", roles: ["admin", "manager", "content creator"] },
  { prefix: "/review-queue", roles: ["approver"] },
  { prefix: "/analytics", roles: ["admin", "manager", "auditor"] },
  { prefix: "/user-management", roles: ["admin", "auditor"] },
  { prefix: "/department-management", roles: ["admin", "manager"] },
  { prefix: "/audit-log", roles: ["admin", "auditor"] },
  { prefix: "/developer", roles: ["admin"] },
];

function rolesAllow(userRoles: string[], required: string[]): boolean {
  // Admin bypasses every role check, matching the nav's hasRole() helper.
  if (userRoles.includes("admin")) return true;
  return required.some((r) => userRoles.includes(r));
}

function requiredRolesFor(pathname: string): string[] | null {
  for (const route of ROLE_GATED_ROUTES) {
    if (pathname === route.prefix || pathname.startsWith(route.prefix + "/")) {
      return route.roles;
    }
  }
  return null;
}

export const handle: Handle = async ({ event, resolve }) => {
  const sessionToken = event.cookies.get("session_token");
  const pathname = event.url.pathname;
  const isPublicRoute = PUBLIC_ROUTES.some((route) =>
    pathname.startsWith(route),
  );

  // Resolve the current user once per request (only when there's a session).
  // The result is stashed on locals.user so +layout.server.ts can reuse it
  // instead of re-fetching /api/me.
  //
  // A single transient backend failure (5xx, connection reset during an Air
  // reload, etc.) must NOT count as "logged out" — doing so bounces a still-
  // valid session to /login and, combined with the login page's cookie cleanup,
  // permanently destroys a good session. So we retry once before falling back
  // to the redirect below; only an explicit 401 is a definitive logout.
  //
  // `logoutReason` records WHY user ended up null so we can report it to the
  // auth_trace log when we redirect — this is the key signal for diagnosing the
  // "random logout" problem from the SSR layer's perspective.
  let user: User | null = null;
  let logoutReason: string | null = null;
  if (sessionToken) {
    const resolveUser = async (): Promise<Response | null> => {
      try {
        return await apiFetch(event, "/api/me");
      } catch {
        return null; // network error — backend unreachable
      }
    };

    let res = await resolveUser();
    if (res && !res.ok && res.status !== 401) {
      // Transient failure (5xx, proxy hiccup): wait briefly and retry once.
      await new Promise((r) => setTimeout(r, 500));
      res = await resolveUser();
    }

    if (res?.ok) {
      const data = await res.json();
      const role = data.role as string;
      user = {
        id: data.id as number,
        email: data.email as string,
        name: data.name as string,
        role,
        roles: (data.roles ?? [role]) as string[],
      };
    } else if (!res) {
      logoutReason = "network_error"; // backend unreachable after retry
    } else if (res.status === 401) {
      logoutReason = "session_invalid_401"; // explicit 401 from /api/me
    } else {
      logoutReason = `http_${res.status}`; // persistent 5xx after retry
    }
  } else {
    logoutReason = "no_session_cookie";
  }
  event.locals.user = user;

  // Unauthenticated user on a protected route:
  //  - landing on the bare root → show the immersive /welcome experience
  //  - any deeper protected route → keep the direct login flow (preserves
  //    deep-link intent and the existing behaviour)
  if (!user && !isPublicRoute) {
    // Record WHY this redirect happened so we can trace random logouts. The
    // detail includes the reason, the route the user was trying to reach, and
    // the referrer (which tab/origin) when available.
    reportTrace(
      event,
      "ssr_redirect",
      {
        reason: logoutReason ?? "unknown",
        target: pathname === "/" ? "/welcome" : "/login",
        referrer: event.request.headers.get("referer") ?? null,
      },
      pathname,
    );
    throw redirect(307, pathname === "/" ? "/welcome" : "/login");
  }

  // Authenticated but lacking the role for this route → send to the dashboard
  // rather than leaking the page. (Backend API still enforces the same rules,
  // so this is defense-in-depth, not the only gate.)
  if (user) {
    const required = requiredRolesFor(pathname);
    if (required && !rolesAllow(user.roles, required)) {
      throw redirect(303, "/");
    }
  }

  const response = await resolve(event);

  // Capture any Set-Cookie the SSR layer is about to send that touches
  // session_token. If SvelteKit (or any load function) mutated event.cookies,
  // it surfaces here — this is how we'd catch an SSR-originated cookie deletion.
  const setCookies = response.headers.getSetCookie?.() ?? [];
  for (const sc of setCookies) {
    if (sc.includes("session_token")) {
      const category =
        sc.includes("Max-Age=0") ||
        sc.includes("Max-Age=-1") ||
        sc.includes("session_token=;")
          ? "delete"
          : "set";
      reportTrace(
        event,
        "ssr_set_cookie",
        { category, header: sc },
        pathname,
      );
    }
  }

  return response;
};
