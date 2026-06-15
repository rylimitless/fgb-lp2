import type { LayoutServerLoad } from "./$types";

export const load: LayoutServerLoad = async ({ fetch, cookies }) => {
  const sessionToken = cookies.get("session_token");

  try {
    const res = await fetch("/api/me", {
      headers: sessionToken ? { cookie: `session_token=${sessionToken}` } : {},
    });
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
  } catch (e) {
    console.error("[layout] /api/me failed:", e);
  }
  return { user: null };
};
