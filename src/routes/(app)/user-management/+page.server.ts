import type { PageServerLoad } from "./$types";
import { apiFetch } from "$lib/server/api";

export type User = {
  id: number;
  email: string;
  name: string;
  role: string;
  roles?: string[];
  created_at: string;
};

export const load: PageServerLoad = async (event) => {
  const res = await apiFetch(event, "/api/admin/users");
  const users: User[] = res.ok ? await res.json() : [];

  return { users };
};
