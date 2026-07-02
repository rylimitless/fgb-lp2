import type { PageServerLoad } from "./$types";
import { apiFetch } from "$lib/server/api";

export const load: PageServerLoad = async (event) => {
  const [
    docsRes,
    coursesRes,
    reviewDocsRes,
    reviewCoursesRes,
    streakRes,
    scoreRes,
    dashRes,
    enrollmentsRes,
    certsRes,
    badgesCountRes,
  ] = await Promise.all([
    apiFetch(event, "/api/documents"),
    apiFetch(event, "/api/courses"),
    apiFetch(event, "/api/review/documents?limit=100"),
    apiFetch(event, "/api/review/courses?limit=100"),
    apiFetch(event, "/api/gamification/streak"),
    apiFetch(event, "/api/gamification/score"),
    apiFetch(event, "/api/dashboard"),
    apiFetch(event, "/api/enrollments"),
    apiFetch(event, "/api/certificates"),
    apiFetch(event, "/api/badges/earned-count"),
  ]);

  const docs = docsRes.ok ? await docsRes.json() : [];
  const courses = coursesRes.ok ? await coursesRes.json() : [];
  const reviewDocsData = reviewDocsRes.ok
    ? await reviewDocsRes.json()
    : { items: [], total: 0 };
  const reviewCoursesData = reviewCoursesRes.ok
    ? await reviewCoursesRes.json()
    : { items: [], total: 0 };

  let streakDays = 0;
  let totalScore = 0;
  try {
    if (streakRes.ok) {
      const sData = await streakRes.json();
      streakDays = sData.streak_days ?? 0;
    }
  } catch {
    /* ignore */
  }
  try {
    if (scoreRes.ok) {
      const scData = await scoreRes.json();
      totalScore = scData.total_score ?? 0;
    }
  } catch {
    /* ignore */
  }

  let theta = 0;
  try {
    const pref = await apiFetch(event, "/api/adaptive/start?course_id=1");
    if (pref.ok) {
      const data = await pref.json();
      theta = data.theta ?? 0;
    }
  } catch {
    /* ignore */
  }

  // Dashboard aggregated data
  let dashboard = null;
  try {
    if (dashRes.ok) {
      dashboard = await dashRes.json();
    }
  } catch {
    /* ignore */
  }

  const approvedDocs = docs.filter((d: any) => d.approved);
  const publishedCourses = courses.filter(
    (c: any) => c.status === "published" && c.approved,
  );

  return {
    stats: {
      totalDocs: docs.length,
      approvedDocs: approvedDocs.length,
      totalCourses: courses.length,
      publishedCourses: publishedCourses.length,
      pendingReview: reviewDocsData.total + reviewCoursesData.total,
      theta,
    },
    recentDocs: docs.slice(0, 5),
    recentCourses: courses.slice(0, 5),
    gamification: {
      streakDays,
      totalScore,
    },
    dashboard,
    enrollments: enrollmentsRes.ok ? await enrollmentsRes.json() : [],
    certificates: certsRes.ok ? await certsRes.json() : [],
    badgesEarnedCount: badgesCountRes.ok
      ? (((await badgesCountRes.json()) as any).count ?? 0)
      : 0,
  };
};
