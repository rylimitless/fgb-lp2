import { redirect, type Handle } from "@sveltejs/kit";

// Public routes need no session. /welcome is the immersive marketing landing.
const PUBLIC_ROUTES = ["/login", "/setup", "/welcome", "/certificates"];

export const handle: Handle = async ({ event, resolve }) => {
  const sessionToken = event.cookies.get("session_token");
  const pathname = event.url.pathname;
  const isPublicRoute = PUBLIC_ROUTES.some((route) =>
    pathname.startsWith(route),
  );

  // Unauthenticated user on a protected route:
  //  - landing on the bare root → show the immersive /welcome experience
  //  - any deeper protected route → keep the direct login flow (preserves
  //    deep-link intent and the existing behaviour)
  if (!sessionToken && !isPublicRoute) {
    throw redirect(307, pathname === "/" ? "/welcome" : "/login");
  }

  return resolve(event);
};
