import type { PageServerLoad } from "./$types";
import { apiFetch } from "$lib/server/api";

export const load: PageServerLoad = async (event) => {
  const res = await apiFetch(event, "/api/documents");

  if (!res.ok) {
    return { documents: [] };
  }

  const documents = await res.json();
  return { documents };
};
