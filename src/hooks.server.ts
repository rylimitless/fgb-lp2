import { redirect, type Handle } from "@sveltejs/kit";

const PUBLIC_ROUTES = ["/login", "/setup"];

export const handle: Handle = async ({ event, resolve }) => {
  const sessionToken = event.cookies.get("session_token");
  const isPublicRoute = PUBLIC_ROUTES.some((route) =>
    event.url.pathname.startsWith(route),
  );

  // Authenticated user on a public route → redirect to home
  if (sessionToken && isPublicRoute) {
    throw redirect(307, "/");
  }

  // Unauthenticated user on a protected route → redirect to login
  if (!sessionToken && !isPublicRoute) {
    throw redirect(307, "/login");
  }

  return resolve(event);
};
