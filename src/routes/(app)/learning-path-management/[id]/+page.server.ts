import type { PageServerLoad } from "./$types";
import { apiFetch } from "$lib/server/api";
import { error } from "@sveltejs/kit";
import type {
    LearningPath,
    PathCourse,
    PublishedCourse,
    User,
} from "../+page.server";

export type PathEnrollment = {
    id: number;
    user_id: number;
    user_name: string;
    user_email: string;
    status: string;
    progress_pct: string;
    started_at: string;
    completed_at: string | null;
    dropped_at: string | null;
};

export type Department = {
    id: number;
    name: string;
};

export const load: PageServerLoad = async (event) => {
    const id = event.params.id;

    const pathRes = await apiFetch(event, `/api/admin/learning-paths/${id}`);
    if (!pathRes.ok) {
        if (pathRes.status === 404) {
            throw error(404, "Learning path not found");
        }
        throw error(pathRes.status, "Failed to load learning path");
    }
    const path: LearningPath = await pathRes.json();

    const [coursesRes, enrollmentsRes, publishedRes, usersRes, deptsRes] =
        await Promise.all([
            apiFetch(event, `/api/admin/learning-paths/${id}/courses`),
            apiFetch(event, `/api/admin/learning-paths/${id}/enrollments`),
            apiFetch(event, "/api/courses/published"),
            apiFetch(event, "/api/admin/users"),
            apiFetch(event, "/api/admin/departments"),
        ]);

    const courses: PathCourse[] = coursesRes.ok ? await coursesRes.json() : [];
    const enrollments: PathEnrollment[] = enrollmentsRes.ok
        ? await enrollmentsRes.json()
        : [];
    const publishedCourses: PublishedCourse[] = publishedRes.ok
        ? await publishedRes.json()
        : [];
    const users: User[] = usersRes.ok ? await usersRes.json() : [];
    const departments: Department[] = deptsRes.ok ? await deptsRes.json() : [];

    return { path, courses, enrollments, publishedCourses, users, departments };
};
