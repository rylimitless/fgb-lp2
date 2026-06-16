import { redirect } from "@sveltejs/kit";
import type { PageServerLoad } from "./$types";
import { apiFetch } from "$lib/server/api";

export const load: PageServerLoad = async (event) => {
  const token = event.cookies.get("session_token");
  if (token) {
    throw redirect(307, "/");
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
