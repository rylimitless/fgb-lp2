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

export const load: PageServerLoad = async (event) => {
  const pathsRes = await apiFetch(event, "/api/admin/learning-paths");
  const paths: LearningPath[] = pathsRes.ok ? await pathsRes.json() : [];

  // Also fetch published courses for the "add course" picker
  const coursesRes = await apiFetch(event, "/api/courses/published");
  const courses: PublishedCourse[] = coursesRes.ok
    ? await coursesRes.json()
    : [];

  return { paths, publishedCourses: courses };
};
