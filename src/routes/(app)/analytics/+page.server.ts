import type { PageServerLoad } from "./$types";

export const load: PageServerLoad = async ({ fetch }) => {
  const [overviewRes, failedRes, courseEffRes, coachUsageRes, auditRes] =
    await Promise.all([
      fetch("/api/analytics/overview"),
      fetch("/api/analytics/most-failed"),
      fetch("/api/analytics/course-effectiveness"),
      fetch("/api/analytics/coach-usage"),
      fetch("/api/analytics/audit-log?limit=100"),
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
