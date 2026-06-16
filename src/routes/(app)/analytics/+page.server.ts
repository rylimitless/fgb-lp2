import type { PageServerLoad } from "./$types";
import { apiFetch } from "$lib/server/api";

export const load: PageServerLoad = async (event) => {
  const [overviewRes, failedRes, courseEffRes, coachUsageRes, auditRes] =
    await Promise.all([
      apiFetch(event, "/api/analytics/overview"),
      apiFetch(event, "/api/analytics/most-failed"),
      apiFetch(event, "/api/analytics/course-effectiveness"),
      apiFetch(event, "/api/analytics/coach-usage"),
      apiFetch(event, "/api/analytics/audit-log?limit=100"),
    ]);

  const overview = overviewRes.ok ? await overviewRes.json() : null;
  const mostFailed = failedRes.ok ? await failedRes.json() : [];
  const courseEffectiveness = courseEffRes.ok ? await courseEffRes.json() : [];
  const coachUsage = coachUsageRes.ok ? await coachUsageRes.json() : [];
  const auditLog = auditRes.ok ? await auditRes.json() : [];

  return {
    overview,
    mostFailed,
    courseEffectiveness,
    coachUsage,
    auditLog,
  };
};
