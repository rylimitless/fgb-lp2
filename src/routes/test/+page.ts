import type { PageLoad } from "./$types";
import { redirect } from "@sveltejs/kit";

export const load = async ({ fetch, params }) => {
  
  const res = await fetch("http://localhost:5555/check");
  
  if (res.status === 401) {
    throw redirect(307, "/login");
  }
  
  const data = await res.json();
  return { message: data.Message };
};
