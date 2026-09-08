<script lang="ts">
    import {
        Beaker,
        Coins,
        Gauge,
        ArrowDownToLine,
        ArrowUpFromLine,
        Brain,
        LoaderCircle,
        Play,
        Square,
        Trash2,
        AlertTriangle,
    } from "@lucide/svelte";
    import { PageHeader, StatCard, showToast } from "$lib/components/brand";
    import * as Button from "$lib/components/ui/button";
    import { invalidateAll } from "$app/navigation";

    let { data }: { data: any } = $props();

    // ---- Pricing model -------------------------------------------------
    // Tokens come from the SDK verbatim; spend is computed in the browser so
    // deployments can override rates without a backend redeploy. Defaults are
    // DeepSeek's published deepseek-chat pricing (USD per million tokens) as of
    // 2024-2025. Cached prompt tokens get the cache-hit discount.
    let inputRate = $state(0.27); // $/M input tokens (cache miss)
    let cachedRate = $state(0.07); // $/M cached prompt tokens
    let outputRate = $state(1.1); // $/M output tokens

    // ---- Mock generator state -----------------------------------------
    type Preset = {
        id: string;
        label: string;
        description: string;
        modules: number;
        items_per_module: number;
        sections_per_module: number;
    };

    const PRESETS: Preset[] = [
        {
            id: "tiny",
            label: "Tiny",
            description: "1 module · 1 section · 2 items",
            modules: 1,
            sections_per_module: 1,
            items_per_module: 2,
        },
        {
            id: "small",
            label: "Small",
            description: "2 modules · 2 sections · 3 items",
            modules: 2,
            sections_per_module: 2,
            items_per_module: 3,
        },
        {
            id: "medium",
            label: "Medium",
            description: "4 modules · 3 sections · 5 items",
            modules: 4,
            sections_per_module: 3,
            items_per_module: 5,
        },
        {
            id: "large",
            label: "Large",
            description: "6 modules · 4 sections · 8 items",
            modules: 6,
            sections_per_module: 4,
            items_per_module: 8,
        },
        {
            id: "huge",
            label: "Huge",
            description: "30 modules · 5 sections · 5 items (~300 items)",
            modules: 30,
            sections_per_module: 5,
            items_per_module: 5,
        },
    ];

    let selectedPreset = $state("small");
    let generating = $state(false);
    let abortCtrl: AbortController | null = null;
    let runError = $state("");
    let liveSteps = $state<string[]>([]);
    let liveTokens = $state<
        {
            label: string;
            prompt: number;
            completion: number;
            total: number;
            running: number;
        }[]
    >([]);
    let lastRun = $state<{
        preset: string;
        course_id: number;
        llm_calls: number;
        prompt: number;
        completion: number;
        total: number;
        cached: number;
        reasoning: number;
        seconds: number;
        modules: number;
        items: number;
        cost: number;
    } | null>(null);

    // ---- Derived from server data -------------------------------------
    let tokenUsage = $derived(data.tokenUsage);
    let totals = $derived(tokenUsage?.totals);
    let bySource = $derived(tokenUsage?.by_source ?? []);
    let byJob = $derived(tokenUsage?.by_job ?? []);
    let recent = $derived(tokenUsage?.recent ?? []);
    let mockJobs = $derived(tokenUsage?.mock_jobs ?? 0);

    // Average spend per mock course (across ALL mock runs, not just the last).
    let mockTotals = $derived(
        bySource.find((s: any) => s.source === "mock_course") ?? {
            source: "mock_course",
            calls: 0,
            prompt_tokens: 0,
            completion_tokens: 0,
            total_tokens: 0,
            cached_prompt_tokens: 0,
            reasoning_tokens: 0,
        },
    );
    let avgPerMock = $derived(
        mockJobs > 0
            ? {
                  prompt: mockTotals.prompt_tokens / mockJobs,
                  completion: mockTotals.completion_tokens / mockJobs,
                  total: mockTotals.total_tokens / mockJobs,
                  calls: mockTotals.calls / mockJobs,
                  cost:
                      computeCost(
                          mockTotals.prompt_tokens,
                          mockTotals.completion_tokens,
                          mockTotals.cached_prompt_tokens,
                      ) / mockJobs,
              }
            : null,
    );

    // ---- Helpers -------------------------------------------------------
    function fmt(n: number | undefined | null): string {
        if (n === undefined || n === null) return "—";
        return n.toLocaleString(undefined, { maximumFractionDigits: 0 });
    }
    function money(n: number): string {
        if (!isFinite(n)) return "—";
        if (n < 0.01) return `$${n.toFixed(4)}`;
        return `$${n.toFixed(2)}`;
    }
    function computeCost(
        prompt: number,
        completion: number,
        cached: number,
    ): number {
        const nonCachedPrompt = Math.max(0, prompt - cached);
        return (
            (nonCachedPrompt * inputRate) / 1_000_000 +
            (cached * cachedRate) / 1_000_000 +
            (completion * outputRate) / 1_000_000
        );
    }
    function totalCost(calls: {
        prompt_tokens: number;
        completion_tokens: number;
        cached_prompt_tokens: number;
    }): number {
        return computeCost(
            calls.prompt_tokens,
            calls.completion_tokens,
            calls.cached_prompt_tokens,
        );
    }

    // ---- Run a mock course --------------------------------------------
    async function runMock() {
        if (generating) return;
        generating = true;
        runError = "";
        liveSteps = [];
        liveTokens = [];
        lastRun = null;
        abortCtrl = new AbortController();

        try {
            const res = await fetch(
                `/api/dev/mock-course/generate?preset=${encodeURIComponent(selectedPreset)}`,
                {
                    method: "GET",
                    credentials: "include",
                    signal: abortCtrl.signal,
                },
            );
            if (!res.ok || !res.body) {
                const err = await res.json().catch(() => ({}));
                runError = err.error || `Failed to start (HTTP ${res.status})`;
                generating = false;
                return;
            }
            const reader = res.body.getReader();
            const decoder = new TextDecoder();
            let buffer = "";
            let eventType = "";

            while (true) {
                const { done, value } = await reader.read();
                if (done) break;
                buffer += decoder.decode(value, { stream: true });
                const lines = buffer.split("\n");
                buffer = lines.pop() || "";

                for (const line of lines) {
                    if (line.startsWith("event: ")) {
                        eventType = line.slice(7).trim();
                    } else if (line.startsWith("data: ")) {
                        let p: any;
                        try {
                            p = JSON.parse(line.slice(6));
                        } catch {
                            continue;
                        }
                        handleEvent(eventType, p);
                        eventType = "";
                    }
                }
            }
        } catch (e: any) {
            if (e?.name === "AbortError") {
                runError = "Run cancelled";
            } else {
                runError = "Network error — is the backend reachable?";
            }
        } finally {
            generating = false;
            abortCtrl = null;
            // Refresh server-side aggregates so the dashboard reflects the
            // run we just persisted.
            await invalidateAll();
        }
    }

    function handleEvent(type: string, p: any) {
        if (type === "step") {
            liveSteps = [...liveSteps, p.detail];
        } else if (type === "tokens") {
            liveTokens = [
                ...liveTokens,
                {
                    label: p.label,
                    prompt: p.prompt_tokens,
                    completion: p.completion_tokens,
                    total: p.total_tokens,
                    running: p.running_total,
                },
            ];
        } else if (type === "outline") {
            liveSteps = [
                ...liveSteps,
                `Created course #${p.course_id} — ${p.module_count} modules`,
            ];
        } else if (type === "error") {
            runError = p.message || "Run failed";
        } else if (type === "done") {
            const cost = computeCost(
                p.prompt_tokens,
                p.completion_tokens,
                p.cached_prompt_tokens,
            );
            lastRun = {
                preset: p.preset_id,
                course_id: p.course_id,
                llm_calls: p.llm_calls,
                prompt: p.prompt_tokens,
                completion: p.completion_tokens,
                total: p.total_tokens,
                cached: p.cached_prompt_tokens,
                reasoning: p.reasoning_tokens,
                seconds: p.duration_seconds,
                modules: p.modules_created,
                items: p.items_created,
                cost,
            };
            showToast(`Mock course complete · ${money(cost)} · ${fmt(p.total_tokens)} tokens`, {
                title: "Mock course generated",
                variant: "success",
            });
        }
    }

    function cancelMock() {
        if (abortCtrl) abortCtrl.abort();
    }
