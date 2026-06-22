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
  ] = await Promise.all([
    apiFetch(event, "/api/documents"),
    apiFetch(event, "/api/courses"),
    apiFetch(event, "/api/review/documents?limit=100"),
    apiFetch(event, "/api/review/courses?limit=100"),
    apiFetch(event, "/api/gamification/streak"),
    apiFetch(event, "/api/gamification/score"),
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
  };
};
