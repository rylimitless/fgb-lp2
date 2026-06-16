import type { RequestEvent } from "@sveltejs/kit";

const INTERNAL_BACKEND_URL =
  process.env.INTERNAL_BACKEND_URL ?? "http://backend:5555";

export async function apiFetch(
  event: Pick<RequestEvent, "cookies">,
  path: string,
  init: RequestInit = {},
): Promise<Response> {
  const url = `${INTERNAL_BACKEND_URL}${path.startsWith("/") ? path : `/${path}`}`;

  const headers = new Headers(init.headers ?? {});
  const sessionToken = event.cookies.get("session_token");
  if (sessionToken && !headers.has("Cookie")) {
    headers.set("Cookie", `session_token=${sessionToken}`);
  }

  return fetch(url, { ...init, headers });
}
