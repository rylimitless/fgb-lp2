import type { PageServerLoad } from "./$types";
import { apiFetch } from "$lib/server/api";
import type { PublishedCourse } from "../+page.server";

export const load: PageServerLoad = async (event) => {
    // Pre-fetch the approved course catalog so the builder can show titles and
    // counts immediately without waiting for the generation call.
    const res = await apiFetch(event, "/api/courses/published");
    const catalog: PublishedCourse[] = res.ok ? await res.json() : [];
    return { catalog };
};
