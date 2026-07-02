import type { PageServerLoad } from "./$types";
import { apiFetch } from "$lib/server/api";

export const load: PageServerLoad = async (event) => {
  const code = event.params.code;

  try {
    const res = await apiFetch(event, `/api/certificates/${code}`);
    if (res.ok) {
      return { certificate: await res.json() };
    }
  } catch {
    // fall through to null
  }

  return { certificate: null };
};
