import { redirect } from "@sveltejs/kit";
import type { PageServerLoad } from "./$types";

// The legacy one-shot AI generator has been replaced by the staged Course
// Builder. Redirect any lingering bookmarks/nav entries there.
export const load: PageServerLoad = async () => {
  throw redirect(307, "/course-builder");
};
