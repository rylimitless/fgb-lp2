import { redirect } from "@sveltejs/kit";
import type { PageServerLoad } from "./$types";
import { apiFetch } from "$lib/server/api";

export const load: PageServerLoad = async (event) => {
  // Only redirect away from the login page if the session token is actually
  // valid. Checking the cookie's mere presence causes a redirect loop when a
  // stale/expired token lingers in the browser: /login → / → /welcome → /login.
  // We rely on hooks.server.ts to have already populated event.locals.user from
  // /api/me, so a valid session means the user is genuinely authenticated.
  const user = event.locals.user;
  if (user) {
    throw redirect(307, "/");
  }

  // If the session is invalid/expired, clear the stale cookie so the browser
  // doesn't keep sending it on every request (which would make every hit to
  // a public route needlessly try /api/me).
  const token = event.cookies.get("session_token");
  if (token) {
    event.cookies.delete("session_token", { path: "/" });
  }

  try {
    const res = await apiFetch(event, "/api/check-first-user");
    if (!res.ok) return;
    const data = await res.json();
    if (data.first_user) {
      throw redirect(307, "/setup");
    }
  } catch (e) {
    if (e && typeof e === "object" && "status" in e) throw e;
  }
};
