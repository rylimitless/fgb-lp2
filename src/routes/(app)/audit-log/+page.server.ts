import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch, url }) => {
  const limit = url.searchParams.get('limit') ?? '50';
  const offset = url.searchParams.get('offset') ?? '0';
  const action = url.searchParams.get('action') ?? '';

  const res = await fetch(
    `/api/analytics/audit-log?limit=${limit}&offset=${offset}${action ? `&action=${action}` : ''}`
  );

  const auditLog = res.ok ? await res.json() : [];

  return {
    auditLog,
    filters: { limit, offset, action },
  };
};
