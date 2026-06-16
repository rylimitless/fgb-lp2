import type { PageServerLoad } from "./$types";
import { apiFetch } from "$lib/server/api";

export const load: PageServerLoad = async (event) => {
  const [coursesRes, docsRes] = await Promise.all([
    apiFetch(event, "/api/courses"),
    apiFetch(event, "/api/documents"),
  ]);

  const courses = coursesRes.ok ? await coursesRes.json() : [];
  const documents = docsRes.ok ? await docsRes.json() : [];

  return { courses, documents };
};
