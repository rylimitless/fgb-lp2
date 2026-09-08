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

  // Deliberately do NOT delete a lingering session_token cookie here. When
  // hooks.server.ts fails to resolve the user due to a TRANSIENT backend
  // problem (5xx, restart, network blip) rather than a genuinely invalid
  // session, deleting the cookie would destroy a still-valid session and turn
  // a one-off hiccup into a real logout. A stale cookie is harmless — it is
  // overwritten on the next successful login — and costs one extra /api/me
  // call per visit to the login page.

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