</script>

<svelte:head><title>Developer Options · FGB Academy</title></svelte:head>

<div class="mx-auto flex w-full max-w-7xl flex-col gap-6">
    <PageHeader
        title="Developer options"
        eyebrow="Internal tooling"
        description="Mock-course generator and LLM spend dashboard. Every chat completion is logged with the provider's own token accounting — use this to benchmark cost per course and per source."
    >
        {#snippet icon()}
            <Beaker class="size-6 text-primary" />
        {/snippet}
    </PageHeader>

    <!-- Headline stats -->
    <section
        class="grid grid-cols-2 gap-3 md:grid-cols-4 motion-rise-in"
    >
        <StatCard
            label="Total tokens (all sources)"
            value={totals?.total_tokens ?? 0}
            tone="primary"
            hint={`${fmt(totals?.calls)} LLM calls tracked`}
        />
        <StatCard
            label="Input tokens"
            value={totals?.prompt_tokens ?? 0}
            tone="info"
            hint={`${fmt(totals?.cached_prompt_tokens)} from cache`}
        />
        <StatCard
            label="Output tokens"
            value={totals?.completion_tokens ?? 0}
            tone="accent"
            hint={`${fmt(totals?.reasoning_tokens)} reasoning`}
        />
        <StatCard
            label="Estimated total spend"
            value={totalCost({
                prompt_tokens: totals?.prompt_tokens ?? 0,
                completion_tokens: totals?.completion_tokens ?? 0,
                cached_prompt_tokens: totals?.cached_prompt_tokens ?? 0,
            })}
            tone="success"
            prefix="$"
            decimals={2}
            hint="At current rates below"
        />
    </section>

    <!-- Mock generator -->
    <section
        class="rounded-2xl border border-border bg-card p-5 lift motion-rise-in motion-stagger-1"
    >
        <header class="mb-4 flex items-start gap-3">
            <span
                class="flex size-9 items-center justify-center rounded-xl bg-primary-soft text-primary"
            >
                <Beaker class="size-5" />
            </span>
            <div class="flex-1">
                <h2 class="text-sm font-semibold text-foreground">
                    Mock course generator
                </h2>
                <p class="text-xs text-muted-foreground">
                    Generates a real course of a fixed shape using the live LLM.
                    Every call's token usage is recorded so each run is
                    comparable to the next.
                </p>
            </div>
        </header>

        <!-- Preset picker -->
        <div class="mb-4 grid grid-cols-2 gap-2 md:grid-cols-5">
            {#each PRESETS as p (p.id)}
                <button
                    type="button"
                    onclick={() => !generating && (selectedPreset = p.id)}
                    disabled={generating}
                    class="flex flex-col gap-1 rounded-xl border p-3 text-left transition-colors disabled:opacity-50
                        {selectedPreset === p.id
                        ? 'border-primary bg-primary/5'
                        : 'border-border-strong bg-surface-1 hover:border-primary/50'}"
                >
                    <span class="text-sm font-semibold text-foreground">
                        {p.label}
                    </span>
                    <span class="text-[11px] text-muted-foreground">
                        {p.description}
                    </span>
                    <span class="mt-1 text-[10px] uppercase tracking-wider text-muted-foreground/80">
                        {p.modules} modules · {p.modules * p.sections_per_module} sections · {p.modules * p.items_per_module} items
                    </span>
                </button>
            {/each}
        </div>

        <!-- Action row -->
        <div class="flex flex-wrap items-center gap-2">
            {#if !generating}
                <Button.Root onclick={runMock}>
                    <Play class="size-4" />
                    Generate mock course ({PRESETS.find((p) => p.id === selectedPreset)?.label})
                </Button.Root>
            {:else}
                <Button.Root variant="destructive" onclick={cancelMock}>
                    <Square class="size-4" />
                    Cancel run
                </Button.Root>
                <span class="inline-flex items-center gap-1.5 text-xs text-muted-foreground">
                    <LoaderCircle class="size-3 animate-spin" />
                    Running — watch the live log below
                </span>
            {/if}
            <span class="ml-auto text-[11px] text-muted-foreground/80">
                {mockJobs > 0
                    ? `${mockJobs} previous mock run${mockJobs === 1 ? "" : "s"} on record`
                    : "No mock runs yet"}
            </span>
        </div>

        {#if runError}
            <div
                class="mt-3 rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 flex items-center gap-2"
            >
                <AlertTriangle class="size-4 text-destructive shrink-0" />
                <p class="text-xs text-destructive">{runError}</p>
            </div>
        {/if}

        <!-- Live progress (during a run) -->
        {#if generating || liveSteps.length > 0}
            <div class="mt-4 grid grid-cols-1 gap-3 lg:grid-cols-2">
                <!-- Steps log -->
                <div
                    class="rounded-xl border border-border bg-surface-1 p-3 max-h-64 overflow-y-auto"
                >
                    <p class="mb-2 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
                        Progress log
                    </p>
                    <div class="flex flex-col gap-1">
                        {#each liveSteps.slice(-12) as s, i}
                            <p class="text-xs text-foreground/90">
                                <span class="text-muted-foreground/60">›</span>
                                {s}
                            </p>
                        {/each}
                    </div>
                </div>

                <!-- Per-call tokens -->
                <div
                    class="rounded-xl border border-border bg-surface-1 p-3 max-h-64 overflow-y-auto"
                >
                    <p class="mb-2 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
                        Per-call token usage
                    </p>
                    <div class="flex flex-col gap-1">
                        {#each liveTokens as t}
                            <div
                                class="flex items-center justify-between gap-2 text-[11px]"
                            >
                                <span class="truncate text-muted-foreground">
                                    {t.label}
                                </span>
                                <span class="flex items-center gap-2 font-mono">
                                    <span class="text-info" title="input">
                                        ↓{fmt(t.prompt)}
                                    </span>
                                    <span class="text-accent" title="output">
                                        ↑{fmt(t.completion)}
                                    </span>
                                    <span
                                        class="text-muted-foreground/70"
                                        title="running total"
                                    >
                                        Σ {fmt(t.running)}
                                    </span>
                                </span>
                            </div>
                        {/each}
                    </div>
                </div>
            </div>
        {/if}

        <!-- Last run summary -->
        {#if lastRun}
            <div
                class="mt-4 rounded-xl border border-success/30 bg-success/5 p-4"
            >
                <div class="mb-2 flex items-center gap-2">
                    <Coins class="size-4 text-success" />
                    <p class="text-sm font-semibold text-foreground">
                        Run complete · {lastRun.preset}
                    </p>
                    <span class="ml-auto text-sm font-bold text-success tabular">
                        {money(lastRun.cost)}
                    </span>
                </div>
                <div class="grid grid-cols-2 gap-2 text-xs md:grid-cols-4">
                    <div>
                        <p class="text-muted-foreground">Course</p>
                        <p class="font-medium text-foreground">
                            #{lastRun.course_id}
                        </p>
                    </div>
                    <div>
                        <p class="text-muted-foreground">LLM calls</p>
                        <p class="font-medium text-foreground tabular">
                            {fmt(lastRun.llm_calls)}
                        </p>
                    </div>
                    <div>
                        <p class="text-muted-foreground">Tokens (in / out)</p>
                        <p class="font-medium text-foreground tabular">
                            {fmt(lastRun.prompt)} / {fmt(lastRun.completion)}
                        </p>
                    </div>
                    <div>
                        <p class="text-muted-foreground">Total tokens</p>
                        <p class="font-medium text-foreground tabular">
                            {fmt(lastRun.total)}
                        </p>
                    </div>
                    <div>
                        <p class="text-muted-foreground">Duration</p>
                        <p class="font-medium text-foreground tabular">
                            {lastRun.seconds.toFixed(1)}s
                        </p>
                    </div>
                    <div>
                        <p class="text-muted-foreground">Modules / items</p>
                        <p class="font-medium text-foreground tabular">
                            {lastRun.modules} / {lastRun.items}
                        </p>
                    </div>
                    <div>
                        <p class="text-muted-foreground">Cached prompt</p>
                        <p class="font-medium text-foreground tabular">
                            {fmt(lastRun.cached)}
                        </p>
                    </div>
                    <div>
                        <p class="text-muted-foreground">Avg / call</p>
                        <p class="font-medium text-foreground tabular">
                            {money(lastRun.cost / Math.max(1, lastRun.llm_calls))}
                        </p>
                    </div>
                </div>
            </div>
        {/if}
    </section>

    <!-- Pricing controls + averages -->
    <section
        class="grid grid-cols-1 gap-4 lg:grid-cols-2 motion-rise-in motion-stagger-2"
    >
        <!-- Pricing -->
        <div class="rounded-2xl border border-border bg-card p-5 lift">
            <header class="mb-3 flex items-center gap-2">
                <Gauge class="size-4 text-muted-foreground" />
                <h3 class="text-sm font-semibold text-foreground">
                    Pricing model
                </h3>
            </header>
            <p class="mb-3 text-xs text-muted-foreground">
                Adjust to match your provider's published rates. Spend
                recalculates instantly.
            </p>
            <div class="flex flex-col gap-2">
                <label
                    class="flex items-center justify-between gap-3 rounded-lg border border-border-strong bg-surface-1 px-3 py-2 text-xs"
                >
                    <span class="flex items-center gap-1.5 text-muted-foreground">
                        <ArrowDownToLine class="size-3 text-info" />
                        Input ($/M tokens)
                    </span>
                    <input
                        type="number"
                        step="0.01"
                        min="0"
                        bind:value={inputRate}
                        class="w-20 rounded-md border border-border-strong bg-card px-2 py-1 text-right font-mono text-foreground outline-none focus-visible:ring-2 focus-visible:ring-ring/40"
                    />
                </label>
                <label
                    class="flex items-center justify-between gap-3 rounded-lg border border-border-strong bg-surface-1 px-3 py-2 text-xs"
                >
                    <span class="flex items-center gap-1.5 text-muted-foreground">
                        <Brain class="size-3 text-success" />
                        Cached input ($/M)
                    </span>
                    <input
                        type="number"
                        step="0.01"
                        min="0"
                        bind:value={cachedRate}
                        class="w-20 rounded-md border border-border-strong bg-card px-2 py-1 text-right font-mono text-foreground outline-none focus-visible:ring-2 focus-visible:ring-ring/40"
                    />
                </label>
                <label
                    class="flex items-center justify-between gap-3 rounded-lg border border-border-strong bg-surface-1 px-3 py-2 text-xs"
                >
                    <span class="flex items-center gap-1.5 text-muted-foreground">
                        <ArrowUpFromLine class="size-3 text-accent" />
                        Output ($/M tokens)
                    </span>
                    <input
                        type="number"
                        step="0.01"
                        min="0"
                        bind:value={outputRate}
                        class="w-20 rounded-md border border-border-strong bg-card px-2 py-1 text-right font-mono text-foreground outline-none focus-visible:ring-2 focus-visible:ring-ring/40"
                    />
                </label>
            </div>
        </div>

        <!-- Averages -->
        <div class="rounded-2xl border border-border bg-card p-5 lift">
            <header class="mb-3 flex items-center gap-2">
                <Coins class="size-4 text-muted-foreground" />
                <h3 class="text-sm font-semibold text-foreground">
                    Average per mock course
                </h3>
            </header>
            {#if avgPerMock}
                <div class="grid grid-cols-2 gap-3 md:grid-cols-3">
                    <div>
                        <p class="text-[11px] text-muted-foreground">
                            Avg tokens / course
                        </p>
                        <p class="text-lg font-bold text-foreground tabular">
                            {fmt(avgPerMock.total)}
                        </p>
                    </div>
                    <div>
                        <p class="text-[11px] text-muted-foreground">
                            Avg calls / course
                        </p>
                        <p class="text-lg font-bold text-foreground tabular">
                            {fmt(avgPerMock.calls)}
                        </p>
                    </div>
                    <div>
                        <p class="text-[11px] text-muted-foreground">
                            Avg spend / course
                        </p>
                        <p class="text-lg font-bold text-success tabular">
                            {money(avgPerMock.cost)}
                        </p>
                    </div>
                    <div>
                        <p class="text-[11px] text-muted-foreground">
                            Avg input / course
                        </p>
                        <p class="text-sm font-semibold text-info tabular">
                            ↓ {fmt(avgPerMock.prompt)}
                        </p>
                    </div>
                    <div>
                        <p class="text-[11px] text-muted-foreground">
                            Avg output / course
                        </p>
                        <p class="text-sm font-semibold text-accent tabular">
                            ↑ {fmt(avgPerMock.completion)}
                        </p>
                    </div>
                </div>
            {:else}
                <div
                    class="rounded-lg border border-dashed border-border bg-muted/20 px-4 py-6 text-center"
                >
                    <Beaker class="mx-auto mb-1 size-5 text-muted-foreground/60" />
                    <p class="text-xs text-muted-foreground">
                        Run your first mock course to see averages here.
                    </p>
                </div>
            {/if}
        </div>
    </section>

    <!-- Per-source breakdown -->
    <section
        class="rounded-2xl border border-border bg-card p-5 lift motion-rise-in motion-stagger-3"
    >
        <header class="mb-3 flex items-center gap-2">
            <Trash2 class="size-4 text-muted-foreground" />
            <h3 class="text-sm font-semibold text-foreground">
                Spend by source
            </h3>
            <span class="ml-auto text-[11px] text-muted-foreground/80">
                Every LLM call across the app
            </span>
        </header>
        {#if bySource.length === 0}
            <p class="text-xs text-muted-foreground py-4">
                No token usage logged yet. Run a mock course to populate this.
            </p>
        {:else}
            <div class="overflow-x-auto">
                <table class="w-full text-xs">
                    <thead>
                        <tr
                            class="border-b border-border text-left text-muted-foreground"
                        >
                            <th class="px-2 py-2 font-medium">Source</th>
                            <th class="px-2 py-2 text-right font-medium">Calls</th>
                            <th class="px-2 py-2 text-right font-medium">
                                Input
                            </th>
                            <th class="px-2 py-2 text-right font-medium">
                                Output
                            </th>
                            <th class="px-2 py-2 text-right font-medium">
                                Total
                            </th>
                            <th class="px-2 py-2 text-right font-medium">
                                Est. spend
                            </th>
                        </tr>
                    </thead>
                    <tbody>
                        {#each bySource as s (s.source)}
                            <tr
                                class="border-b border-border/60 last:border-0"
                            >
                                <td class="px-2 py-2 font-mono text-foreground">
                                    {s.source}
                                </td>
                                <td class="px-2 py-2 text-right tabular">
                                    {fmt(s.calls)}
                                </td>
                                <td
                                    class="px-2 py-2 text-right tabular text-info"
                                >
                                    {fmt(s.prompt_tokens)}
                                </td>
                                <td
                                    class="px-2 py-2 text-right tabular text-accent"
                                >
                                    {fmt(s.completion_tokens)}
                                </td>
                                <td
                                    class="px-2 py-2 text-right tabular text-foreground"
                                >
                                    {fmt(s.total_tokens)}
                                </td>
                                <td
                                    class="px-2 py-2 text-right tabular font-semibold text-success"
                                >
                                    {money(totalCost(s))}
                                </td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
            </div>
        {/if}
    </section>

    <!-- Recent calls -->
    <section
        class="rounded-2xl border border-border bg-card p-5 lift motion-rise-in motion-stagger-3"
    >
        <header class="mb-3 flex items-center gap-2">
            <Beaker class="size-4 text-muted-foreground" />
            <h3 class="text-sm font-semibold text-foreground">
                Mock course runs
            </h3>
            <span class="ml-auto text-[11px] text-muted-foreground/80">
                Per-run token totals
            </span>
        </header>
        {#if byJob.length === 0}
            <p class="text-xs text-muted-foreground py-4">
                No mock runs logged yet.
            </p>
        {:else}
            <div class="max-h-72 overflow-y-auto">
                <table class="w-full text-xs">
                    <thead class="sticky top-0 bg-card">
                        <tr
                            class="border-b border-border text-left text-muted-foreground"
                        >
                            <th class="px-2 py-2 font-medium">Job</th>
                            <th class="px-2 py-2 text-right font-medium">
                                Calls
                            </th>
                            <th class="px-2 py-2 text-right font-medium">
                                Input
                            </th>
                            <th class="px-2 py-2 text-right font-medium">
                                Output
                            </th>
                            <th class="px-2 py-2 text-right font-medium">
                                Total
                            </th>
                            <th class="px-2 py-2 text-right font-medium">
                                Est. spend
                            </th>
                        </tr>
                    </thead>
                    <tbody>
                        {#each byJob as j (j.source)}
                            <tr
                                class="border-b border-border/40 last:border-0"
                            >
                                <td
                                    class="px-2 py-1.5 font-mono text-[11px] text-muted-foreground truncate max-w-[200px]"
                                    title={j.source}
                                >
                                    {j.source || "(no ref)"}
                                </td>
                                <td class="px-2 py-1.5 text-right tabular">
                                    {fmt(j.calls)}
                                </td>
                                <td
                                    class="px-2 py-1.5 text-right tabular text-info"
                                >
                                    {fmt(j.prompt_tokens)}
                                </td>
                                <td
                                    class="px-2 py-1.5 text-right tabular text-accent"
                                >
                                    {fmt(j.completion_tokens)}
                                </td>
                                <td
                                    class="px-2 py-1.5 text-right tabular text-foreground"
                                >
                                    {fmt(j.total_tokens)}
                                </td>
                                <td
                                    class="px-2 py-1.5 text-right tabular font-semibold text-success"
                                >
                                    {money(totalCost(j))}
                                </td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
            </div>
        {/if}
    </section>

    <!-- Recent calls -->
    <section
        class="rounded-2xl border border-border bg-card p-5 lift motion-rise-in motion-stagger-3"
    >
        <header class="mb-3 flex items-center gap-2">
            <Coins class="size-4 text-muted-foreground" />
            <h3 class="text-sm font-semibold text-foreground">
                Recent LLM calls
            </h3>
            <span class="ml-auto text-[11px] text-muted-foreground/80">
                Latest {recent.length}
            </span>
        </header>
        {#if recent.length === 0}
            <p class="text-xs text-muted-foreground py-4">
                Nothing logged yet.
            </p>
        {:else}
            <div class="max-h-72 overflow-y-auto">
                <table class="w-full text-xs">
                    <thead class="sticky top-0 bg-card">
                        <tr
                            class="border-b border-border text-left text-muted-foreground"
                        >
                            <th class="px-2 py-2 font-medium">When</th>
                            <th class="px-2 py-2 font-medium">Source</th>
                            <th class="px-2 py-2 font-medium">Label</th>
                            <th class="px-2 py-2 text-right font-medium">↓In</th>
                            <th class="px-2 py-2 text-right font-medium">↑Out</th>
                            <th class="px-2 py-2 text-right font-medium">Total</th>
                        </tr>
                    </thead>
                    <tbody>
                        {#each recent as r (r.id)}
                            <tr
                                class="border-b border-border/40 last:border-0"
                            >
                                <td
                                    class="px-2 py-1.5 text-muted-foreground whitespace-nowrap"
                                >
                                    {new Date(r.created_at).toLocaleTimeString()}
                                </td>
                                <td
                                    class="px-2 py-1.5 font-mono text-[11px] text-muted-foreground"
                                >
                                    {r.source}
                                </td>
                                <td
                                    class="px-2 py-1.5 font-mono text-[11px] text-foreground/90"
                                >
                                    {r.label || "—"}
                                </td>
                                <td
                                    class="px-2 py-1.5 text-right tabular text-info"
                                >
                                    {fmt(r.prompt_tokens)}
                                </td>
                                <td
                                    class="px-2 py-1.5 text-right tabular text-accent"
                                >
                                    {fmt(r.completion_tokens)}
                                </td>
                                <td
                                    class="px-2 py-1.5 text-right tabular text-foreground"
                                >
                                    {fmt(r.total_tokens)}
                                </td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
            </div>
        {/if}
    </section>
</div>
