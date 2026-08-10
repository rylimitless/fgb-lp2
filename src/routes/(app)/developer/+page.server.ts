import type { PageServerLoad } from "./$types";
import { apiFetch } from "$lib/server/api";

export type TokenUsageAggregate = {
  source: string;
  calls: number;
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
  cached_prompt_tokens: number;
  reasoning_tokens: number;
};

export type TokenUsageRow = {
  id: number;
  model: string;
  source: string;
  label: string;
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
  cached_prompt_tokens: number;
  reasoning_tokens: number;
  course_id: number | null;
  user_id: number | null;
  job_ref: string | null;
  created_at: string;
};

export type TokenUsageResponse = {
  by_source: TokenUsageAggregate[];
  by_job: TokenUsageAggregate[];
  recent: TokenUsageRow[];
  totals: TokenUsageAggregate;
  mock_jobs: number;
};

export const load: PageServerLoad = async (event) => {
  const res = await apiFetch(event, "/api/dev/token-usage");
  const tokenUsage: TokenUsageResponse | null = res.ok
    ? await res.json()
    : null;
  return { tokenUsage };
};
