import type { LayoutServerLoad } from "./$types";
import { apiFetch } from "$lib/server/api";

export const load: LayoutServerLoad = async (event) => {
  // hooks.server.ts already resolved the session into event.locals.user via a
  // single /api/me call; reuse it instead of fetching again.
  let user = event.locals.user ?? null;
  let streakDays = 0;
  let totalScore = 0;

  try {
    const [streakRes, scoreRes] = await Promise.all([
      apiFetch(event, "/api/gamification/streak"),
      apiFetch(event, "/api/gamification/score"),
    ]);

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
