import type { PageServerLoad } from './$types';
import { apiFetch } from '$lib/server/api';

export const load: PageServerLoad = async (event) => {
  const [certRes, earnedRes, countRes] = await Promise.all([
    apiFetch(event, '/api/certificates'),
    apiFetch(event, '/api/badges/earned'),
    apiFetch(event, '/api/badges/earned-count'),
  ]);

  const certificates = certRes.ok ? await certRes.json() : [];
  const earnedBadges = earnedRes.ok ? await earnedRes.json() : [];

  let earnedCount = 0;
  try {
    if (countRes.ok) {
      const c = await countRes.json();
      earnedCount = c.count ?? 0;
    }
  } catch {
    /* ignore */
  }

  return {
    certificates,
    earnedBadges,
    earnedCount,
  };
};
