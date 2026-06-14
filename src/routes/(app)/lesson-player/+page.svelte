<script lang="ts">
    import {
        GraduationCap,
        BookOpen,
        ChevronLeft,
        ChevronRight,
        CheckCircle,
        Clock,
        LoaderCircle,
        Play,
        Layers,
        Trophy,
        RotateCcw,
        ArrowRight,
        Eye,
    } from "@lucide/svelte";
    import * as Button from "$lib/components/ui/button";

    let courses = $state<any[]>([]);
    let loading = $state(true);

    // Preview mode
    let previewMode = $state(false);
    let previewId = $state<number | null>(null);
    // Player state
    let enrolledCourse = $state<any>(null);
    let playerLoading = $state(false);
    let currentModuleIdx = $state(0);
    let showResults = $state(false);

    // Answers: { [itemId]: answer }
    let answers = $state<Record<number, any>>({});

    $effect(() => {
        const params = new URLSearchParams(window.location.search);
        const previewParam = params.get("preview");
        if (previewParam) {
            previewMode = true;
            previewId = parseInt(previewParam, 10);
            if (!isNaN(previewId)) {
                loadPreview(previewId);
            }
        } else {
            loadCourses();
        }
    });

    async function loadPreview(courseId: number) {
        playerLoading = true;
        loading = false;
        try {
            const res = await fetch(`/api/courses/${courseId}/preview`, {
                credentials: "include",
            });
            if (res.ok) {
                enrolledCourse = await res.json();
            }
        } catch {
            /* ignore */
        }
        playerLoading = false;
    }

    async function loadCourses() {
        loading = true;
        try {
            const res = await fetch("/api/courses/published", {
                credentials: "include",
            });
            if (res.ok) courses = await res.json();
        } catch {
            /* ignore */
        }
        loading = false;
    }

    async function enroll(courseId: number) {
        playerLoading = true;
        enrolledCourse = null;
        currentModuleIdx = 0;
        answers = {};
        showResults = false;
        try {
            const res = await fetch(`/api/courses/${courseId}/play`, {
                credentials: "include",
            });
            if (res.ok) enrolledCourse = await res.json();
        } catch {
            /* ignore */
        }
        playerLoading = false;
    }

    function goHome() {
        enrolledCourse = null;
        previewMode = false;
        previewId = null;
        loadCourses();
    }

    function nextModule() {
        if (currentModuleIdx < (enrolledCourse.modules?.length ?? 0) - 1) {
            currentModuleIdx++;
        }
    }

    function prevModule() {
        if (currentModuleIdx > 0) currentModuleIdx--;
    }

    function finishCourse() {
        showResults = true;
    }

    function setAnswer(itemId: number, answer: any) {
        answers[itemId] = answer;
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
            default:
                return false;
        }
    }

    function computeScore(): { correct: number; total: number } {
        let correct = 0,
            total = 0;
        if (!enrolledCourse) return { correct: 0, total: 0 };
        for (const mod of enrolledCourse.modules) {
            if (!mod?.items) continue;
            for (const item of mod.items) {
                if (item.item_type === "content") continue;
                total++;
                if (isCorrect(item, answers[item.id])) correct++;
            }
        }
        return { correct, total };
    }

    function moduleProgress(mod: any): number {
        let done = 0,
            total = 0;
        if (!mod?.items) return 100;
        for (const item of mod.items) {
            if (item.item_type === "content") continue;
            total++;
            if (answers[item.id] !== undefined) done++;
        }
        return total === 0 ? 100 : Math.round((done / total) * 100);
    }

    let currentModule = $derived(enrolledCourse?.modules?.[currentModuleIdx]);
</script>

