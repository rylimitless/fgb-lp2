<script lang="ts">
    import type { PublishedCourse } from "../+page.server";
    import {
        Layers,
        ArrowLeft,
        Sparkles,
        Wand2,
        LoaderCircle,
        CheckCircle,
        X,
        Lightbulb,
        BookOpen,
        RefreshCw,
    } from "@lucide/svelte";
    import * as Button from "$lib/components/ui/button";
    import { PageHeader, EmptyState } from "$lib/components/brand";
    import { showToast } from "$lib/components/brand/toast.svelte";
    import { goto } from "$app/navigation";

    let { data } = $props();

    let catalog = $state<PublishedCourse[]>([]);
    $effect(() => {
        catalog = data.catalog ?? [];
    });

    type SuggestedCourse = {
        course_id: number;
        title: string;
        is_required: boolean;
        rationale: string;
    };
    type Generation = {
        title: string;
        description: string;
        summary: string;
        courses: SuggestedCourse[];
    };

    let goal = $state("");
    let generating = $state(false);
    let generation = $state<Generation | null>(null);
    let deselectedIds = $state<Set<number>>(new Set());
    let aiTitle = $state("");
    let aiDescription = $state("");
    let creating = $state(false);

    let activeCourseCount = $derived(
        generation
            ? generation.courses.filter((c) => !deselectedIds.has(c.course_id))
                  .length
            : 0,
    );

    async function handleGenerate() {
        if (!goal.trim()) {
            showToast("Describe the goal of your learning path first.", {
                variant: "error",
            });
            return;
        }
        generating = true;
        generation = null;
        deselectedIds = new Set();
        try {
            const res = await fetch("/api/learning-paths/generate", {
                method: "POST",
                credentials: "include",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ goal: goal.trim() }),
            });
            if (!res.ok) {
                const err = await res.json().catch(() => ({}));
                showToast(err.error ?? "Failed to generate learning path", {
                    variant: "error",
                });
                return;
            }
            const result: Generation = await res.json();
            generation = result;
            aiTitle = result.title;
            aiDescription = result.description;
            if (result.courses.length === 0) {
                showToast(
                    "No matching courses found in the catalog. Try a different goal.",
                    { variant: "default" },
                );
            }
        } finally {
            generating = false;
        }
    }

    function toggleDeselect(id: number) {
        const next = new Set(deselectedIds);
        if (next.has(id)) next.delete(id);
        else next.add(id);
        deselectedIds = next;
    }

    function toggleRequired(c: SuggestedCourse) {
        if (!generation) return;
        generation = {
            ...generation,
            courses: generation.courses.map((x) =>
                x.course_id === c.course_id
                    ? { ...x, is_required: !x.is_required }
                    : x,
            ),
        };
    }

    function resetAll() {
        generation = null;
        deselectedIds = new Set();
        aiTitle = "";
        aiDescription = "";
    }

    async function handleCreateFromAI() {
        if (!generation) return;
        if (!aiTitle.trim()) {
            showToast("Title is required.", { variant: "error" });
            return;
        }
        creating = true;
        try {
            // 1. Create the path.
            const createRes = await fetch("/api/admin/learning-paths", {
                method: "POST",
                credentials: "include",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    title: aiTitle.trim(),
                    description: aiDescription.trim(),
                }),
            });
            if (!createRes.ok) {
                const err = await createRes.json().catch(() => ({}));
                showToast(err.error ?? "Failed to create path", {
                    variant: "error",
                });
                return;
            }
            const created = await createRes.json();
            const pathId = created.id;

            // 2. Add each (non-deselected) course in order.
            const selected = generation.courses.filter(
                (c) => !deselectedIds.has(c.course_id),
            );
            let added = 0;
            for (let i = 0; i < selected.length; i++) {
                const c = selected[i];
                const res = await fetch(
                    `/api/admin/learning-paths/${pathId}/courses`,
                    {
                        method: "POST",
                        credentials: "include",
                        headers: { "Content-Type": "application/json" },
                        body: JSON.stringify({
                            course_id: c.course_id,
                            sort_order: i,
                            is_required: c.is_required,
                        }),
                    },
                );
                if (res.ok) added++;
            }
            showToast(
                `Path created with ${added} course${added === 1 ? "" : "s"}.`,
                { variant: "success" },
            );
            await goto(`/learning-path-management/${pathId}`);
        } finally {
            creating = false;
        }
    }
