import type { PageServerLoad } from "./$types";
import { apiFetch } from "$lib/server/api";
import { error } from "@sveltejs/kit";

export const load: PageServerLoad = async (event) => {
    const id = event.params.id;
    const res = await apiFetch(event, `/api/courses/${id}`);
    if (!res.ok) {
        if (res.status === 404) {
            throw error(404, "Course not found");
        }
        throw error(res.status, "Failed to load course");
    }
    const course = await res.json();
    return { course };
};