<!-- Course List -->
{#if !enrolledCourse}
    <div class="flex w-full max-w-6xl mx-auto flex-col gap-6">
        <div class="flex items-center gap-3">
            <GraduationCap class="size-6 text-primary" />
            <h1 class="text-2xl font-semibold tracking-tight text-foreground">
                Guided Lesson Player
            </h1>
        </div>

        {#if loading}
            <div class="flex justify-center py-16">
                <LoaderCircle
                    class="size-6 text-muted-foreground animate-spin"
                />
            </div>
        {:else if courses.length === 0}
            <div
                class="rounded-xl border border-border bg-card p-12 text-center"
            >
                <BookOpen
                    class="size-10 text-muted-foreground/40 mx-auto mb-3"
                />
                <p class="text-sm text-muted-foreground">
                    No approved courses available yet. Courses must be published
                    and approved to appear here.
                </p>
            </div>
        {:else}
            <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                {#each courses as course}
                    <div
                        class="rounded-xl border border-border bg-card p-6 flex flex-col gap-4 hover:border-primary/30 transition-colors"
                    >
                        <div class="flex items-start gap-3">
                            <BookOpen
                                class="size-6 text-primary shrink-0 mt-0.5"
                            />
                            <div class="min-w-0">
                                <h3
                                    class="text-base font-semibold text-foreground leading-snug"
                                >
                                    {course.title}
                                </h3>
                                <p
                                    class="text-sm text-muted-foreground mt-1.5 line-clamp-2"
                                >
                                    {course.description}
                                </p>
                            </div>
                        </div>

                        {#if course.progress}
                            <div class="space-y-2">
                                <div
                                    class="flex items-center justify-between text-xs"
                                >
                                    <span class="text-muted-foreground"
                                        >Module {course.progress
                                            .current_module + 1}</span
                                    >
                                    <span class="text-muted-foreground"
                                        >{course.progress.completed
                                            ? "Completed"
                                            : "In progress"}</span
                                    >
                                </div>
                                <div
                                    class="h-2 w-full rounded-full bg-muted overflow-hidden"
                                >
                                    <div
                                        class="h-full rounded-full bg-primary transition-all"
                                        style="width: {course.progress.completed
                                            ? '100%'
                                            : Math.min(
                                                  (course.progress
                                                      .current_module /
                                                      5) *
                                                      100,
                                                  90,
                                              ) + '%'}"
                                    ></div>
                                </div>
                            </div>
                        {/if}

                        <div
                            class="flex items-center gap-3 text-xs text-muted-foreground"
                        >
                            <span class="flex items-center gap-1"
                                ><Layers class="size-3.5" />
                                {course.source_doc_ids?.length ?? 0} sources</span
                            >
                            {#if course.progress?.completed}
                                <span
                                    class="flex items-center gap-1 text-emerald-500"
                                    ><CheckCircle class="size-3.5" />
                                    {course.progress.score_pct}%</span
                                >
                            {/if}
                        </div>

                        {#if course.progress?.completed}
                            <Button.Root
                                variant="outline"
                                size="default"
                                class="w-full mt-auto"
                                onclick={() => enroll(course.id)}
                            >
                                <RotateCcw class="size-4 mr-1.5" /> Retake
                            </Button.Root>
                        {:else if course.progress}
                            <Button.Root
                                size="default"
                                class="w-full mt-auto"
                                onclick={() => enroll(course.id)}
                            >
                                <Play class="size-4 mr-1.5" /> Continue
                            </Button.Root>
                        {:else}
                            <Button.Root
                                size="default"
                                class="w-full mt-auto"
                                onclick={() => enroll(course.id)}
                            >
                                <Play class="size-4 mr-1.5" /> Enroll
                            </Button.Root>
                        {/if}
                    </div>
                {/each}
            </div>
        {/if}
    </div>

    <!-- Player View -->
{:else if playerLoading}
    <div class="flex justify-center py-16">
        <LoaderCircle class="size-6 text-muted-foreground animate-spin" />
    </div>
{:else if showResults}
    {@const score = computeScore()}
    <div
        class="flex w-full max-w-2xl mx-auto flex-col items-center gap-6 py-12"
    >
        <Trophy class="size-16 text-amber-500" />
        <h1 class="text-2xl font-semibold text-foreground">Course Complete!</h1>
        <p class="text-muted-foreground">{enrolledCourse.title}</p>
        <div class="text-5xl font-bold text-foreground">
            {Math.round((score.correct / Math.max(score.total, 1)) * 100)}%
        </div>
        <p class="text-sm text-muted-foreground">
            {score.correct} of {score.total} correct
        </p>
        <div class="flex gap-3">
            <Button.Root variant="outline" onclick={goHome}
                >Back to Courses</Button.Root
            >
            <Button.Root
                onclick={() => {
                    showResults = false;
                    currentModuleIdx = 0;
                }}>Review Answers</Button.Root
            >
        </div>
    </div>
{:else}
    {#if previewMode}
        <div
            class="w-full rounded-lg border border-blue-500/30 bg-blue-500/5 px-4 py-2.5 mb-4 flex items-center justify-between"
        >
            <div class="flex items-center gap-2">
                <Eye class="size-4 text-blue-500" />
                <span
                    class="text-sm font-medium text-blue-600 dark:text-blue-400"
                    >Preview Mode</span
                >
                <span class="text-xs text-blue-500/70"
                    >— this is a course preview, answers are not saved</span
                >
            </div>
            <Button.Root variant="outline" size="sm" onclick={goHome}>
                <ChevronLeft class="size-3.5 mr-1" /> Back to Editor
            </Button.Root>
        </div>
    {/if}
    <div class="flex w-full max-w-6xl mx-auto gap-6">
        <!-- Module Sidebar -->
        <aside class="w-[240px] shrink-0 flex flex-col gap-2">
            <button
                class="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground mb-2"
                onclick={goHome}
            >
                <ChevronLeft class="size-4" /> Back
            </button>
            <h3 class="text-sm font-semibold text-foreground mb-1">
                {enrolledCourse.title}
            </h3>
            {#each enrolledCourse.modules as mod, mi}
                <button
                    class="flex items-center gap-2 rounded-lg px-3 py-2 text-left text-sm transition-colors {mi ===
                    currentModuleIdx
                        ? 'bg-primary/10 text-foreground font-medium'
                        : 'text-muted-foreground hover:bg-muted hover:text-foreground'}"
                    onclick={() => (currentModuleIdx = mi)}
                >
                    <div
                        class="size-5 rounded-full border flex items-center justify-center shrink-0 text-[10px] font-bold
                        {mi === currentModuleIdx
                            ? 'border-primary text-primary'
                            : 'border-border text-muted-foreground'}"
                    >
                        {moduleProgress(mod) === 100 ? "✓" : mi + 1}
                    </div>
                    <span class="truncate">{mod.title}</span>
                </button>
            {/each}
        </aside>

        <!-- Main Content -->
        <main class="flex-1 min-w-0">
            {#if currentModule}
                <div class="rounded-xl border border-border bg-card p-6 mb-4">
                    <h2 class="text-lg font-semibold text-foreground">
                        {currentModule.title}
                    </h2>
                    <p class="text-sm text-muted-foreground mt-1">
                        {currentModule.description}
                    </p>
                </div>

                <div class="flex flex-col gap-4">
                    {#each currentModule.items as item, ii}
                        {#if item.item_type === "content"}
                            <div
                                class="rounded-xl border border-border bg-card p-5"
                            >
                                <p
                                    class="text-[10px] font-semibold uppercase text-muted-foreground mb-2"
                                >
                                    Learning Material
                                </p>
                                <div
                                    class="text-sm text-foreground leading-relaxed whitespace-pre-line"
                                >
                                    {item.data?.body ?? ""}
                                </div>
                            </div>
                        {:else if item.item_type === "mc"}
                            <div
                                class="rounded-xl border border-border bg-card p-5"
                            >
                                <p
                                    class="text-sm font-medium text-foreground mb-3"
                                >
                                    {ii + 1}. {item.data?.question ?? ""}
                                </p>
                                <div class="flex flex-col gap-2">
                                    {#each item.data?.options ?? [] as opt, oi}
                                        <label
                                            class="flex items-center gap-2.5 rounded-lg border px-3.5 py-2.5 cursor-pointer transition-colors {answers[
                                                item.id
                                            ] === oi
                                                ? 'border-primary bg-primary/5'
                                                : 'border-border hover:border-muted-foreground/30'}"
                                        >
                                            <input
                                                type="radio"
                                                name="mc-{item.id}"
                                                checked={answers[item.id] ===
                                                    oi}
                                                onchange={() =>
                                                    setAnswer(item.id, oi)}
                                                class="sr-only"
                                            />
                                            <div
                                                class="size-4 rounded-full border-2 flex items-center justify-center shrink-0 {answers[
                                                    item.id
                                                ] === oi
                                                    ? 'border-primary'
                                                    : 'border-muted-foreground/30'}"
                                            >
                                                {#if answers[item.id] === oi}<div
                                                        class="size-2 rounded-full bg-primary"
                                                    ></div>{/if}
                                            </div>
                                            <span
                                                class="text-sm text-foreground"
                                                >{opt}</span
                                            >
                                        </label>
                                    {/each}
                                </div>
                                {#if answers[item.id] !== undefined && showResults}
                                    <p
                                        class="mt-2 text-xs {isCorrect(
                                            item,
                                            answers[item.id],
                                        )
                                            ? 'text-emerald-500'
                                            : 'text-red-500'}"
                                    >
                                        {isCorrect(item, answers[item.id])
                                            ? "✓ Correct"
                                            : "✗ Incorrect"} — {item.data
                                            ?.explanation ?? ""}
                                    </p>
                                {/if}
                            </div>
                        {:else if item.item_type === "ma"}
                            <div
                                class="rounded-xl border border-border bg-card p-5"
                            >
                                <p
                                    class="text-sm font-medium text-foreground mb-3"
                                >
                                    {ii + 1}. {item.data?.question ?? ""}
                                    <span class="text-xs text-muted-foreground"
                                        >(select all that apply)</span
                                    >
                                </p>
                                <div class="flex flex-col gap-2">
                                    {#each item.data?.options ?? [] as opt, oi}
                                        <label
                                            class="flex items-center gap-2.5 rounded-lg border px-3.5 py-2.5 cursor-pointer transition-colors {(
                                                answers[item.id] ?? []
                                            ).includes(oi)
                                                ? 'border-primary bg-primary/5'
                                                : 'border-border hover:border-muted-foreground/30'}"
                                        >
                                            <input
                                                type="checkbox"
                                                checked={(
                                                    answers[item.id] ?? []
                                                ).includes(oi)}
                                                onchange={(e) => {
                                                    const cur =
                                                        answers[item.id] ?? [];
                                                    const target =
                                                        e.target as HTMLInputElement;
                                                    setAnswer(
                                                        item.id,
                                                        target.checked
                                                            ? [...cur, oi]
                                                            : cur.filter(
                                                                  (v: number) =>
                                                                      v !== oi,
                                                              ),
                                                    );
                                                }}
                                                class="sr-only"
                                            />
                                            <div
                                                class="size-4 rounded border-2 flex items-center justify-center shrink-0 {(
                                                    answers[item.id] ?? []
                                                ).includes(oi)
                                                    ? 'bg-primary border-primary text-primary-foreground'
                                                    : 'border-muted-foreground/30'}"
                                            >
                                                {#if (answers[item.id] ?? []).includes(oi)}<CheckCircle
                                                        class="size-3"
                                                    />{/if}
                                            </div>
                                            <span
                                                class="text-sm text-foreground"
                                                >{opt}</span
                                            >
                                        </label>
                                    {/each}
                                </div>
                            </div>
                        {:else if item.item_type === "tf"}
                            <div
                                class="rounded-xl border border-border bg-card p-5"
                            >
                                <p
                                    class="text-sm font-medium text-foreground mb-3"
                                >
                                    {ii + 1}. {item.data?.statement ?? ""}
                                </p>
                                <div class="flex gap-3">
                                    {#each [true, false] as val}
                                        <button
                                            class="flex-1 rounded-lg border px-4 py-2.5 text-sm font-medium transition-colors {answers[
                                                item.id
                                            ] === val
                                                ? 'border-primary bg-primary/5 text-primary'
                                                : 'border-border text-muted-foreground hover:border-muted-foreground/30'}"
                                            onclick={() =>
                                                setAnswer(item.id, val)}
                                        >
                                            {val ? "True" : "False"}
                                        </button>
                                    {/each}
                                </div>
                            </div>
                        {:else if item.item_type === "fb"}
                            <div
                                class="rounded-xl border border-border bg-card p-5"
                            >
                                <p
                                    class="text-sm font-medium text-foreground mb-3"
                                >
                                    {ii + 1}. Fill in the blanks:
                                </p>
                                <p
                                    class="text-sm text-foreground leading-relaxed mb-3 whitespace-pre-line"
                                >
                                    {item.data?.text ?? ""}
                                </p>
                                <div class="flex flex-col gap-2">
                                    {#each item.data?.blanks ?? [] as blank, bi}
                                        <input
                                            type="text"
                                            placeholder="Answer {bi + 1}"
                                            value={answers[item.id]?.[bi] ?? ""}
                                            oninput={(e) => {
                                                const cur = [
                                                    ...(answers[item.id] ?? []),
                                                ];
                                                cur[bi] = (
                                                    e.target as HTMLInputElement
                                                ).value;
                                                setAnswer(item.id, cur);
                                            }}
                                            class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                                        />
                                    {/each}
                                </div>
                            </div>
                        {:else if item.item_type === "sa"}
                            <div
                                class="rounded-xl border border-border bg-card p-5"
                            >
                                <p
                                    class="text-sm font-medium text-foreground mb-3"
                                >
                                    {ii + 1}. {item.data?.question ?? ""}
                                </p>
                                <textarea
                                    rows={3}
                                    value={answers[item.id] ?? ""}
                                    oninput={(e) =>
                                        setAnswer(
                                            item.id,
                                            (e.target as HTMLTextAreaElement)
                                                .value,
                                        )}
                                    placeholder="Type your answer..."
                                    class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring resize-none"
                                ></textarea>
                                {#if showResults && item.data?.sample_answer}
                                    <p
                                        class="mt-2 text-xs text-muted-foreground"
                                    >
                                        Sample answer: {item.data.sample_answer}
                                    </p>
                                {/if}
                            </div>
                        {:else}
                            <div
                                class="rounded-xl border border-border bg-card p-5"
                            >
                                <p class="text-xs text-muted-foreground">
                                    {item.item_type} — interactive component coming
                                    soon
                                </p>
                            </div>
                        {/if}
                    {/each}
                </div>

                <!-- Navigation -->
                <div class="flex items-center justify-between mt-6">
                    <Button.Root
                        variant="ghost"
                        size="sm"
                        disabled={currentModuleIdx === 0}
                        onclick={prevModule}
                    >
                        <ChevronLeft class="size-4 mr-1" /> Previous
                    </Button.Root>
                    <span class="text-xs text-muted-foreground"
                        >Module {currentModuleIdx + 1} of {enrolledCourse
                            .modules?.length ?? 0}</span
                    >
                    {#if currentModuleIdx < (enrolledCourse.modules?.length ?? 0) - 1}
                        <Button.Root size="sm" onclick={nextModule}>
                            Next <ChevronRight class="size-4 ml-1" />
                        </Button.Root>
                    {:else}
                        <Button.Root size="sm" onclick={finishCourse}>
                            <Trophy class="size-4 mr-1.5" /> Finish
                        </Button.Root>
                    {/if}
                </div>
            {/if}
        </main>
    </div>
{/if}
