import type { PageServerLoad } from "./$types";
import { apiFetch } from "$lib/server/api";

export type LearningPath = {
    id: number;
    title: string;
    description: string;
    status: string;
    created_at: string;
};

export type PathEnrollment = {
    id: number;
    user_id: number;
    learning_path_id: number;
    status: string;
    progress_pct: string;
    started_at: string;
    completed_at: string | null;
    path_title: string;
    path_description: string;
};

export type PathDetail = {
    path: LearningPath;
    courses: PathDetailCourse[];
    enrollment: PathEnrollment | null;
    progress?: {
        total_courses: number;
        completed_courses: number;
        required_total: number;
        required_completed: number;
        progress_pct: number;
        is_complete: boolean;
    } | null;
};

export type PathDetailCourse = {
    id: number;
    learning_path_id: number;
    course_id: number;
    sort_order: number;
    is_required: boolean;
    course_title: string;
    course_description: string;
    course_status: string;
    user_status?: "none" | "active" | "completed";
    user_progress?: string;
};

export const load: PageServerLoad = async (event) => {
    // Fetch published paths
    const pathsRes = await apiFetch(event, "/api/learning-paths");
    const paths: LearningPath[] = pathsRes.ok ? await pathsRes.json() : [];

    // Fetch user's enrollments
    const enrRes = await apiFetch(event, "/api/learning-paths/my-enrollments");
    const myEnrollments: PathEnrollment[] = enrRes.ok ? await enrRes.json() : [];

    return { paths, myEnrollments };
};
