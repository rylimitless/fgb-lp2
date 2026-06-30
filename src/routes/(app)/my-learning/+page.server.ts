import type { PageServerLoad } from "./$types";
import { apiFetch } from "$lib/server/api";

export const load: PageServerLoad = async (event) => {
	let enrollments: any[] = [];

	try {
		const res = await apiFetch(event, "/api/enrollments");
		if (res.ok) {
			enrollments = await res.json();
		}
	} catch {
		// silently empty
	}

	return { enrollments };
};