</script>

<svelte:head
    ><title>AI Path Builder - FGB Academy</title></svelte:head
>

<div class="mx-auto flex w-full max-w-6xl flex-col gap-6">
    <PageHeader
        eyebrow="AI builder"
        title="Learning Path Builder"
        description="Describe a goal and the AI will assemble a path from your approved courses — selecting, ordering, and explaining each choice. You review everything before it's created."
    >
        {#snippet icon()}
            <Wand2 class="size-6 text-primary" />
        {/snippet}
        {#snippet actions()}
            <Button.Root
                variant="outline"
                size="sm"
                onclick={() => goto("/learning-path-management")}
            >
                <ArrowLeft class="size-4" /> Back
            </Button.Root>
        {/snippet}
    </PageHeader>

    <div class="flex flex-col gap-6 md:flex-row">
        <!-- LEFT: goal input + generate -->
        <section class="flex shrink-0 flex-col gap-4 md:w-[440px]">
            <div class="rounded-xl border border-primary/30 bg-primary/5 p-5">
                <h2
                    class="mb-1 flex items-center gap-2 text-sm font-semibold text-foreground"
                >
                    <Sparkles class="size-4 text-primary" /> Your goal
                </h2>
                <p class="mb-3 text-xs text-muted-foreground">
                    What should learners achieve by finishing this path?
                </p>
                <textarea
                    rows="4"
                    bind:value={goal}
                    placeholder="e.g. Onboard new engineering hires on our systems, from local setup through production deploys."
                    class="w-full rounded-lg border border-border bg-background/40 px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                ></textarea>
                <div class="mt-3">
                    <Button.Root
                        variant="default"
                        class="w-full"
                        onclick={handleGenerate}
                        disabled={generating}
                    >
                        {#if generating}
                            <LoaderCircle class="size-4 animate-spin" />
                            Generating...
                        {:else}
                            <Wand2 class="size-4" /> Generate Path
                        {/if}
                    </Button.Root>
                </div>
                {#if generation}
                    <Button.Root
                        variant="ghost"
                        size="xs"
                        class="mt-2 w-full"
                        onclick={handleGenerate}
                        disabled={generating}
                    >
                        <RefreshCw class="size-3" /> Regenerate
                    </Button.Root>
                {/if}
            </div>

            <!-- Catalog context -->
            <div class="rounded-xl border border-border bg-card p-5">
                <h2
                    class="mb-2 flex items-center gap-2 text-xs font-semibold text-muted-foreground"
                >
                    <BookOpen class="size-4" /> Catalog
                </h2>
                <p class="text-sm text-foreground">
                    {catalog.length} approved
                    {catalog.length === 1 ? "course" : "courses"}
                    available
                </p>
                <p class="mt-1 text-xs text-muted-foreground">
                    The AI only selects from approved, published courses.
                </p>
            </div>
        </section>

        <!-- RIGHT: result / preview -->
        <section class="min-w-0 flex-1">
            <div class="rounded-xl border border-border bg-card p-6">
                {#if !generation}
                    {#if generating}
                        <div
                            class="flex flex-col items-center justify-center gap-3 py-16"
                        >
                            <LoaderCircle
                                class="size-8 animate-spin text-primary"
                            />
                            <p class="text-sm text-muted-foreground">
                                Assembling your learning path...
                            </p>
                        </div>
                    {:else}
                        <EmptyState
                            title="Your generated path will appear here"
                            description="Describe a goal on the left, then click Generate. The AI will pick and order courses from your approved catalog."
                        >
                            {#snippet icon()}
                                <Layers class="size-12" />
                            {/snippet}
                        </EmptyState>
                    {/if}
                {:else}
                    <div class="flex flex-col gap-5">
                        <!-- Header row -->
                        <div
                            class="flex items-center justify-between gap-3 border-b border-border pb-4"
                        >
                            <div>
                                <h3 class="text-base font-semibold text-foreground">
                                    Suggested path
                                </h3>
                                <p class="text-xs text-muted-foreground">
                                    {activeCourseCount} of
                                    {generation.courses.length} courses selected
                                </p>
                            </div>
                            <Button.Root
                                variant="ghost"
                                size="sm"
                                onclick={resetAll}
                            >
                                <X class="size-4" /> Clear
                            </Button.Root>
                        </div>

                        <!-- Summary -->
                        {#if generation.summary}
                            <div
                                class="flex items-start gap-2 rounded-lg border border-border bg-muted/30 p-3"
                            >
                                <Lightbulb
                                    class="mt-0.5 size-4 shrink-0 text-primary"
                                />
                                <p class="text-xs text-muted-foreground">
                                    {generation.summary}
                                </p>
                            </div>
                        {/if}

                        <!-- Editable title / description -->
                        <div class="flex flex-col gap-2">
                            <label
                                for="ai-title"
                                class="text-xs font-medium text-muted-foreground"
                                >Path title</label
                            >
                            <input
                                id="ai-title"
                                type="text"
                                bind:value={aiTitle}
                                class="w-full rounded-lg border border-border bg-muted px-3 py-2 text-sm font-semibold text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                            />
                            <label
                                for="ai-desc"
                                class="mt-1 text-xs font-medium text-muted-foreground"
                                >Description</label
                            >
                            <textarea
                                id="ai-desc"
                                rows="2"
                                bind:value={aiDescription}
                                class="w-full rounded-lg border border-border bg-muted px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                            ></textarea>
                        </div>

                        <!-- Selected courses -->
                        <div>
                            <h4
                                class="mb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground"
                            >
                                Courses
                            </h4>
                            {#if generation.courses.length === 0}
                                <p class="text-sm text-muted-foreground">
                                    No courses suggested. Try a different goal or
                                    add more approved courses to your catalog.
                                </p>
                            {:else}
                                <div class="flex flex-col gap-2">
                                    {#each generation.courses as c, i (c.course_id)}
                                        <div
                                            class={`flex items-start gap-3 rounded-lg border p-3 transition-colors ${deselectedIds.has(c.course_id) ? "border-border bg-muted/20 opacity-50" : "border-border bg-card"}`}
                                        >
                                            <span
                                                class="mt-0.5 flex size-6 shrink-0 items-center justify-center rounded-full bg-primary text-xs font-bold text-primary-foreground"
                                            >
                                                {i + 1}
                                            </span>
                                            <div class="min-w-0 flex-1">
                                                <p
                                                    class="text-sm font-medium text-foreground"
                                                >
                                                    {c.title}
                                                </p>
                                                {#if c.rationale}
                                                    <p
                                                        class="mt-0.5 text-xs text-muted-foreground"
                                                    >
                                                        {c.rationale}
                                                    </p>
                                                {/if}
                                                <div
                                                    class="mt-2 flex items-center gap-3"
                                                >
                                                    <button
                                                        type="button"
                                                        class="text-[11px] font-medium {c.is_required ? "text-primary" : "text-muted-foreground"}"
                                                        onclick={() => toggleRequired(c)}
                                                    >
                                                        {c.is_required
                                                            ? "● Required"
                                                            : "○ Optional"}
                                                    </button>
                                                </div>
                                            </div>
                                            <button
                                                type="button"
                                                class="shrink-0 rounded p-1 text-muted-foreground hover:bg-muted hover:text-destructive"
                                                onclick={() =>
                                                    toggleDeselect(c.course_id)}
                                                title={deselectedIds.has(
                                                    c.course_id,
                                                )
                                                    ? "Restore"
                                                    : "Remove"}
                                            >
                                                {#if deselectedIds.has(
                                                    c.course_id,
                                                )}
                                                    <X class="size-3.5" />
                                                {:else}
                                                    <X class="size-3.5" />
                                                {/if}
                                            </button>
                                        </div>
                                    {/each}
                                </div>
                            {/if}
                        </div>

                        <!-- Create -->
                        <div
                            class="flex items-center justify-between gap-2 border-t border-border pt-4"
                        >
                            <p class="text-xs text-muted-foreground">
                                Ready to create? You can still edit everything
                                afterward.
                            </p>
                            <Button.Root
                                variant="default"
                                size="sm"
                                onclick={handleCreateFromAI}
                                disabled={creating || activeCourseCount === 0}
                            >
                                {#if creating}
                                    <LoaderCircle class="size-4 animate-spin" />
                                    Creating...
                                {:else}
                                    <CheckCircle class="size-4" />
                                    Create Path
                                {/if}
                            </Button.Root>
                        </div>
                    </div>
                {/if}
            </div>
        </section>
    </div>
</div>
