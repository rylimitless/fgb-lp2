import type { PageServerLoad } from "./$types";
import { apiFetch } from "$lib/server/api";

async function fetchWithRetry(
    event: Parameters<typeof apiFetch>[0],
    path: string,
    maxRetries = 5,
): Promise<Response> {
    let lastErr: unknown;
    for (let i = 0; i < maxRetries; i++) {
        try {
            const res = await apiFetch(event, path);
            if (res.ok) return res;
            // Non-connection errors (e.g. 401) — don't retry
            return res;
        } catch (err) {
            lastErr = err;
            if (i < maxRetries - 1) {
                await new Promise((r) => setTimeout(r, 1000 * (i + 1)));
            }
        }
    }
    throw lastErr;
}

export const load: PageServerLoad = async (event) => {
    // Surface only draft courses here: those are the ones still being
    // shaped in the builder. Published/archived courses are managed from
    // the review queue / content repository.
    const [coursesRes, docsRes] = await Promise.all([
        fetchWithRetry(event, "/api/courses"),
        fetchWithRetry(event, "/api/documents"),
    ]);

    const all = coursesRes.ok ? await coursesRes.json() : [];
    const documents = docsRes.ok ? await docsRes.json() : [];
    const drafts = all.filter(
        (c: any) => c.status === "draft" || c.status === undefined,
    );
    return { drafts, documents };
};
