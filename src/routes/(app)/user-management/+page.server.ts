import type { PageServerLoad } from "./$types";

export type User = {
  id: number;
  email: string;
  name: string;
  role: string;
  created_at: string;
};

export const load: PageServerLoad = async ({ fetch }) => {
  const res = await fetch("/api/admin/users");
  const users: User[] = res.ok ? await res.json() : [];

  return { users };
};
