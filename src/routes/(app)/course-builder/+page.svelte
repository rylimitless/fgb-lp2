<script lang="ts">
    import {
        Wand2,
        BookOpen,
        LoaderCircle,
        XCircle,
        ArrowRight,
        Sparkles,
        Plus,
        FileText,
        CheckCircle,
        Compass,
        Lightbulb,
        ArrowDown,
    } from "@lucide/svelte";
    import * as Button from "$lib/components/ui/button";
    import { PageHeader } from "$lib/components/brand";
    import { goto } from "$app/navigation";

    let { data } = $props();

    // Approved (ready) source documents the user can pin the outline to.
    // Processing docs are surfaced too, but disabled, so the user knows
    // they exist and can come back once they're ready.
    let approvedDocs = $derived(
        (data.documents ?? []).filter(
            (d: any) => d.approved && d.status === "ready",
        ),
    );
    let processingDocs = $derived(
        (data.documents ?? []).filter(
            (d: any) => d.status === "uploaded" || d.status === "processing",
        ),
    );

    // ---- Outline-form state ----
    let title = $state("");
    let description = $state("");
    let selectedDocIds = $state<number[]>([]);
    // Optional HARD cap on the number of unique questions the course may
    // contain. Blank = no cap. Enforced per module as ceil(total/modules).
    let maxQuestions = $state<number | null>(null);
    let generating = $state(false);
    let genError = $state("");
    let activeJobId = $state<string | null>(null);
    let streamAbort = $state<AbortController | null>(null);

    // ---- SSE stream state ----
    let progress = $state<string[]>([]);

    // ---- Discovery (Stage 0) state ----
    // Before the user commits to a description, they can ask the AI to explore
    // the source material and propose course angles. Picking an angle
    // populates title + description so the outline form below is grounded in
    // what's actually in the docs.
    let discoveryTopic = $state("");
    let discovering = $state(false);
    let discoveryError = $state("");
    let discoveryResult = $state<any>(null);

    async function runDiscovery() {
        discoveryError = "";
        discovering = true;
        discoveryResult = null;
        try {
            const res = await fetch("/api/courses/discover", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                credentials: "include",
                body: JSON.stringify({
                    topic: discoveryTopic.trim(),
                    source_doc_ids: selectedDocIds,
                }),
            });
            if (!res.ok) {
                const err = await res.json().catch(() => ({}));
                discoveryError = err.error || "Discovery failed";
                return;
            }
            discoveryResult = await res.json();
        } catch {
            discoveryError = "Network error. Is the backend running?";
        } finally {
            discovering = false;
        }
    }

    function useAngle(angle: any) {
        title = angle.title ?? "";
        description = angle.description ?? "";
        // Scroll the outline form into view so the user sees the populated
        // fields and the Generate Outline button.
        setTimeout(() => {
            document
                .getElementById("outline-form")
                ?.scrollIntoView({ behavior: "smooth", block: "start" });
        }, 50);
    }

    function toggleDoc(docId: number) {
        if (selectedDocIds.includes(docId)) {
            selectedDocIds = selectedDocIds.filter((id) => id !== docId);
        } else {
            selectedDocIds = [...selectedDocIds, docId];
        }
    }

    function clearSelectedDocs() {
        selectedDocIds = [];
    }

    async function handleOutline() {
        if (!description.trim()) return;
        genError = "";
        progress = [];
        generating = true;
        streamAbort = new AbortController();

        try {
            const enqueue = await fetch("/api/courses/outline", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                credentials: "include",
                body: JSON.stringify({
                    title: title.trim() || "Untitled Course",
                    description: description.trim(),
                    source_doc_ids: selectedDocIds,
                    max_questions: maxQuestions && maxQuestions > 0 ? maxQuestions : undefined,
                }),
            });
            if (!enqueue.ok) {
                const err = await enqueue.json().catch(() => ({}));
                genError = err.error || "Failed to start outline generation";
                generating = false;
                return;
            }
            const { job_id } = await enqueue.json();
            activeJobId = job_id;
            await streamJob(job_id, streamAbort.signal);
        } catch (e: any) {
            if (e?.name === "AbortError") {
                genError = "Outline cancelled";
            } else if (!genError) {
                genError = "Network error. Is the backend running?";
            }
        } finally {
            generating = false;
            activeJobId = null;
            streamAbort = null;
        }
    }

    async function cancelOutline() {
        if (streamAbort) streamAbort.abort();
        if (activeJobId) {
            try {
                await fetch(
                    `/api/courses/generate/${activeJobId}/cancel`,
                    {
                        method: "POST",
                        credentials: "include",
                    },
                );
            } catch {
                /* best-effort */
            }
        }
        generating = false;
        activeJobId = null;
    }

    async function streamJob(jobId: string, signal: AbortSignal) {
        const res = await fetch(`/api/courses/generate/${jobId}/stream`, {
            credentials: "include",
            signal,
        });
        if (!res.ok) {
            genError = "Failed to connect to stream";
            return;
        }
        const reader = res.body?.getReader();
        if (!reader) {
            genError = "Streaming not supported";
            return;
        }
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
                    let payload: any;
                    try {
                        payload = JSON.parse(line.slice(6));
                    } catch {
                        continue;
                    }
                    if (eventType === "step") {
                        progress = [...progress, payload.detail || payload.step];
                    } else if (eventType === "error") {
                        genError =
                            payload.message || "Outline generation failed";
                        return;
                    } else if (eventType === "done") {
                        // Course outline saved — go straight into the builder.
                        if (payload.id) {
                            await goto(
                                `/course-builder/${payload.id}`,
                            );
                            return;
                        }
                    }
                    eventType = "";
                }
            }
        }
    }
