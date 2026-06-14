import type { PageServerLoad } from "./$types";

export const load: PageServerLoad = async ({ fetch }) => {
  const [docsRes, coursesRes, reviewDocsRes, reviewCoursesRes] =
    await Promise.all([
      fetch("/api/documents"),
      fetch("/api/courses"),
      fetch("/api/review/documents?limit=100"),
      fetch("/api/review/courses?limit=100"),
    ]);

  const docs = docsRes.ok ? await docsRes.json() : [];
  const courses = coursesRes.ok ? await coursesRes.json() : [];
  const reviewDocsData = reviewDocsRes.ok
    ? await reviewDocsRes.json()
    : { items: [], total: 0 };
  const reviewCoursesData = reviewCoursesRes.ok
    ? await reviewCoursesRes.json()
    : { items: [], total: 0 };

  let theta = 0;
  try {
    const pref = await fetch("/api/adaptive/start?course_id=1");
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
  };
};
