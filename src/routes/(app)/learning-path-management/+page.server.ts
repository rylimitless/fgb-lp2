import type { PageServerLoad } from "./$types";
import { apiFetch } from "$lib/server/api";

export type LearningPath = {
    id: number;
    title: string;
    description: string;
    created_by: number;
    status: string;
    created_at: string;
    updated_at: string;
};

export type PathCourse = {
    id: number;
    learning_path_id: number;
    course_id: number;
    sort_order: number;
    is_required: boolean;
    course_title: string;
    course_description: string;
    course_status: string;
};

export type PublishedCourse = {
    id: number;
    title: string;
    description: string;
    status: string;
    source_doc_ids?: number[];
    settings?: any;
};

export type User = {
    id: number;
    email: string;
    name: string;
    role: string;
    roles?: string[];
};

export const load: PageServerLoad = async (event) => {
    const pathsRes = await apiFetch(event, "/api/admin/learning-paths");
    const paths: LearningPath[] = pathsRes.ok ? await pathsRes.json() : [];
    return { paths };
};
