import { redirect } from "@sveltejs/kit";
import type { PageLoad } from "./$types";

export const load: PageLoad = async () => {
  try {
    const res = await fetch("http://localhost:5555/api/check-first-user");
    if (!res.ok) return; // server not ready, just show login
    const data = await res.json();
    if (data.first_user) {
      throw redirect(307, "/setup");
    }
  } catch (e) {
    if (e && typeof e === "object" && "status" in e) throw e;
    // Server not reachable — show login anyway
  }
};
