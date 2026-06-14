import type { PageServerLoad } from "./$types";

export const load: PageServerLoad = async ({ fetch }) => {
    const [coursesRes, docsRes] = await Promise.all([
        fetch("/api/courses"),
        fetch("/api/documents"),
    ]);

    const courses = coursesRes.ok ? await coursesRes.json() : [];
    const documents = docsRes.ok ? await docsRes.json() : [];

    return { courses, documents };
};
