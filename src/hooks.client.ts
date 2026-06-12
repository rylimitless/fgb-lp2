import { redirect } from "@sveltejs/kit";
import type { HandleFetch } from "@sveltejs/kit";

export const handleFetch: HandleFetch = async ({ event, request, fetch }) => {
  const response = await fetch(request);

  if (response.status === 401 && !event.url.pathname.startsWith("/login")) {
    throw redirect(307, "/login");
  }

  return response;
};
