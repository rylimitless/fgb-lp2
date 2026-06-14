import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch }) => {
  const [overviewRes, failedRes, courseEffRes, coachUsageRes] =
    await Promise.all([
      fetch('/api/analytics/overview'),
      fetch('/api/analytics/most-failed'),
      fetch('/api/analytics/course-effectiveness'),
      fetch('/api/analytics/coach-usage'),
    ]);

  const overview = overviewRes.ok ? await overviewRes.json() : null;
  const mostFailed = failedRes.ok ? await failedRes.json() : [];
  const courseEffectiveness = courseEffRes.ok
    ? await courseEffRes.json()
    : [];
  const coachUsage = coachUsageRes.ok ? await coachUsageRes.json() : [];

  return {
    overview,
    mostFailed,
    courseEffectiveness,
    coachUsage,
  };
};
