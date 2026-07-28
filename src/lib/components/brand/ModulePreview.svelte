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
            <aside class="w-64 shrink-0 border-r border-border bg-card/50 overflow-y-auto">
                <div class="p-3">
                    <p class="text-[10px] font-semibold uppercase tracking-[0.18em] text-muted-foreground mb-2 px-1">
                        Modules ({sortedModules.length})
                    </p>
                    <div class="flex flex-col gap-0.5">
                        {#each sortedModules as mod, mi (mod.id)}
                            {@const status = moduleStatusIcon(mod)}
                            <button
                                class="flex items-start gap-2 rounded-lg px-2.5 py-2 text-left transition-colors {mod.id === module?.id ? 'bg-primary/10 text-foreground' : 'text-muted-foreground hover:bg-muted hover:text-foreground'}"
                                onclick={() => switchToModule(mod)}
                                title={mod.title}
                            >
                                <status.Icon class="size-3.5 shrink-0 mt-0.5 {mod.id === module?.id ? 'text-primary' : status.class}" />
                                <div class="min-w-0 flex-1">
                                    <p class="text-[10px] font-medium text-muted-foreground">
                                        Module {mi + 1}
                                    </p>
                                    <p class="text-xs font-medium truncate">
                                        {mod.title}
                                    </p>
                                    <p class="text-[10px] text-muted-foreground/70 mt-0.5">
                                        {mod.items?.length ?? 0} items
                                    </p>
                                </div>
                            </button>
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
    </div>
</div>

<!-- Reusable question card snippet. `showNav` controls whether the
     walk-through nav buttons render (they're rendered separately above
     in walk mode). -->
{#snippet questionCard(item: any, showNav: boolean)}
    <div class="rounded-xl border border-border bg-card p-5">
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
            {@const text = item.data?.text ?? ""}
            {@const parts = text.split("___")}
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
