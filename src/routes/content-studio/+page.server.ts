import type { PageServerLoad } from "./$types";

export const load: PageServerLoad = async ({ fetch }) => {
  const res = await fetch("http://localhost:5555/api/documents");

  if (!res.ok) {
    return { documents: [] };
  }

  const documents = await res.json();
  return { documents };
};
