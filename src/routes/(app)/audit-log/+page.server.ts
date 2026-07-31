import type { PageServerLoad } from "./$types";
import { apiFetch } from "$lib/server/api";

export type AuditLogEntry = {
  id: number;
  user_id: number | null;
  user_name: string | null;
  user_role: string | null;
  action: string;
  created_at: string;
  details: Record<string, any> | null;
};

export const load: PageServerLoad = async (event) => {
  const limit = event.url.searchParams.get("limit") ?? "50";
  const offset = event.url.searchParams.get("offset") ?? "0";
  const action = event.url.searchParams.get("action") ?? "";

  const res = await apiFetch(
    event,
    `/api/analytics/audit-log?limit=${limit}&offset=${offset}${action ? `&action=${action}` : ""}`,
  );

  const auditLog: AuditLogEntry[] = res.ok ? await res.json() : [];

  return {
    auditLog,
    filters: { limit, offset, action },
  };
};
