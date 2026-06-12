import { redirect } from "@sveltejs/kit";
import type { PageServerLoad } from "./$types";

export const load: PageServerLoad = async ({ fetch }) => {
  try {
    const res = await fetch("/api/check-first-user");
    if (!res.ok) return;
    const data = await res.json();
    if (data.first_user) {
      throw redirect(307, "/setup");
    }
  } catch (e) {
    if (e && typeof e === "object" && "status" in e) throw e;
  }
};
