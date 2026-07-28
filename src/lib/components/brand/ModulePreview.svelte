<script lang="ts">
    import {
        X,
        CheckCircle,
        Eye,
        RefreshCw,
        ChevronLeft,
        ChevronRight,
        BookOpen,
        Award,
        PanelLeftClose,
        PanelLeftOpen,
        Circle,
        CheckCircle2,
        AlertTriangle,
        ChevronDown,
        BarChart3,
        FileText,
        Hash,
        MessageSquare,
    } from "@lucide/svelte";
    import * as Button from "$lib/components/ui/button";
    import Markdown from "$lib/components/brand/Markdown.svelte";
    import AnswerFeedback from "$lib/components/brand/AnswerFeedback.svelte";
    import QuestionMatching from "$lib/components/brand/QuestionMatching.svelte";
    import QuestionOrdering from "$lib/components/brand/QuestionOrdering.svelte";
    import QuestionHotspot from "$lib/components/brand/QuestionHotspot.svelte";

    let {
        module = $bindable(),
        courseTitle = "",
        allModules = [],
        onClose,
    }: {
        module: any;
        courseTitle?: string;
        allModules?: any[];
        onClose: () => void;
    } = $props();

    // ---- Learner state ----
    // answers[itemId] = the learner's selected answer for that item.
    // checked[itemId] = whether the learner has submitted their answer.
    // revealed[itemId] = whether the learner chose to reveal the correct answer.
    let answers = $state<Record<number, any>>({});
    let checked = $state<Record<number, boolean>>({});
    let revealed = $state<Record<number, boolean>>({});

    // Currently shown item index for the one-at-a-time walk-through mode.
    // -1 = show all items (overview mode).
    let currentIndex = $state(0);
    let walkMode = $state(true);

    // ---- Mini-map sidebar ----
    // Collapsible list of all modules in the course so the user can jump
    // between modules without closing the preview.
    let sidebarOpen = $state(true);

    // Which modules are expanded in the sidebar to show their items.
    let expandedSidebarModules = $state<Record<number, boolean>>({});

    function toggleSidebarModule(id: number) {
        expandedSidebarModules[id] = !expandedSidebarModules[id];
    }

    // ---- Metrics slide-in panel ----
    let metricsOpen = $state(false);

    // Compute course-level metrics across ALL modules (not just the current
    // one) so the metrics panel gives a full picture.
    let allItems = $derived(
        (allModules ?? []).flatMap((m: any) => m.items ?? []),
    );
    let allContentItems = $derived(
        allItems.filter((i: any) => i.item_type === "content"),
    );
    let allQuestionItems = $derived(
        allItems.filter((i: any) => i.item_type !== "content"),
    );

    // Question type distribution: { mc: 5, tf: 3, ... }
    let typeDistribution = $derived.by(() => {
        const counts: Record<string, number> = {};
        for (const item of allQuestionItems) {
            const t = item.item_type ?? "unknown";
            counts[t] = (counts[t] ?? 0) + 1;
        }
        // Sort by count descending for display.
        return Object.entries(counts)
            .sort((a, b) => b[1] - a[1])
            .map(([type, count]) => ({ type, count }));
    });

    // Module status distribution
    let statusDistribution = $derived.by(() => {
        const counts: Record<string, number> = {};
        for (const m of allModules ?? []) {
            const s = m.status ?? "pending";
            counts[s] = (counts[s] ?? 0) + 1;
        }
        return counts;
    });

    // Per-module item counts for the sidebar expansion
    function moduleItems(mod: any): any[] {
        return (mod.items ?? []).slice().sort(
            (a: any, b: any) => (a.sort_order ?? 0) - (b.sort_order ?? 0),
        );
    }

    // Short preview text for an item in the sidebar
    function itemPreview(item: any): string {
        const d = item.data ?? {};
        const text = d.question ?? d.statement ?? d.text ?? d.body ?? "";
        if (!text) return "(empty)";
        // Strip markdown and truncate.
        const stripped = String(text).replace(/[#*`_~>-]/g, "").trim();
        return stripped.length > 60 ? stripped.slice(0, 60) + "…" : stripped;
    }

    // Pre-computed for the metrics panel (can't use {@const} in template divs).
    let maxTypeCount = $derived(typeDistribution[0]?.count ?? 1);
    let contentPct = $derived(allItems.length ? allContentItems.length / allItems.length * 100 : 0);
    let questionPct = $derived(allItems.length ? allQuestionItems.length / allItems.length * 100 : 0);

    // When the user switches modules via the mini-map, update the bound
    // `module` prop so the parent tracks the change too. Also reset the
    // learner state for the new module.
    function switchToModule(mod: any) {
        if (!mod || mod.id === module?.id) return;
        module = mod;
        answers = {};
        checked = {};
        revealed = {};
        currentIndex = 0;
    }

    // Sorted module list for the sidebar (same ordering as the builder).
    let sortedModules = $derived(
        [...(allModules ?? [])].sort(
            (a, b) => (a.sort_order ?? 0) - (b.sort_order ?? 0),
        ),
    );

    function moduleStatusIcon(mod: any) {
        const itemCount = mod.items?.length ?? 0;
        if (itemCount === 0) return { Icon: Circle, class: "text-muted-foreground/40" };
        if (mod.status === "ready") return { Icon: CheckCircle2, class: "text-success" };
        return { Icon: BookOpen, class: "text-muted-foreground" };
    }

    let items = $derived(
        (module?.items ?? []).slice().sort(
            (a: any, b: any) => (a.sort_order ?? 0) - (b.sort_order ?? 0),
        ),
    );

    let assessableItems = $derived(
        items.filter((i: any) => i.item_type !== "content"),
    );

    let score = $derived(() => {
        let correct = 0;
        let total = assessableItems.length;
        for (const item of assessableItems) {
            if (checked[item.id] && isCorrect(item, answers[item.id])) {
                correct++;
            }
        }
        return { correct, total };
    });

    let currentItem = $derived(walkMode ? items[currentIndex] : null);

    function setAnswer(itemId: number, answer: any) {
        answers[itemId] = answer;
    }

    function toggleMA(itemId: number, optionIndex: number) {
        const current = answers[itemId] ?? [];
        if (current.includes(optionIndex)) {
            answers[itemId] = current.filter((i: number) => i !== optionIndex);
        } else {
            answers[itemId] = [...current, optionIndex];
        }
    }

    function check(itemId: number) {
        checked[itemId] = true;
    }

    function reveal(itemId: number) {
        revealed[itemId] = true;
        checked[itemId] = true;
    }

    function next() {
        if (currentIndex < items.length - 1) {
            currentIndex++;
        }
    }

    function prev() {
        if (currentIndex > 0) {
            currentIndex--;
        }
    }

    function resetAll() {
        answers = {};
        checked = {};
        revealed = {};
        currentIndex = 0;
    }

    function isCorrect(item: any, answer: any): boolean {
        const d = item.data;
        switch (item.item_type) {
            case "mc":
                return answer === d.correct;
            case "ma": {
                if (!Array.isArray(answer) || !d.correct) return false;
                const correct = d.correct as number[];
                return (
                    answer.length === correct.length &&
                    correct.every((c: number) => answer.includes(c))
                );
            }
            case "tf":
                return answer === d.answer;
            case "fb": {
                if (!Array.isArray(answer) || !Array.isArray(d.blanks))
                    return false;
                return (
                    answer.length === d.blanks.length &&
                    d.blanks.every(
                        (b: string, i: number) =>
                            String(answer[i] ?? "")
                                .trim()
                                .toLowerCase() ===
                            String(b).trim().toLowerCase(),
                    )
                );
            }
            case "sa":
                // Short answer is always "correct" on check — it's self-assessed.
                return answer !== undefined && String(answer).trim().length > 0;
            case "hotspot": {
                // The QuestionHotspot component normalizes various data shapes
                // into regions[].correct. Mirror that logic here.
                if (typeof answer !== "number") return false;
                const regions = d.regions ?? d.hotspots ?? d.areas ?? d.options ?? [];
                const region = regions[answer];
                if (!region) return false;
                if (typeof region.correct === "boolean") return region.correct;
                if (typeof d.correct === "number") return d.correct === answer;
                if (typeof d.correct === "string")
                    return String(region.id ?? region.label ?? "").toLowerCase() === d.correct.toLowerCase();
                return false;
            }
            default:
                return false;
        }
    }

    function correctAnswerLabel(item: any): string {
        const d = item.data ?? {};
        switch (item.item_type) {
            case "mc":
                return d.options?.[d.correct] ?? "";
            case "ma":
                return Array.isArray(d.correct)
                    ? d.correct
                          .map((i: number) => d.options?.[i])
                          .filter(Boolean)
                          .join(", ")
                    : "";
            case "tf":
                return d.answer === true ? "True" : "False";
            case "fb":
                return Array.isArray(d.blanks) ? d.blanks.join(", ") : "";
            case "hotspot": {
                const regs = d.regions ?? d.hotspots ?? d.areas ?? d.options ?? [];
                const correctReg = regs.find((r: any, i: number) => {
                    if (typeof r.correct === "boolean") return r.correct;
                    if (typeof d.correct === "number") return d.correct === i;
                    if (typeof d.correct === "string")
                        return String(r.id ?? r.label ?? "").toLowerCase() === d.correct.toLowerCase();
                    return false;
                });
                return correctReg?.label ?? correctReg?.text ?? "";
            }
            default:
                return "";
        }
    }

    function itemData(item: any): any {
        let d: any = {};
        try {
            d =
                typeof item.data === "string"
                    ? JSON.parse(item.data)
                    : item.data;
        } catch {
            d = item.data ?? {};
        }
        return d ?? {};
    }

    // Use itemData() to normalize, since the SSE item events send data as
    // raw JSON objects while the GET /courses/:id response sends them as
    // json.RawMessage (which may or may not be stringified).
    function normalize(item: any): any {
        return { ...item, data: itemData(item) };
    }

    // Human-readable label for each question type. Shown on every question
    // card so the reviewer can tell what type a broken/empty question was
    // supposed to be.
    const TYPE_LABELS: Record<string, string> = {
        mc: "Multiple Choice",
        ma: "Multiple Answer",
        tf: "True / False",
        fb: "Fill in the Blank",
        sa: "Short Answer",
        matching: "Matching",
        drag_sort: "Drag & Sort",
        hotspot: "Hotspot",
        sequence: "Sequence",
        scale: "Scale",
        dd: "Drag & Drop",
        essay: "Essay",
    };
    function questionTypeLabel(type: string): string {
        return TYPE_LABELS[type] ?? type ?? "Unknown";
    }

    // Detect whether a question has no usable content — the AI sometimes
    // generates items with empty or malformed data. This flags them so the
    // reviewer knows it's a generation bug, not a rendering bug.
    function isQuestionEmpty(item: any): boolean {
        const d = item.data ?? {};
        switch (item.item_type) {
            case "mc":
            case "ma":
                return !d.question && !d.options?.length;
            case "tf":
                return !d.statement && d.answer === undefined;
            case "fb":
                return !d.text && !d.question && !d.blanks?.length;
            case "sa":
                return !d.question && !d.sample_answer;
            case "matching":
                return !d.pairs?.length && !d.question;
            case "drag_sort":
                return !d.items?.length && !d.question;
            case "hotspot":
                return !d.regions?.length && !d.hotspots?.length && !d.areas?.length && !d.options?.length;
            default:
                return !d.question && !d.statement && !d.body;
        }
    }

    let progressPct = $derived(
        assessableItems.length === 0
            ? 0
            : Math.round(
                  (assessableItems.filter(
                      (i: any) => checked[i.id] !== undefined,
                  ).length /
                      assessableItems.length) *
                      100,
              ),
    );

    // Handle Escape key to close
    function handleKeydown(e: KeyboardEvent) {
        if (e.key === "Escape") {
            onClose();
        } else if (walkMode && e.key === "ArrowRight") {
            next();
        } else if (walkMode && e.key === "ArrowLeft") {
            prev();
        }
    }
</script>

<svelte:window on:keydown={handleKeydown} />

<!-- Full-screen overlay -->
<div
    class="fixed inset-0 z-50 bg-background/95 backdrop-blur-sm flex flex-col"
    role="dialog"
    aria-modal="true"
    aria-label="Module preview"
>
    <!-- Header -->
    <header
        class="shrink-0 border-b border-border bg-card px-4 py-3 flex items-center gap-3"
    >
        <!-- Sidebar toggle (only shows if there are multiple modules) -->
        {#if sortedModules.length > 1}
            <button
                class="size-8 rounded-md hover:bg-muted flex items-center justify-center text-muted-foreground hover:text-foreground shrink-0"
                onclick={() => (sidebarOpen = !sidebarOpen)}
                title={sidebarOpen ? "Hide module list" : "Show module list"}
            >
                {#if sidebarOpen}
                    <PanelLeftClose class="size-4" />
                {:else}
                    <PanelLeftOpen class="size-4" />
                {/if}
            </button>
        {/if}
        <div class="flex items-center gap-2 min-w-0 flex-1">
            <BookOpen class="size-4 text-muted-foreground shrink-0" />
            <div class="min-w-0">
                <p class="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground truncate">
                    {courseTitle || "Course"} · Learner preview
                </p>
                <p class="text-sm font-semibold text-foreground truncate">
                    {module?.title ?? "Module"}
                </p>
            </div>
        </div>

        <!-- Score badge -->
        {#if assessableItems.length > 0}
            <div class="flex items-center gap-1.5 text-xs px-2.5 py-1 rounded-full border border-border bg-muted/40">
                <Award class="size-3.5 text-accent" />
                <span class="font-medium text-foreground">{score().correct}/{score().total}</span>
                <span class="text-muted-foreground">answered</span>
            </div>
        {/if}

        <!-- Walk/overview toggle -->
        <div class="flex items-center rounded-lg border border-border bg-muted/30 p-0.5">
            <button
                class="px-2.5 py-1 text-xs font-medium rounded-md transition-colors {walkMode ? 'bg-background text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'}"
                onclick={() => (walkMode = true)}
            >
                Walk-through
            </button>
            <button
                class="px-2.5 py-1 text-xs font-medium rounded-md transition-colors {!walkMode ? 'bg-background text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'}"
                onclick={() => (walkMode = false)}
            >
                All at once
            </button>
        </div>

        <!-- Metrics toggle -->
        <button
            class="inline-flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs font-medium {metricsOpen ? 'bg-muted text-foreground' : 'text-muted-foreground hover:text-foreground'} hover:bg-muted transition-colors"
            onclick={() => (metricsOpen = !metricsOpen)}
            title="Show metrics"
        >
            <BarChart3 class="size-3.5" />
            Metrics
        </button>

        <button
            class="inline-flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
            onclick={onClose}
            title="Close preview (Esc)"
        >
            <X class="size-3.5" />
            Cancel preview
        </button>
    </header>

    <!-- Progress bar -->
    {#if assessableItems.length > 0}
        <div class="shrink-0 h-1 bg-muted">
            <div
                class="h-full bg-primary transition-all duration-300"
                style="width: {progressPct}%"
            ></div>
        </div>
    {/if}

    <!-- Body: sidebar (mini-map) + content -->
    <div class="flex-1 flex overflow-hidden">
        <!-- Mini-map sidebar -->
        {#if sidebarOpen && sortedModules.length > 1}
            <aside class="w-72 shrink-0 border-r border-border bg-card/50 overflow-y-auto">
                <div class="p-3">
                    <p class="text-[10px] font-semibold uppercase tracking-[0.18em] text-muted-foreground mb-2 px-1">
                        Modules ({sortedModules.length})
                    </p>
                    <div class="flex flex-col gap-0.5">
                        {#each sortedModules as mod, mi (mod.id)}
                            {@const status = moduleStatusIcon(mod)}
                            {@const isExpanded = expandedSidebarModules[mod.id]}
                            {@const modItems = moduleItems(mod)}
                            <div>
                                <button
                                    class="flex items-start gap-2 rounded-lg px-2.5 py-2 text-left transition-colors w-full {mod.id === module?.id ? 'bg-primary/10 text-foreground' : 'text-muted-foreground hover:bg-muted hover:text-foreground'}"
                                    onclick={() => switchToModule(mod)}
                                    title={mod.title}
                                >
                                    <span
                                        class="size-4 shrink-0 mt-0.5 flex items-center justify-center rounded hover:bg-muted-foreground/10 cursor-pointer"
                                        onclick={(e) => {
                                            e.stopPropagation();
                                            toggleSidebarModule(mod.id);
                                        }}
                                        onkeydown={(e) => { if (e.key === 'Enter') toggleSidebarModule(mod.id); }}
                                        role="button"
                                        tabindex="0"
                                        title={isExpanded ? "Collapse items" : "Show items"}
                                    >
                                        <ChevronDown class="size-3 text-muted-foreground transition-transform {isExpanded ? 'rotate-0' : '-rotate-90'}" />
                                    </span>
                                    <status.Icon class="size-3.5 shrink-0 mt-0.5 {mod.id === module?.id ? 'text-primary' : status.class}" />
                                    <div class="min-w-0 flex-1">
                                        <p class="text-[10px] font-medium text-muted-foreground">
                                            Module {mi + 1}
                                        </p>
                                        <p class="text-xs font-medium truncate">
                                            {mod.title}
                                        </p>
                                        <p class="text-[10px] text-muted-foreground/70 mt-0.5">
                                            {modItems.length} items
                                        </p>
                                    </div>
                                </button>
                                <!-- Expanded items under this module -->
                                {#if isExpanded && modItems.length > 0}
                                    <div class="ml-7 border-l border-border/50 pl-2 mt-0.5 flex flex-col gap-0.5">
                                        {#each modItems as item (item.id)}
                                            <div class="flex items-start gap-1.5 rounded px-2 py-1 text-[10px] text-muted-foreground">
                                                {#if item.item_type === "content"}
                                                    <FileText class="size-3 mt-0.5 shrink-0" />
                                                {:else}
                                                    <MessageSquare class="size-3 mt-0.5 shrink-0" />
                                                {/if}
                                                <span class="truncate">{itemPreview(item)}</span>
                                            </div>
                                        {/each}
                                    </div>
                                {:else if isExpanded}
                                    <p class="ml-7 text-[10px] text-muted-foreground/50 mt-0.5 mb-1">No items yet</p>
                                {/if}
                            </div>
                        {/each}
                    </div>
                </div>
            </aside>
        {/if}

        <!-- Content area -->
        <div class="flex-1 overflow-y-auto">
        <div class="max-w-3xl mx-auto px-4 py-6 md:py-8">
            {#if items.length === 0}
                <div class="text-center py-16">
                    <BookOpen class="size-8 text-muted-foreground/40 mx-auto mb-3" />
                    <p class="text-sm text-muted-foreground">
                        This module has no content to preview yet.
                    </p>
                </div>
            {:else if walkMode && currentItem}
                {@const item = normalize(currentItem)}
                <div class="motion-rise-in">
                    {#if item.item_type === "content"}
                        <article class="rounded-2xl border border-border bg-card p-6 md:p-8">
                            <span class="text-[10px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">
                                Learning material
                            </span>
                            {#if item.data?.body}
                                <Markdown source={item.data.body} />
                            {/if}
                        </article>
                        <div class="flex items-center justify-between mt-4">
                            <Button.Root variant="outline" size="sm" disabled={currentIndex === 0} onclick={prev}>
                                <ChevronLeft class="size-4 mr-1" />
                                Previous
                            </Button.Root>
                            <span class="text-xs text-muted-foreground">
                                {currentIndex + 1} / {items.length}
                            </span>
                            {#if currentIndex < items.length - 1}
                                <Button.Root size="sm" onclick={next}>
                                    Next
                                    <ChevronRight class="size-4 ml-1" />
                                </Button.Root>
                            {:else}
                                <Button.Root size="sm" variant="outline" onclick={() => (walkMode = false)}>
                                    View all
                                </Button.Root>
                            {/if}
                        </div>
                    {:else}
                        <!-- Question rendering (shared with all-at-once mode below) -->
                        {@render questionCard(item, true)}
                        <div class="flex items-center justify-between mt-4">
                            <Button.Root variant="outline" size="sm" disabled={currentIndex === 0} onclick={prev}>
                                <ChevronLeft class="size-4 mr-1" />
                                Previous
                            </Button.Root>
                            <span class="text-xs text-muted-foreground">
                                {currentIndex + 1} / {items.length}
                            </span>
                            <Button.Root size="sm" onclick={next} disabled={currentIndex >= items.length - 1}>
                                Next
                                <ChevronRight class="size-4 ml-1" />
                            </Button.Root>
                        </div>
                    {/if}
                </div>
            {:else}
                <!-- All-at-once mode: render every item -->
                <div class="flex flex-col gap-4">
                    {#each items as rawItem, ii (rawItem.id)}
                        {@const item = normalize(rawItem)}
                        {#if item.item_type === "content"}
                            <article class="rounded-2xl border border-border bg-card p-6 md:p-8 motion-rise-in" style="animation-delay: {Math.min(ii * 30, 200)}ms">
                                <span class="text-[10px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">
                                    Learning material
                                </span>
                                {#if item.data?.body}
                                    <Markdown source={item.data.body} />
                                {/if}
                            </article>
                        {:else}
                            {@render questionCard(item, false)}
                        {/if}
                    {/each}
                </div>

                {#if assessableItems.length > 0}
                    <div class="mt-8 flex items-center justify-center">
                        <Button.Root variant="outline" size="sm" onclick={resetAll}>
                            <RefreshCw class="size-3.5 mr-1.5" />
                            Reset answers ({score().correct}/{score().total} correct)
                        </Button.Root>
                    </div>
                {/if}
            {/if}
        </div>
    </div>

        <!-- Metrics slide-in panel -->
        {#if metricsOpen}
            <aside class="w-72 shrink-0 border-l border-border bg-card/50 overflow-y-auto">
                <div class="p-4 flex flex-col gap-4">
                    <div class="flex items-center justify-between">
                        <p class="text-[10px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">
                            Course metrics
                        </p>
                        <button
                            class="size-6 rounded hover:bg-muted flex items-center justify-center text-muted-foreground"
                            onclick={() => (metricsOpen = false)}
                            title="Close metrics"
                        >
                            <X class="size-3.5" />
                        </button>
                    </div>

                    <!-- Overview -->
                    <div class="flex flex-col gap-1.5">
                        <p class="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">Overview</p>
                        <div class="grid grid-cols-2 gap-2">
                            <div class="rounded-lg border border-border bg-background px-3 py-2 text-center">
                                <span class="text-lg font-semibold text-foreground">{allModules?.length ?? 0}</span>
                                <p class="text-[10px] text-muted-foreground">Modules</p>
                            </div>
                            <div class="rounded-lg border border-border bg-background px-3 py-2 text-center">
                                <span class="text-lg font-semibold text-foreground">{allContentItems.length}</span>
                                <p class="text-[10px] text-muted-foreground">Content blocks</p>
                            </div>
                            <div class="rounded-lg border border-border bg-background px-3 py-2 text-center">
                                <span class="text-lg font-semibold text-foreground">{allQuestionItems.length}</span>
                                <p class="text-[10px] text-muted-foreground">Questions</p>
                            </div>
                            <div class="rounded-lg border border-border bg-background px-3 py-2 text-center">
                                <span class="text-lg font-semibold text-foreground">{allItems.length}</span>
                                <p class="text-[10px] text-muted-foreground">Total items</p>
                            </div>
                        </div>
                    </div>

                    <!-- Module statuses -->
                    {#if Object.keys(statusDistribution).length > 0}
                        <div class="flex flex-col gap-1.5">
                            <p class="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">Module statuses</p>
                            <div class="flex flex-col gap-1">
                                {#each Object.entries(statusDistribution) as [status, count]}
                                    {@const pill = status === 'ready' ? 'text-success bg-success/10' : status === 'generating' ? 'text-info bg-info/10' : status === 'failed' ? 'text-destructive bg-destructive/10' : 'text-muted-foreground bg-muted/40'}
                                    <div class="flex items-center justify-between rounded-md px-2.5 py-1 {pill}">
                                        <span class="text-xs font-medium capitalize">{status}</span>
                                        <span class="text-xs font-semibold">{count}</span>
                                    </div>
                                {/each}
                            </div>
                        </div>
                    {/if}

                    <!-- Question types -->
                    {#if typeDistribution.length > 0}
                        <div class="flex flex-col gap-1.5">
                            <p class="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">Question types</p>
                            <div class="flex flex-col gap-1">
                                {#each typeDistribution as entry}
                                    <div class="flex items-center justify-between rounded-md px-2.5 py-1 border border-border bg-background">
                                        <span class="text-xs text-foreground">{questionTypeLabel(entry.type)}</span>
                                        <div class="flex items-center gap-2">
                                            <!-- Mini bar chart -->
                                            <div class="w-16 h-1.5 rounded-full bg-muted overflow-hidden">
                                                <div class="h-full bg-primary rounded-full" style="width: {((entry.count / maxTypeCount) * 100).toFixed(0)}%"></div>
                                            </div>
                                            <span class="text-xs font-semibold text-muted-foreground">{entry.count}</span>
                                        </div>
                                    </div>
                                {/each}
                            </div>
                        </div>
                    {/if}

                    <!-- Content/question ratio -->
                    {#if allItems.length > 0}
                        <div class="flex flex-col gap-1.5">
                            <p class="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">Content / Question ratio</p>
                            <div class="h-2 rounded-full bg-muted overflow-hidden flex">
                                <div class="h-full bg-success" style="width: {contentPct.toFixed(0)}%"></div>
                                <div class="h-full bg-primary" style="width: {questionPct.toFixed(0)}%"></div>
                            </div>
                            <div class="flex items-center gap-3 text-xs text-muted-foreground">
                                <div class="flex items-center gap-1"><span class="size-2 rounded-full bg-success"></span> Content ({contentPct.toFixed(0)}%)</div>
                                <div class="flex items-center gap-1"><span class="size-2 rounded-full bg-primary"></span> Questions ({questionPct.toFixed(0)}%)</div>
                            </div>
                        </div>
                    {/if}
                </div>
            </aside>
        {/if}
    </div>
</div>

<!-- Reusable question card snippet. `showNav` controls whether the
     walk-through nav buttons render (they're rendered separately above
     in walk mode). -->
{#snippet questionCard(item: any, showNav: boolean)}
    <div class="rounded-xl border border-border bg-card p-5">
        <!-- Type label: always visible so you can tell what type a question
             was supposed to be even if the data is missing/broken. -->
        <div class="flex items-center justify-between mb-2">
            <span class="text-[10px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">
                {questionTypeLabel(item.item_type)}
            </span>
            {#if isQuestionEmpty(item)}
                <span class="inline-flex items-center gap-1 text-[10px] font-medium text-warning px-1.5 py-0.5 rounded border border-warning/40 bg-warning/10">
                    <AlertTriangle class="size-2.5" />
                    Empty data
                </span>
            {/if}
        </div>
        {#if item.item_type === "mc"}
            <p class="text-sm font-medium text-foreground mb-3">
                {item.data?.question ?? ""}
            </p>
            <div class="flex flex-col gap-2">
                {#each item.data?.options ?? [] as opt, oi}
                    <label
                        class="flex items-center gap-2.5 rounded-lg border px-3.5 py-2.5 cursor-pointer transition-colors {answers[item.id] === oi
                            ? 'border-primary bg-primary/5'
                            : 'border-border hover:border-muted-foreground/30'}"
                    >
                        <input
                            type="radio"
                            name="mc-{item.id}"
                            checked={answers[item.id] === oi}
                            onchange={() => setAnswer(item.id, oi)}
                            class="sr-only"
                        />
                        <div
                            class="size-4 rounded-full border-2 flex items-center justify-center shrink-0 {answers[item.id] === oi ? 'border-primary' : 'border-muted-foreground/30'}"
                        >
                            {#if answers[item.id] === oi}
                                <div class="size-2 rounded-full bg-primary"></div>
                            {/if}
                        </div>
                        <span class="text-sm text-foreground">{opt}</span>
                    </label>
                {/each}
            </div>
        {:else if item.item_type === "ma"}
            <p class="text-sm font-medium text-foreground mb-1">
                {item.data?.question ?? ""}
            </p>
            <p class="text-xs text-muted-foreground mb-3">(select all that apply)</p>
            <div class="flex flex-col gap-2">
                {#each item.data?.options ?? [] as opt, oi}
                    <label
                        class="flex items-center gap-2.5 rounded-lg border px-3.5 py-2.5 cursor-pointer transition-colors {(answers[item.id] ?? []).includes(oi)
                            ? 'border-primary bg-primary/5'
                            : 'border-border hover:border-muted-foreground/30'}"
                    >
                        <input
                            type="checkbox"
                            checked={(answers[item.id] ?? []).includes(oi)}
                            onchange={() => toggleMA(item.id, oi)}
                            class="sr-only"
                        />
                        <div
                            class="size-4 rounded border-2 flex items-center justify-center shrink-0 {(answers[item.id] ?? []).includes(oi) ? 'border-primary bg-primary' : 'border-muted-foreground/30'}"
                        >
                            {#if (answers[item.id] ?? []).includes(oi)}
                                <CheckCircle class="size-3 text-primary-foreground" />
                            {/if}
                        </div>
                        <span class="text-sm text-foreground">{opt}</span>
                    </label>
                {/each}
            </div>
        {:else if item.item_type === "tf"}
            <p class="text-sm font-medium text-foreground mb-3">
                {item.data?.statement ?? ""}
            </p>
            <div class="flex gap-2">
                <button
                    class="flex-1 rounded-lg border px-4 py-3 text-sm font-medium transition-colors {answers[item.id] === true
                        ? 'border-primary bg-primary/5 text-primary'
                        : 'border-border text-muted-foreground hover:border-muted-foreground/30'}"
                    onclick={() => setAnswer(item.id, true)}
                >
                    True
                </button>
                <button
                    class="flex-1 rounded-lg border px-4 py-3 text-sm font-medium transition-colors {answers[item.id] === false
                        ? 'border-primary bg-primary/5 text-primary'
                        : 'border-border text-muted-foreground hover:border-muted-foreground/30'}"
                    onclick={() => setAnswer(item.id, false)}
                >
                    False
                </button>
            </div>
        {:else if item.item_type === "fb"}
            {@const text = item.data?.text ?? item.data?.question ?? ""}
            {@const parts = text.split("___")}
            {#if parts.length > 1}
                <p class="text-sm font-medium text-foreground mb-3 leading-relaxed">
                    {#each parts as part, pi}
                        {part}
                        {#if pi < parts.length - 1}
                            <input
                                type="text"
                                value={answers[item.id]?.[pi] ?? ""}
                                oninput={(e) => {
                                    const current = answers[item.id] ?? [];
                                    current[pi] = e.currentTarget.value;
                                    answers[item.id] = [...current];
                                }}
                                class="inline-block w-32 mx-1 rounded-md border border-input bg-background px-2 py-0.5 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                                placeholder="…"
                            />
                        {/if}
                    {/each}
                </p>
            {:else}
                <!-- No ___ markers found — show as plain text with a note -->
                <p class="text-sm font-medium text-foreground mb-3 leading-relaxed">
                    {text}
                </p>
                {#if item.data?.blanks?.length}
                    <div class="flex flex-col gap-2">
                        <p class="text-xs text-muted-foreground">Fill in the blank(s):</p>
                        {#each item.data.blanks as _blank, bi}
                            <input
                                type="text"
                                value={answers[item.id]?.[bi] ?? ""}
                                oninput={(e) => {
                                    const current = answers[item.id] ?? [];
                                    current[bi] = e.currentTarget.value;
                                    answers[item.id] = [...current];
                                }}
                                class="w-48 rounded-md border border-input bg-background px-2 py-1 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                                placeholder={`Blank ${bi + 1}…`}
                            />
                        {/each}
                    </div>
                {/if}
            {/if}
        {:else if item.item_type === "sa"}
            <p class="text-sm font-medium text-foreground mb-3">
                {item.data?.question ?? ""}
            </p>
            <textarea
                value={answers[item.id] ?? ""}
                oninput={(e) => setAnswer(item.id, e.currentTarget.value)}
                rows={4}
                class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-ring resize-none"
                placeholder="Type your answer…"
            ></textarea>
            {#if checked[item.id] && item.data?.sample_answer}
                <div class="mt-2 rounded-lg border border-info/30 bg-info/5 px-3 py-2">
                    <p class="text-[10px] font-semibold uppercase tracking-wider text-info mb-1">
                        Sample answer
                    </p>
                    <p class="text-xs text-foreground/80">
                        {item.data.sample_answer}
                    </p>
                </div>
            {/if}
        {:else if item.item_type === "matching"}
            <QuestionMatching
                data={item.data}
                value={answers[item.id] ?? {}}
                reveal={revealed[item.id] || (checked[item.id] && isCorrect(item, answers[item.id]))}
                onChange={(v) => setAnswer(item.id, v)}
            />
        {:else if item.item_type === "drag_sort"}
            <QuestionOrdering
                data={item.data}
                value={answers[item.id]}
                reveal={revealed[item.id] || (checked[item.id] && isCorrect(item, answers[item.id]))}
                onChange={(v) => setAnswer(item.id, v)}
            />
        {:else if item.item_type === "hotspot"}
            <QuestionHotspot
                data={item.data}
                value={answers[item.id]}
                reveal={revealed[item.id] || (checked[item.id] && isCorrect(item, answers[item.id]))}
                onChange={(v) => setAnswer(item.id, v)}
            />
        {:else}
            <!-- Truly unsupported type — show what we can. -->
            <p class="text-sm font-medium text-foreground mb-2">
                {item.data?.question ?? item.data?.statement ?? "Question"}
            </p>
            <div class="rounded-lg border border-warning/30 bg-warning/5 px-3 py-2">
                <p class="text-xs text-warning">
                    This question type ({item.item_type}) isn’t supported in preview yet.
                </p>
            </div>
            {#if item.data?.explanation}
                <details class="mt-2">
                    <summary class="text-xs text-muted-foreground cursor-pointer hover:text-foreground">
                        Show explanation
                    </summary>
                    <p class="text-xs text-foreground/80 mt-1">{item.data.explanation}</p>
                </details>
            {/if}
        {/if}

        <!-- Check / Reveal / Feedback (shared across all question types) -->
        {#if item.item_type !== "content"}
            {#if !checked[item.id]}
                <button
                    class="mt-3 inline-flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-muted-foreground hover:border-primary hover:text-primary transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                    disabled={answers[item.id] === undefined || (Array.isArray(answers[item.id]) && answers[item.id].length === 0)}
                    onclick={() => check(item.id)}
                >
                    <CheckCircle class="size-3" />
                    Check answer
                </button>
            {/if}
            {#if checked[item.id]}
                <AnswerFeedback
                    status={answers[item.id] === undefined
                        ? "unanswered"
                        : isCorrect(item, answers[item.id])
                          ? "correct"
                          : "incorrect"}
                    correctAnswer={correctAnswerLabel(item)}
                    revealAnswer={revealed[item.id] || item.item_type === "sa"}
                >
                    {#snippet explanation()}
                        {#if item.data?.explanation && (revealed[item.id] || isCorrect(item, answers[item.id]) || item.item_type === "sa")}
                            {item.data.explanation}
                        {/if}
                    {/snippet}
                </AnswerFeedback>
                {#if !revealed[item.id] && !isCorrect(item, answers[item.id]) && item.item_type !== "sa"}
                    <button
                        class="mt-2 inline-flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-muted-foreground hover:border-info hover:text-info transition-colors"
                        onclick={() => reveal(item.id)}
                    >
                        <Eye class="size-3" />
                        Reveal answer
                    </button>
                {/if}
            {/if}
        {/if}
    </div>
{/snippet}
