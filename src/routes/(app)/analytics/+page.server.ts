import type { PageServerLoad } from "./$types";
import { apiFetch } from "$lib/server/api";

export const load: PageServerLoad = async (event) => {
  // Optional date-range window for time-series charts (7/30/90/365), default 30.
  const daysParam = event.url.searchParams.get("days");
  const days = ["7", "30", "90", "365"].includes(daysParam ?? "")
    ? (daysParam as string)
    : "30";

  const [
    overviewRes,
    failedRes,
    courseEffRes,
    coachUsageRes,
    auditRes,
    teamOverviewRes,
    strugglingRes,
    completionRatesRes,
    enrollmentOverviewRes,
    enrollmentTimelineRes,
    stalledRes,
    credentialOverviewRes,
    credentialTimelineRes,
    userOverviewRes,
    userGrowthRes,
    learningPathOverviewRes,
    contentOverviewRes,
    engagementOverviewRes,
    courseBreakdownRes,
  ] = await Promise.all([
    apiFetch(event, "/api/analytics/overview"),
    apiFetch(event, "/api/analytics/most-failed"),
    apiFetch(event, "/api/analytics/course-effectiveness"),
    apiFetch(event, "/api/analytics/coach-usage?days=" + days),
    apiFetch(event, "/api/analytics/audit-log?limit=100"),
    apiFetch(event, "/api/analytics/team-overview"),
    apiFetch(event, "/api/analytics/struggling-learners"),
    apiFetch(event, "/api/analytics/completion-rates"),
    apiFetch(event, "/api/analytics/enrollments/overview"),
    apiFetch(event, "/api/analytics/enrollments/timeline?days=" + days),
    apiFetch(event, "/api/analytics/enrollments/stalled"),
    apiFetch(event, "/api/analytics/credentials/overview"),
    apiFetch(event, "/api/analytics/credentials/timeline?days=" + days),
    apiFetch(event, "/api/analytics/users/overview"),
    apiFetch(event, "/api/analytics/users/growth?days=" + days),
    apiFetch(event, "/api/analytics/learning-paths/overview"),
    apiFetch(event, "/api/analytics/content/overview"),
    apiFetch(event, "/api/analytics/engagement/overview"),
    apiFetch(event, "/api/analytics/courses/breakdown"),
  ]);

  const overview = overviewRes.ok ? await overviewRes.json() : null;
  const mostFailed = failedRes.ok ? await failedRes.json() : [];
  const courseEffectiveness = courseEffRes.ok ? await courseEffRes.json() : [];
  const coachUsage = coachUsageRes.ok ? await coachUsageRes.json() : [];
  const auditLog = auditRes.ok ? await auditRes.json() : [];
  const teamOverview = teamOverviewRes.ok ? await teamOverviewRes.json() : [];
  const strugglingLearners = strugglingRes.ok ? await strugglingRes.json() : [];
  const completionRates = completionRatesRes.ok ? await completionRatesRes.json() : [];
  const enrollmentOverview = enrollmentOverviewRes.ok ? await enrollmentOverviewRes.json() : null;
  const enrollmentTimeline = enrollmentTimelineRes.ok ? await enrollmentTimelineRes.json() : [];
  const stalledEnrollments = stalledRes.ok ? await stalledRes.json() : [];
  const credentialOverview = credentialOverviewRes.ok ? await credentialOverviewRes.json() : null;
  const credentialTimeline = credentialTimelineRes.ok ? await credentialTimelineRes.json() : [];
  const userOverview = userOverviewRes.ok ? await userOverviewRes.json() : null;
  const userGrowth = userGrowthRes.ok ? await userGrowthRes.json() : [];
  const learningPathOverview = learningPathOverviewRes.ok ? await learningPathOverviewRes.json() : null;
  const contentOverview = contentOverviewRes.ok ? await contentOverviewRes.json() : null;
  const engagementOverview = engagementOverviewRes.ok ? await engagementOverviewRes.json() : null;
  const courseBreakdown = courseBreakdownRes.ok ? await courseBreakdownRes.json() : [];

  return {
    overview,
    mostFailed,
    courseEffectiveness,
    coachUsage,
    auditLog,
    teamOverview,
    strugglingLearners,
    completionRates,
    enrollmentOverview,
    enrollmentTimeline,
    stalledEnrollments,
    credentialOverview,
    credentialTimeline,
    userOverview,
    userGrowth,
    learningPathOverview,
    contentOverview,
    engagementOverview,
    courseBreakdown,
    days,
  };
};
