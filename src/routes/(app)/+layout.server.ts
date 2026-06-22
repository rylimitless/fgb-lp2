import type { LayoutServerLoad } from "./$types";
import { apiFetch } from "$lib/server/api";

export const load: LayoutServerLoad = async (event) => {
  let user = null;
  let streakDays = 0;
  let totalScore = 0;

  try {
    const [meRes, streakRes, scoreRes] = await Promise.all([
      apiFetch(event, "/api/me"),
      apiFetch(event, "/api/gamification/streak"),
      apiFetch(event, "/api/gamification/score"),
    ]);

    if (meRes.ok) {
      const userData = await meRes.json();
      user = {
        id: userData.id as number,
        email: userData.email as string,
        name: userData.name as string,
        role: userData.role as string,
        roles: (userData.roles ?? [userData.role]) as string[],
      };
    }

    if (streakRes.ok) {
      const sData = await streakRes.json();
      streakDays = sData.streak_days ?? 0;
    }

    if (scoreRes.ok) {
      const scData = await scoreRes.json();
      totalScore = scData.total_score ?? 0;
    }
  } catch {
    // User not authenticated or fetch failed — hooks.server will redirect to login
  }
  return { user, gamification: { streakDays, totalScore } };
};
