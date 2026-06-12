import type { PageServerLoad } from "./$types";

export const load: PageServerLoad = async ({ fetch }) => {
  const res = await fetch("/api/documents");

  if (!res.ok) {
    return { documents: [] };
  }

  const documents = await res.json();
  return { documents };
};