</script>

<div class="flex w-full max-w-7xl mx-auto flex-col gap-5">
    <PageHeader
        eyebrow="Staged builder"
        title="Course Builder"
        description="Start with a focused outline, then build and review modules at your own pace."
    >
        {#snippet icon()}
            <Wand2 class="size-6 text-primary" />
        {/snippet}
    </PageHeader>

    <div class="grid items-start gap-5 xl:grid-cols-[minmax(0,1fr)_360px]">
        <!-- Primary creation flow -->
        <section class="flex min-w-0 flex-col gap-4">
            <!-- Shared source document picker (used by both discovery + outline) -->
            {#if approvedDocs.length > 0 || processingDocs.length > 0}
                <div class="order-1 rounded-lg border border-border bg-card p-4">
                    <div class="mb-2 flex items-center justify-between gap-3">
                        <div>
                            <p class="text-sm font-semibold text-foreground">Source documents</p>
                            <p class="text-xs text-muted-foreground">
                                {#if selectedDocIds.length > 0}
                                    Using {selectedDocIds.length} selected document{selectedDocIds.length === 1 ? "" : "s"}.
                                {:else}
                                    All approved documents will be used.
                                {/if}
                            </p>
                        </div>
                        {#if selectedDocIds.length > 0}
                            <button
                                type="button"
                                class="text-xs font-medium text-primary hover:underline"
                                onclick={clearSelectedDocs}
                            >
                                Use all
                            </button>
                        {/if}
                    </div>
                    <div class="flex flex-col gap-1 max-h-48 overflow-y-auto rounded-lg border border-border bg-background/40 p-1">
                        {#each approvedDocs as doc}
                            <button
                                type="button"
                                class="flex items-center gap-2 rounded-md px-2.5 py-1.5 text-xs transition-colors {selectedDocIds.includes(doc.id)
                                    ? 'bg-primary/10 text-foreground'
                                    : 'text-muted-foreground hover:bg-muted hover:text-foreground'}"
                                onclick={() => toggleDoc(doc.id)}
                            >
                                <div
                                    class="size-3.5 rounded border flex items-center justify-center shrink-0 {selectedDocIds.includes(doc.id)
                                        ? 'bg-primary border-primary text-primary-foreground'
                                        : 'border-border'}"
                                >
                                    {#if selectedDocIds.includes(doc.id)}
                                        <CheckCircle class="size-2.5" />
                                    {/if}
                                </div>
                                <FileText class="size-3 shrink-0" />
                                <span class="truncate">{doc.title}</span>
                            </button>
                        {/each}
                        {#each processingDocs as doc}
                            <div
                                class="flex items-center gap-2 rounded-md px-2.5 py-1.5 text-xs text-muted-foreground/70"
                                title="Document is still being processed"
                            >
                                <LoaderCircle class="size-3 animate-spin shrink-0" />
                                <FileText class="size-3 shrink-0" />
                                <span class="truncate">{doc.title}</span>
                                {#if doc.total_chunks > 0}
                                    <span class="ml-auto text-[10px] shrink-0">
                                        {doc.chunks_done}/{doc.total_chunks}
                                    </span>
                                {:else}
                                    <span class="ml-auto text-[10px] shrink-0">
                                        {doc.status}
                                    </span>
                                {/if}
                            </div>
                        {/each}
                    </div>
                </div>
            {/if}

            <!-- Step 1: Discovery -->
            <div class="order-3 rounded-lg border border-primary/30 bg-primary/5 p-5">
                <h2 class="text-sm font-semibold text-foreground flex items-center gap-2 mb-1">
                    <Compass class="size-4 text-primary" />
                    Step 1: Explore sources
                </h2>
                <p class="text-xs text-muted-foreground mb-3">
                    Use this when you need help finding a viable course topic from the selected material.
                </p>
                <div class="flex flex-col gap-3">
                    <input
                        type="text"
                        bind:value={discoveryTopic}
                        disabled={discovering}
                        placeholder="A topic or question (optional — leave blank to survey everything)"
                        class="rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                    />
                    <Button.Root
                        size="sm"
                        disabled={discovering}
                        onclick={runDiscovery}
                    >
                        {#if discovering}
                            <LoaderCircle class="size-3.5 mr-1.5 animate-spin" />
                            Exploring…
                        {:else}
                            <Compass class="size-3.5 mr-1.5" />
                            Explore sources
                        {/if}
                    </Button.Root>

                    {#if discoveryError}
                        <div class="rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 flex items-center gap-2">
                            <XCircle class="size-4 text-destructive shrink-0" />
                            <p class="text-xs text-destructive">{discoveryError}</p>
                        </div>
                    {/if}

                    {#if discoveryResult}
                        <div class="flex flex-col gap-3 mt-1">
                            <!-- AI summary -->
                            {#if discoveryResult.summary}
                                <div class="rounded-lg border border-border bg-background px-3 py-2">
                                    <p class="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-1">
                                        What the AI found
                                        {#if discoveryResult.chunks_seen}
                                            <span class="text-muted-foreground/60 font-normal normal-case tracking-normal">· {discoveryResult.chunks_seen} chunks</span>
                                        {/if}
                                    </p>
                                    <p class="text-xs text-foreground/90 leading-relaxed">
                                        {discoveryResult.summary}
                                    </p>
                                </div>
                            {/if}
                            <!-- Angle cards -->
                            {#if discoveryResult.angles?.length > 0}
                                <div class="flex flex-col gap-2">
                                    <p class="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
                                        Suggested angles ({discoveryResult.angles.length})
                                    </p>
                                    {#each discoveryResult.angles as angle, ai}
                                        <button
                                            type="button"
                                            class="text-left rounded-lg border border-border bg-background px-3 py-2.5 hover:border-primary/40 hover:bg-primary/5 transition-colors group"
                                            onclick={() => useAngle(angle)}
                                        >
                                            <div class="flex items-start gap-2">
                                                <Lightbulb class="size-3.5 text-primary shrink-0 mt-0.5" />
                                                <div class="flex-1 min-w-0">
                                                    <p class="text-xs font-semibold text-foreground">
                                                        {angle.title}
                                                    </p>
                                                    {#if angle.description}
                                                        <p class="text-[11px] text-muted-foreground mt-0.5">
                                                            {angle.description}
                                                        </p>
                                                    {/if}
                                                    {#if angle.rationale}
                                                        <p class="text-[10px] text-muted-foreground/70 mt-1 italic">
                                                            {angle.rationale}
                                                        </p>
                                                    {/if}
                                                </div>
                                                <span class="text-[10px] text-primary opacity-0 group-hover:opacity-100 transition-opacity shrink-0 mt-0.5 font-semibold">
                                                    Use →
                                                </span>
                                            </div>
                                        </button>
                                    {/each}
                                </div>
                            {/if}
                        </div>
                    {/if}
                </div>
            </div>

            <!-- Step 2: Outline form -->
            <div id="outline-form" class="order-2 rounded-lg border border-border bg-card p-5 scroll-mt-4">
                <div class="mb-5 flex items-start justify-between gap-4">
                    <div>
                        <h2 class="text-sm font-semibold text-foreground flex items-center gap-2">
                            <Sparkles class="size-4 text-primary" />
                            Step 1: Define the outline
                        </h2>
                        <p class="mt-1 text-xs text-muted-foreground">Describe the learning outcome and generate an editable module plan.</p>
                    </div>
                    <span class="shrink-0 rounded-full border border-border px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">Draft</span>
                </div>
                <div class="flex flex-col gap-4">
                    <div class="flex flex-col gap-1.5">
                        <label for="ol-title" class="text-xs font-medium text-muted-foreground">
                            Title (optional)
                        </label>
                        <input
                            id="ol-title"
                            type="text"
                            bind:value={title}
                            placeholder="e.g. Introduction to Machine Learning"
                            class="rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                        />
                    </div>
                    <div class="flex flex-col gap-1.5">
                        <label for="ol-desc" class="text-xs font-medium text-muted-foreground">
                            What should this course cover?
                        </label>
                        <textarea
                            id="ol-desc"
                            bind:value={description}
                            rows={5}
                            placeholder="Describe the topic, paste paper text, or list learning goals. The AI will propose an outline you can edit before any content is generated."
                            class="rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring resize-none"
                        ></textarea>
                    </div>
                    <div class="flex flex-col gap-1.5">
                        <label for="ol-maxq" class="text-xs font-medium text-muted-foreground">
                            Max unique questions <span class="text-muted-foreground/60 font-normal">(optional)</span>
                        </label>
                        <input
                            id="ol-maxq"
                            type="number"
                            min="1"
                            bind:value={maxQuestions}
                            placeholder="No cap — e.g. 20 for a hard limit"
                            class="rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                        />
                        <p class="text-[11px] text-muted-foreground/70">
                            Sets a hard cap on distinct questions across the course. Each concept counts once regardless of how many question-type variants it has. Left blank = generate as many as fit the material.
                        </p>
                    </div>

                    {#if genError}
                        <div class="rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 flex items-center gap-2">
                            <XCircle class="size-4 text-destructive shrink-0" />
                            <p class="text-xs text-destructive">{genError}</p>
                        </div>
                    {/if}

                    {#if generating}
                        <div class="flex gap-2">
                            <Button.Root class="flex-1" variant="destructive" onclick={cancelOutline}>
                                <XCircle class="size-4 mr-2" />
                                Stop
                            </Button.Root>
                            <Button.Root class="flex-1" disabled>
                                <LoaderCircle class="size-4 mr-2 animate-spin" />
                                Outlining…
                            </Button.Root>
                        </div>

                        {#if progress.length > 0}
                            <div class="rounded-lg border border-border bg-muted/30 p-3 max-h-40 overflow-y-auto">
                                <div class="flex flex-col gap-1">
                                    {#each progress.slice(-6) as p}
                                        <p class="text-xs text-muted-foreground">{p}</p>
                                    {/each}
                                </div>
                            </div>
                        {/if}
                    {:else}
                        <Button.Root class="w-full" disabled={!description.trim()} onclick={handleOutline}>
                            <Sparkles class="size-4 mr-2" />
                            Generate Outline
                        </Button.Root>
                    {/if}
                </div>
            </div>
        </section>

        <!-- Draft resumption sidebar -->
        <section class="min-w-0 xl:sticky xl:top-4">
            <div class="rounded-lg border border-border bg-card p-4">
                <div class="mb-3 flex items-center justify-between gap-3">
                    <h2 class="text-sm font-semibold text-foreground flex items-center gap-2">
                    <BookOpen class="size-4 text-muted-foreground" />
                    Continue a draft
                    </h2>
                    <span class="rounded-full border border-border px-2 py-0.5 text-[10px] font-semibold text-muted-foreground">{data.drafts.length}</span>
                </div>

                {#if data.drafts.length === 0}
                    <div class="rounded-lg border border-dashed border-border bg-muted/20 px-6 py-10 text-center">
                        <FileText class="size-6 text-muted-foreground/60 mx-auto mb-2" />
                        <p class="text-sm text-muted-foreground">
                            No draft courses yet. Start an outline to begin.
                        </p>
                    </div>
                {:else}
                    <div class="flex max-h-[calc(100vh-14rem)] flex-col gap-2 overflow-y-auto pr-1">
                        {#each data.drafts as course (course.id)}
                            <button
                                class="flex items-center gap-3 rounded-lg border border-border bg-background px-3 py-2.5 text-left transition-colors hover:border-primary/30 hover:bg-muted/40"
                                onclick={() => goto(`/course-builder/${course.id}`)}
                            >
                                <BookOpen class="size-4 text-muted-foreground shrink-0" />
                                <div class="flex-1 min-w-0">
                                    <p class="text-sm font-medium text-foreground truncate">
                                        {course.title}
                                    </p>
                                    {#if course.description}
                                        <p class="text-xs text-muted-foreground truncate">
                                            {course.description}
                                        </p>
                                    {/if}
                                </div>
                                <span class="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground px-2 py-0.5 rounded-full border border-border">
                                    {course.status}
                                </span>
                                <ArrowRight class="size-4 text-muted-foreground shrink-0" />
                            </button>
                        {/each}
                    </div>
                {/if}
            </div>

        </section>
    </div>
</div>
