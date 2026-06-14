import type { LayoutServerLoad } from "./$types";

export const load: LayoutServerLoad = async ({ fetch }) => {
  try {
    const res = await fetch("/api/me");
    if (res.ok) {
      const user = await res.json();
      return {
        user: {
          id: user.id as number,
          email: user.email as string,
          name: user.name as string,
          role: user.role as string,
          roles: (user.roles ?? [user.role]) as string[],
        },
      };
    }
  } catch {
    // User not authenticated or fetch failed — hooks.server will redirect to login
  }
  return { user: null };
};
