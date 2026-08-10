import { redirect, type Handle } from "@sveltejs/kit";
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
  let user: User | null = null;
  if (sessionToken) {
    try {
      const res = await apiFetch(event, "/api/me");
      if (res.ok) {
        const data = await res.json();
        const role = data.role as string;
        user = {
          id: data.id as number,
          email: data.email as string,
          name: data.name as string,
          role,
          roles: (data.roles ?? [role]) as string[],
        };
      }
    } catch {
      // Backend unreachable / invalid session — treat as logged out below.
    }
  }
  event.locals.user = user;

  // Unauthenticated user on a protected route:
  //  - landing on the bare root → show the immersive /welcome experience
  //  - any deeper protected route → keep the direct login flow (preserves
  //    deep-link intent and the existing behaviour)
  if (!user && !isPublicRoute) {
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

  return resolve(event);
};
