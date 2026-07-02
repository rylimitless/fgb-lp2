<script lang="ts">
    import {
        GraduationCap,
        BookOpen,
        ChevronLeft,
        ChevronRight,
        ChevronDown,
        ChevronUp,
        CheckCircle,
        Clock,
        LoaderCircle,
        Play,
        Layers,
        Trophy,
        RotateCcw,
        ArrowRight,
        Eye,
        AlertTriangle,
        Sparkles,
        Search,
        X,
        Printer,
        Settings,
    } from "@lucide/svelte";
    import * as Button from "$lib/components/ui/button";
    import {
        AnswerFeedback,
        BadgeMedal,
        Confetti,
        CourseCard,
        GiaAvatar,
        Spotlight,
        NumberTicker,
        PageHeader,
        EmptyState,
        LoadingDots,
        XpRing,
        Markdown,
        QuestionMatching,
        QuestionOrdering,
        QuestionHotspot,
    } from "$lib/components/brand";

    let { data } = $props();
    let userRoles: string[] = $derived(data?.user?.roles ?? []);
    let isContentCreator = $derived(
        userRoles.includes("content creator") || userRoles.includes("admin"),
    );

    let courses = $state<any[]>([]);
    let loading = $state(true);

    // Catalogue filter state
    let search = $state("");
    let filter = $state<"all" | "in_progress" | "completed" | "new">("all");

    // Preview mode
    let previewMode = $state(false);
    let previewId = $state<number | null>(null);
    // Player state
    let enrolledCourse = $state<any>(null);
    let playerLoading = $state(false);
    let currentModuleIdx = $state(0);
    let showResults = $state(false);
    let saving = $state(false);

    // Confirmation dialog for skipping unanswered questions
    let skipConfirmOpen = $state(false);
    let pendingNavigation: (() => void) | null = $state(null);

    // Settings editor state (for content creators)
    let settingsEditOpen = $state(false);
    let editMaxAttempts = $state<number | null>(null);
    let editDaysToComplete = $state<number | null>(null);
    let editGraded = $state(false);
    let settingsSaving = $state(false);
    let settingsSaveError = $state("");
    let settingsSaved = $state(false);

    function parseCourseSettings(raw: any): any {
        if (!raw || raw === "{}") return {};
        try {
            return typeof raw === "string" ? JSON.parse(raw) : raw;
        } catch {
            return {};
        }
    }
    function openSettingsEditor() {
        const s = parseCourseSettings(enrolledCourse?.settings);
        editMaxAttempts = s.max_attempts ?? null;
        editDaysToComplete = s.days_to_complete ?? null;
        editGraded = s.graded === true;
        settingsEditOpen = true;
        settingsSaveError = "";
        settingsSaved = false;
    }
    async function saveCourseSettings() {
        if (!enrolledCourse) return;
        settingsSaving = true;
        settingsSaveError = "";
        settingsSaved = false;
        try {
            const clean: any = {};
            if (editMaxAttempts && editMaxAttempts > 0)
                clean.max_attempts = editMaxAttempts;
            if (editDaysToComplete && editDaysToComplete > 0)
                clean.days_to_complete = editDaysToComplete;
            clean.graded = editGraded;
            const res = await fetch(
                `/api/courses/${enrolledCourse.id}/settings`,
                {
                    method: "PUT",
                    headers: { "Content-Type": "application/json" },
                    credentials: "include",
                    body: JSON.stringify({ settings: clean }),
                },
            );
            if (!res.ok) {
                const err = await res.json();
                throw new Error(err.error || "Failed to save");
            }
            const updated = await res.json();
            enrolledCourse.settings = updated.settings;
            settingsSaved = true;
            setTimeout(() => (settingsSaved = false), 3000);
        } catch (e: any) {
            settingsSaveError = e.message || "Network error";
        } finally {
            settingsSaving = false;
        }
    }

    // Answers: { [itemId]: answer }
    let answers = $state<Record<number, any>>({});
    // Which items have had their answer checked (correct/incorrect shown)
    let checked = $state<Record<number, boolean>>({});
    // Which items have had their correct answer revealed (separate from check in learning mode)
    let revealed = $state<Record<number, boolean>>({});

    // Derived: is the course in learning (non-graded) mode?
    let isGraded = $derived(
        parseCourseSettings(enrolledCourse?.settings).graded === true,
    );

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

    async function saveProgress(completed: boolean = false) {
        if (previewMode || saving || !enrolledCourse) return;
        saving = true;
        try {
            const score = computeScore();
            const pct =
                score.total === 0
                    ? "0"
                    : ((score.correct / score.total) * 100).toFixed(2);
            const body: any = {
                course_id: enrolledCourse.id,
                current_module: currentModuleIdx,
                completed,
                score_pct: pct,
            };
            await fetch("/api/lessons/progress", {
                method: "POST",
                credentials: "include",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(body),
            });

            // When course is completed, award leaderboard score
            if (completed) {
                const courseScore = Math.min(100, score.correct);
                await fetch("/api/gamification/score", {
                    method: "POST",
                    credentials: "include",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify({
                        course_id: enrolledCourse.id,
                        score: courseScore,
                        completed: true,
                    }),
                }).catch(() => {});
            }
        } catch {
            /* ignore */
        }
        saving = false;
    }

    async function enroll(courseId: number) {
        playerLoading = true;
        enrolledCourse = null;
        currentModuleIdx = 0;
        answers = {};
        checked = {};
        revealed = {};
        showResults = false;
        // Check if this is a retake (course was previously completed)
        const existing = courses.find((c: any) => c.id === courseId);
        const isRetake = existing?.progress?.completed === true;
        try {
            const url =
                `/api/courses/${courseId}/play` +
                (isRetake ? "?retake=true" : "");
            const res = await fetch(url, {
                credentials: "include",
            });
            if (res.ok) {
                enrolledCourse = await res.json();
                // Check if blocked by course settings
                if (enrolledCourse.blocked) {
                    // enrolledCourse stays set so the UI can show the blocked message
                    return;
                }
                // Restore saved item-level answers
                if (enrolledCourse.modules) {
                    for (const mod of enrolledCourse.modules) {
                        if (!mod?.items) continue;
                        for (const item of mod.items) {
                            if (item.saved_answer !== undefined) {
                                answers[item.id] = item.saved_answer;
                                // If previously answered, mark as checked and revealed
                                checked[item.id] = true;
                                revealed[item.id] = true;
                            }
                        }
                    }
                }
                // Resume from saved progress if available
                const prog = enrolledCourse.progress;
                if (
                    prog &&
                    !prog.completed &&
                    prog.current_module !== undefined
                ) {
                    const savedModule = prog.current_module;
                    const maxModule = (enrolledCourse.modules?.length ?? 1) - 1;
                    currentModuleIdx = Math.min(savedModule, maxModule);
                }
            }
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

    function unansweredCount(mod: any): number {
        let unanswered = 0;
        if (!mod?.items) return 0;
        for (const item of mod.items) {
            if (item.item_type === "content") continue;
            if (answers[item.id] === undefined) unanswered++;
        }
        return unanswered;
    }

    function tryNavigate(fn: () => void) {
        const mod = enrolledCourse?.modules?.[currentModuleIdx];
        const unanswered = unansweredCount(mod);
        if (unanswered > 0) {
            pendingNavigation = fn;
            skipConfirmOpen = true;
        } else {
            fn();
        }
    }

    function confirmSkip() {
        skipConfirmOpen = false;
        if (pendingNavigation) {
            pendingNavigation();
            pendingNavigation = null;
        }
    }

    function cancelSkip() {
        skipConfirmOpen = false;
        pendingNavigation = null;
    }

    function nextModule() {
        if (currentModuleIdx < (enrolledCourse.modules?.length ?? 0) - 1) {
            const nav = () => {
                currentModuleIdx++;
                saveProgress(false);
            };
            tryNavigate(nav);
        }
    }

    function prevModule() {
        if (currentModuleIdx > 0) {
            const nav = () => {
                currentModuleIdx--;
                saveProgress(false);
            };
            tryNavigate(nav);
        }
    }

    function finishCourse() {
        const nav = () => {
            showResults = true;
            saveProgress(true);
        };
        tryNavigate(nav);
    }

    function setAnswer(itemId: number, answer: any) {
        answers[itemId] = answer;
        saveItemAnswer(itemId, answer).catch(() => {});
    }

    async function saveItemAnswer(itemId: number, answer: any) {
        if (previewMode || !enrolledCourse) return;
        const mods = enrolledCourse.modules;
        let item: any;
        for (const m of mods) {
            item = (m?.items ?? []).find((it: any) => it.id === itemId);
            if (item) break;
        }
        const correct = item ? isCorrect(item, answer) : null;
        try {
            await fetch("/api/lessons/item-progress", {
                method: "POST",
                credentials: "include",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    course_id: enrolledCourse.id,
                    item_id: itemId,
                    answer,
                    is_correct: correct,
                }),
            });
        } catch {
            /* ignore */
        }
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
            case "matching": {
                const pairs = d.pairs ?? [];
                if (!pairs.length || !answer || typeof answer !== "object")
                    return false;
                return pairs.every(
                    (_p: any, index: number) => answer[String(index)] === index,
                );
            }
            case "drag_sort": {
                const items = d.items ?? [];
                if (!items.length || !Array.isArray(answer)) return false;
                return (
                    answer.length === items.length &&
                    answer.every((v: number, i: number) => v === i)
                );
            }
            case "hotspot": {
                const regions = d.regions ?? [];
                if (typeof answer !== "number") return false;
                return Boolean(regions[answer]?.correct);
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
            case "matching":
                return (d.pairs ?? [])
                    .map((p: any) => `${p.left} → ${p.right}`)
                    .join("; ");
            case "drag_sort":
                return (d.items ?? []).join(" → ");
            case "hotspot":
                return (
                    (d.regions ?? []).find((r: any) => r.correct)?.label ?? ""
                );
            default:
                return "";
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

    function moduleProgress(mod: any): {
        done: number;
        total: number;
        pct: number;
    } {
        let done = 0,
            total = 0;
        if (!mod?.items) return { done: 0, total: 0, pct: 100 };
        for (const item of mod.items) {
            if (item.item_type === "content") continue;
            total++;
            if (answers[item.id] !== undefined) done++;
        }
        const pct = total === 0 ? 100 : Math.round((done / total) * 100);
        return { done, total, pct };
    }

    let currentModule = $derived(enrolledCourse?.modules?.[currentModuleIdx]);

    // ---------------------------------------------------------------------
    // Catalogue derivations
    // ---------------------------------------------------------------------

    function matchesSearch(c: any, q: string): boolean {
        if (!q) return true;
        const needle = q.toLowerCase();
        return (
            (c.title ?? "").toLowerCase().includes(needle) ||
            (c.description ?? "").toLowerCase().includes(needle)
        );
    }

    function courseState(
        c: any,
    ): "completed" | "in_progress" | "new" | "expired" {
        if (c.progress?.completed) return "completed";
        if (c.progress?.is_expired) return "expired";
        if (c.progress?.current_module !== undefined) return "in_progress";
        return "new";
    }

    let filteredCourses = $derived(
        courses.filter(
            (c) =>
                matchesSearch(c, search) &&
                (filter === "all" || courseState(c) === filter),
        ),
    );

    let continueCourses = $derived(
        filteredCourses.filter((c) => courseState(c) === "in_progress"),
    );
    let newCourses = $derived(
        filteredCourses.filter((c) => courseState(c) === "new"),
    );
    let completedCourses = $derived(
        filteredCourses.filter((c) => courseState(c) === "completed"),
    );

    const questionTypeLabels: Record<string, string> = {
        mc: "Multiple choice",
        ma: "Multiple answer",
        tf: "True/false",
        fb: "Fill blanks",
        sa: "Short answer",
        matching: "Matching",
        drag_sort: "Ordering",
        hotspot: "Hotspot",
    };

    let assessmentMix = $derived.by(() => {
        const counts: Record<string, number> = {};
        for (const mod of enrolledCourse?.modules ?? []) {
            for (const item of mod.items ?? []) {
                if (item.item_type === "content") continue;
                counts[item.item_type] = (counts[item.item_type] ?? 0) + 1;
            }
        }
        const total = Object.values(counts).reduce((sum, n) => sum + n, 0);
        return Object.entries(counts)
            .map(([type, count]) => ({
                type,
                label: questionTypeLabels[type] ?? type,
                count,
                pct: total === 0 ? 0 : Math.round((count / total) * 100),
            }))
            .sort((a, b) => b.count - a.count);
    });
</script>

{#if skipConfirmOpen}
    <!-- Skip Confirmation Dialog -->
    <div
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
    >
        <div
            class="rounded-xl border border-border bg-card p-6 max-w-md w-full mx-4 shadow-xl"
        >
            <div class="flex items-start gap-3 mb-4">
                <AlertTriangle class="size-5 text-warning shrink-0 mt-0.5" />
                <div>
                    <h3 class="text-sm font-semibold text-foreground">
                        Unanswered Questions
                    </h3>
                    <p class="text-sm text-muted-foreground mt-1">
                        You have {unansweredCount(
                            enrolledCourse?.modules?.[currentModuleIdx],
                        )} unanswered question{unansweredCount(
                            enrolledCourse?.modules?.[currentModuleIdx],
                        ) !== 1
                            ? "s"
                            : ""} in this module. Are you sure you want to skip them?
                    </p>
                </div>
            </div>
            <div class="flex justify-end gap-3">
                <Button.Root variant="outline" size="sm" onclick={cancelSkip}>
                    Go Back
                </Button.Root>
                <Button.Root size="sm" onclick={confirmSkip}>
                    Skip & Continue
                </Button.Root>
            </div>
        </div>
    </div>
{/if}

<!-- Course List -->
{#if !enrolledCourse}
    <div class="flex w-full max-w-6xl mx-auto flex-col gap-6">
        <PageHeader
            title="Course library"
            eyebrow="Lesson player"
            description="Browse approved courses. Continue where you left off, or pick a new pathway."
        >
            {#snippet icon()}
                <GraduationCap class="size-6 text-primary" />
            {/snippet}
        </PageHeader>

        {#if loading}
            <div
                class="flex flex-col items-center gap-3 py-16 text-muted-foreground"
            >
                <LoadingDots label="Loading courses" />
            </div>
        {:else if courses.length === 0}
            <EmptyState
                title="No approved courses yet"
                description="Courses must be published and approved to appear here. Ask a content creator to publish one, or generate one in the AI Generator."
            />
        {:else}
            <!-- Search + filter bar -->
            <div
                class="flex flex-wrap items-center gap-3 rounded-2xl border border-border bg-card p-3"
                role="search"
            >
                <div class="relative flex-1 min-w-[200px]">
                    <Search
                        class="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
                    />
                    <input
                        bind:value={search}
                        type="search"
                        placeholder="Search by title or description"
                        aria-label="Search courses"
                        class="placeholder:text-muted-foreground/60 flex h-9 w-full rounded-md border border-input bg-background pl-9 pr-9 text-sm text-foreground outline-none transition-colors focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/30"
                    />
                    {#if search}
                        <button
                            type="button"
                            onclick={() => (search = "")}
                            aria-label="Clear search"
                            class="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
                        >
                            <X class="size-3.5" />
                        </button>
                    {/if}
                </div>

                <div
                    class="inline-flex items-center rounded-full border border-border bg-background p-0.5"
                    role="tablist"
                    aria-label="Filter by status"
                >
                    {#each [{ key: "all", label: "All" }, { key: "in_progress", label: "In progress" }, { key: "new", label: "Not started" }, { key: "completed", label: "Completed" }] as f}
                        <button
                            type="button"
                            role="tab"
                            aria-selected={filter === f.key}
                            class={`rounded-full px-3 py-1.5 text-xs font-medium transition-colors ${filter === f.key ? "bg-primary text-primary-foreground" : "text-muted-foreground hover:text-foreground"}`}
                            onclick={() => (filter = f.key as typeof filter)}
                        >
                            {f.label}
                        </button>
                    {/each}
                </div>
            </div>

            {#if filteredCourses.length === 0}
                <EmptyState
                    title="Nothing matches that"
                    description="Try a shorter search or switch the filter to 'All'."
                />
            {:else}
                <!-- Category bands -->
                {#if continueCourses.length > 0 && (filter === "all" || filter === "in_progress")}
                    <section aria-labelledby="band-continue">
                        <div class="flex items-baseline justify-between mb-3">
                            <h2
                                id="band-continue"
                                class="text-sm font-semibold text-foreground inline-flex items-center gap-2"
                            >
                                <Play class="size-4 text-accent" />
                                Continue learning
                            </h2>
                            <span
                                class="text-[10px] uppercase tracking-wider text-muted-foreground tabular"
                            >
                                {continueCourses.length}
                            </span>
                        </div>
                        <div
                            class="flex gap-4 overflow-x-auto snap-x snap-mandatory pb-2 -mx-2 px-2 [scroll-padding-inline:0.5rem]"
                            role="list"
                        >
                            {#each continueCourses as course, i (course.id)}
                                <div
                                    class="snap-start motion-rise-in"
                                    style="animation-delay: {Math.min(
                                        i * 30,
                                        200,
                                    )}ms"
                                    role="listitem"
                                >
                                    <CourseCard
                                        {course}
                                        compact
                                        onenroll={enroll}
                                    />
                                </div>
                            {/each}
                        </div>
                    </section>
                {/if}

                {#if newCourses.length > 0 && (filter === "all" || filter === "new")}
                    <section aria-labelledby="band-new">
                        <div class="flex items-baseline justify-between mb-3">
                            <h2
                                id="band-new"
                                class="text-sm font-semibold text-foreground inline-flex items-center gap-2"
                            >
                                <Sparkles class="size-4 text-accent" />
                                Recommended for you
                            </h2>
                            <span
                                class="text-[10px] uppercase tracking-wider text-muted-foreground tabular"
                            >
                                {newCourses.length}
                            </span>
                        </div>
                        <div
                            class="flex gap-4 overflow-x-auto snap-x snap-mandatory pb-2 -mx-2 px-2 [scroll-padding-inline:0.5rem]"
                            role="list"
                        >
                            {#each newCourses as course, i (course.id)}
                                <div
                                    class="snap-start motion-rise-in"
                                    style="animation-delay: {Math.min(
                                        i * 30,
                                        200,
                                    )}ms"
                                    role="listitem"
                                >
                                    <CourseCard
                                        {course}
                                        compact
                                        onenroll={enroll}
                                    />
                                </div>
                            {/each}
                        </div>
                    </section>
                {/if}

                {#if completedCourses.length > 0 && (filter === "all" || filter === "completed")}
                    <section aria-labelledby="band-done">
                        <div class="flex items-baseline justify-between mb-3">
                            <h2
                                id="band-done"
                                class="text-sm font-semibold text-foreground inline-flex items-center gap-2"
                            >
                                <CheckCircle class="size-4 text-success" />
                                Completed
                            </h2>
                            <span
                                class="text-[10px] uppercase tracking-wider text-muted-foreground tabular"
                            >
                                {completedCourses.length}
                            </span>
                        </div>
                        <div
                            class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4"
                            role="list"
                        >
                            {#each completedCourses as course, i (course.id)}
                                <div
                                    class="motion-rise-in"
                                    style="animation-delay: {Math.min(
                                        i * 30,
                                        200,
                                    )}ms"
                                    role="listitem"
                                >
                                    <CourseCard {course} onenroll={enroll} />
                                </div>
                            {/each}
                        </div>
                    </section>
                {/if}
            {/if}
        {/if}
    </div>

    <!-- Player View -->
{:else if playerLoading}
    <div class="flex justify-center py-16">
        <LoaderCircle class="size-6 text-muted-foreground animate-spin" />
    </div>
{:else if showResults}
    {@const score = computeScore()}
    {@const pct = Math.round((score.correct / Math.max(score.total, 1)) * 100)}
    {@const tier = pct >= 90 ? "gold" : pct >= 75 ? "silver" : "bronze"}
    {@const headline =
        pct >= 90
            ? "Outstanding."
            : pct >= 75
              ? "Strong work."
              : pct >= 50
                ? "Good progress."
                : "Keep going."}

    <!-- Celebration overlay — confetti only on pass (>= 50%) and non-preview -->
    {#if !previewMode && pct >= 50}
        <Confetti count={pct >= 90 ? 120 : 80} duration={2600} />
    {/if}

    <div
        class="relative flex w-full max-w-3xl mx-auto flex-col gap-6 py-8 motion-rise-in"
    >
        <!-- Ceremony card -->
        <section
            class="certificate-print-card relative overflow-hidden rounded-3xl border border-border brand-gradient px-6 py-10 md:px-12 md:py-14 text-primary-foreground"
        >
            <!-- Higgsfield gold laurel emblem ceremony backdrop -->
            <img
                src="/brand/achievements/ceremony.png"
                alt=""
                aria-hidden="true"
                class="absolute inset-0 size-full object-cover opacity-30 mix-blend-luminosity"
                fetchpriority="high"
            />
            <Spotlight class="h-full w-full" opacity={0.7} />

            <div
                class="relative z-10 flex flex-col items-center text-center gap-6"
            >
                <span
                    class="inline-flex items-center gap-1.5 rounded-full bg-accent/15 px-3 py-1 text-[10px] font-semibold uppercase tracking-[0.18em] text-accent"
                >
                    <Sparkles class="size-3" />
                    Course complete
                </span>

                <BadgeMedal {tier} size={120} />

                <div class="flex flex-col items-center gap-2 max-w-lg">
                    <GiaAvatar state="celebrating" size={56} />
                    <h1
                        class="text-display-lg font-bold tracking-tight text-primary-foreground"
                    >
                        {headline}
                    </h1>
                    <p class="text-sm text-primary-foreground/80">
                        {enrolledCourse.title}
                    </p>
                </div>

                <div class="flex items-baseline gap-3 tabular">
                    <p
                        class="text-display-2xl font-bold leading-none text-accent"
                    >
                        <NumberTicker value={pct} suffix="%" />
                    </p>
                </div>
                <p class="text-xs text-primary-foreground/70 tabular">
                    {score.correct} of {score.total} correct ·
                    {tier === "gold"
                        ? "Gold tier"
                        : tier === "silver"
                          ? "Silver tier"
                          : "Bronze tier"}
                </p>
                <!-- Caption layer — fades in after the badge -->
                {#if pct >= 90}
                    <p
                        class="ceremony-caption text-xs text-accent font-medium tracking-wide motion-rise-in"
                        style="animation-delay: 600ms"
                    >
                        Gold on {enrolledCourse.title}. Audit-ready.
                    </p>
                {:else if pct >= 75}
                    <p
                        class="ceremony-caption text-xs text-primary-foreground/80 font-medium tracking-wide motion-rise-in"
                        style="animation-delay: 600ms"
                    >
                        Silver tier. One more attempt and you're at gold.
                    </p>
                {:else if pct >= 50}
                    <p
                        class="ceremony-caption text-xs text-primary-foreground/80 font-medium tracking-wide motion-rise-in"
                        style="animation-delay: 600ms"
                    >
                        Bronze tier earned. Steady the foundations and retry.
                    </p>
                {/if}
                <p class="text-[11px] text-primary-foreground/60">
                    Your progress has been saved.
                </p>
                <!-- Print-only certificate footer; hidden in screen view -->
                <p
                    class="print-only text-[10px] text-primary-foreground/60 mt-2 tabular"
                >
                    Issued by FGB Academy · {new Date().toLocaleDateString(
                        undefined,
                        { year: "numeric", month: "long", day: "numeric" },
                    )}
                </p>
            </div>
        </section>

        <div class="ceremony-actions flex flex-wrap justify-center gap-3">
            <Button.Root variant="outline" size="lg" onclick={goHome}>
                <ChevronLeft class="size-4" />
                Back to courses
            </Button.Root>
            <Button.Root
                size="lg"
                onclick={() => {
                    showResults = false;
                    currentModuleIdx = enrolledCourse.modules?.length
                        ? enrolledCourse.modules.length - 1
                        : 0;
                }}
            >
                Review answers
                <ArrowRight class="size-4" />
            </Button.Root>
            {#if pct >= 75}
                <Button.Root
                    variant="outline"
                    size="lg"
                    onclick={() => window.print()}
                    aria-label="Print certificate"
                >
                    <Printer class="size-4" />
                    Print certificate
                </Button.Root>
            {/if}
        </div>
    </div>
{:else}
    <div class="flex w-full max-w-6xl mx-auto flex-col gap-4">
        {#if enrolledCourse.blocked}
            <div
                class="rounded-2xl border border-destructive/30 bg-destructive/5 px-6 py-8 flex flex-col items-center gap-4 motion-rise-in"
            >
                <AlertTriangle class="size-10 text-destructive" />
                <div class="text-center">
                    <h2 class="text-lg font-semibold text-foreground mb-2">
                        {enrolledCourse.title}
                    </h2>
                    <p class="text-sm text-destructive font-medium">
                        {enrolledCourse.blocked.message}
                    </p>
                    {#if enrolledCourse.blocked.reason === "expired"}
                        <p class="text-xs text-muted-foreground mt-3 max-w-md">
                            This course has a set deadline. You can no longer
                            access it because the completion window has passed.
                            Contact your administrator if you believe this is an
                            error.
                        </p>
                    {/if}
                    {#if enrolledCourse.progress?.completed}
                        <p class="text-xs text-muted-foreground mt-2">
                            Score: {enrolledCourse.progress.score_pct}%
                        </p>
                    {/if}
                </div>
                <Button.Root variant="outline" size="sm" onclick={goHome}>
                    <ChevronLeft class="size-3.5 mr-1" /> Back to courses
                </Button.Root>
            </div>
        {:else if previewMode}
            <div
                role="status"
                class="w-full rounded-2xl border border-info/30 bg-info/5 px-4 py-2.5 flex items-center justify-between motion-rise-in"
            >
                <div class="flex items-center gap-2">
                    <Eye class="size-4 text-info" />
                    <span class="text-sm font-medium text-info"
                        >Preview mode</span
                    >
                    <span class="text-xs text-info/80"
                        >— this is a course preview, answers are not saved</span
                    >
                </div>
                <Button.Root variant="outline" size="sm" onclick={goHome}>
                    <ChevronLeft class="size-3.5 mr-1" /> Back to editor
                </Button.Root>
            </div>
        {/if}
        {#if isContentCreator && !previewMode && !enrolledCourse.blocked}
            <div class="rounded-xl border border-border bg-card p-4">
                <div class="flex items-center justify-between mb-2">
                    <button
                        class="flex items-center gap-2 text-sm font-medium text-foreground hover:text-primary transition-colors"
                        onclick={() => (settingsEditOpen = !settingsEditOpen)}
                    >
                        <Settings class="size-4" />
                        Course Settings
                        {#if settingsEditOpen}
                            <ChevronUp class="size-3.5" />
                        {:else}
                            <ChevronDown class="size-3.5" />
                        {/if}
                    </button>
                    {#if settingsSaved}
                        <span
                            class="text-xs text-success inline-flex items-center gap-1"
                            ><CheckCircle class="size-3" /> Saved</span
                        >
                    {/if}
                </div>
                {#if settingsEditOpen}
                    <div class="flex flex-col gap-3 mt-2">
                        <div class="flex items-center gap-3 flex-wrap">
                            <label class="flex flex-col gap-1">
                                <span class="text-xs text-muted-foreground"
                                    >Max Attempts</span
                                >
                                <input
                                    type="number"
                                    min="1"
                                    bind:value={editMaxAttempts}
                                    placeholder="Unlimited"
                                    class="rounded-lg border border-input bg-background px-3 py-1.5 text-sm w-28"
                                />
                            </label>
                            <label class="flex flex-col gap-1">
                                <span class="text-xs text-muted-foreground"
                                    >Days to Complete</span
                                >
                                <input
                                    type="number"
                                    min="1"
                                    bind:value={editDaysToComplete}
                                    placeholder="No deadline"
                                    class="rounded-lg border border-input bg-background px-3 py-1.5 text-sm w-28"
                                />
                            </label>
                        </div>
                        <label
                            class="flex items-center gap-2.5 cursor-pointer select-none"
                        >
                            <input
                                type="checkbox"
                                bind:checked={editGraded}
                                class="size-4 rounded border-input"
                            />
                            <span class="text-sm text-foreground">
                                Graded course
                            </span>
                            <span class="text-xs text-muted-foreground">
                                — controls are marked immediately on check;
                                otherwise learners can reveal answers
                                separately.
                            </span>
                        </label>
                        {#if settingsSaveError}
                            <p class="text-xs text-destructive">
                                {settingsSaveError}
                            </p>
                        {/if}
                        <Button.Root
                            size="sm"
                            class="self-start"
                            disabled={settingsSaving}
                            onclick={saveCourseSettings}
                        >
                            {#if settingsSaving}<LoaderCircle
                                    class="size-3.5 mr-1.5 animate-spin"
                                />Saving…{:else}Save Settings{/if}
                        </Button.Root>
                    </div>
                {/if}
            </div>
        {/if}
        <div class="flex flex-col md:flex-row w-full gap-4 md:gap-6">
            <!-- Module Sidebar - collapses to horizontal scroll on mobile -->
            <aside
                class="md:w-[240px] shrink-0 flex flex-col gap-2 md:sticky md:top-20 md:self-start"
            >
                <button
                    class="inline-flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors press w-fit"
                    onclick={goHome}
                >
                    <ChevronLeft class="size-3.5" /> All courses
                </button>
                <div
                    class="rounded-2xl border border-border bg-card p-3 flex flex-col gap-1"
                >
                    <h3
                        class="text-xs font-semibold uppercase tracking-wider text-muted-foreground px-2 pt-1 pb-2"
                    >
                        Modules
                    </h3>
                    <p
                        class="text-sm font-semibold text-foreground px-2 line-clamp-2 leading-snug mb-1"
                    >
                        {enrolledCourse.title}
                    </p>
                    {#each enrolledCourse.modules as mod, mi}
                        {@const mp = moduleProgress(mod)}
                        {@const isLast =
                            mi === enrolledCourse.modules.length - 1}
                        {@const isCurrent = mi === currentModuleIdx}
                        <button
                            class="flex items-center gap-2.5 rounded-lg px-2 py-2 text-left text-sm transition-colors {isCurrent
                                ? 'bg-primary/10 text-foreground font-medium'
                                : 'text-muted-foreground hover:bg-muted hover:text-foreground'}"
                            onclick={() => (currentModuleIdx = mi)}
                        >
                            {#if mp.pct === 100 && isLast}
                                <span class="shrink-0" aria-hidden="true">
                                    <BadgeMedal tier="gold" size={24} />
                                </span>
                            {:else if mp.total > 0}
                                <span class="shrink-0" aria-hidden="true">
                                    <XpRing
                                        value={mp.pct}
                                        size={24}
                                        stroke={3}
                                        ring={mp.pct === 100
                                            ? "success"
                                            : "primary"}
                                    >
                                        {#snippet center()}
                                            {#if mp.pct === 100}
                                                <CheckCircle
                                                    class="size-3 text-success"
                                                />
                                            {:else}
                                                <span
                                                    class="text-[9px] font-bold tabular text-muted-foreground"
                                                >
                                                    {mi + 1}
                                                </span>
                                            {/if}
                                        {/snippet}
                                    </XpRing>
                                </span>
                            {:else}
                                <span
                                    class="size-6 rounded-full border border-border flex items-center justify-center shrink-0 text-[10px] font-bold {isCurrent
                                        ? 'border-primary text-primary'
                                        : 'text-muted-foreground'}"
                                >
                                    {mi + 1}
                                </span>
                            {/if}
                            <span class="truncate text-xs">{mod.title}</span>
                        </button>
                    {/each}
                </div>
            </aside>

            <!-- Main Content -->
            <main class="flex-1 min-w-0">
                {#if currentModule}
                    {@const mp = moduleProgress(currentModule)}
                    <div
                        class="rounded-2xl border border-border bg-card p-6 mb-4 lift"
                    >
                        <div class="flex items-start justify-between gap-4">
                            <div class="min-w-0">
                                <span
                                    class="text-[10px] font-semibold uppercase tracking-[0.18em] text-muted-foreground"
                                >
                                    Module {currentModuleIdx + 1} of {enrolledCourse
                                        .modules.length}
                                </span>
                                <h2
                                    class="text-xl font-semibold text-foreground mt-1 leading-tight"
                                >
                                    {currentModule.title}
                                </h2>
                                {#if currentModule.description}
                                    <p
                                        class="text-sm text-muted-foreground mt-1.5 leading-relaxed"
                                    >
                                        {currentModule.description}
                                    </p>
                                {/if}
                            </div>
                            {#if mp.total > 0}
                                <span
                                    class="shrink-0"
                                    aria-label={`${mp.done} of ${mp.total} questions answered`}
                                >
                                    <XpRing
                                        value={mp.pct}
                                        size={56}
                                        stroke={5}
                                        ring={mp.pct === 100
                                            ? "success"
                                            : "primary"}
                                    >
                                        {#snippet center()}
                                            <span
                                                class="text-[10px] font-bold tabular text-foreground"
                                            >
                                                {mp.done}/{mp.total}
                                            </span>
                                        {/snippet}
                                    </XpRing>
                                </span>
                            {/if}
                        </div>
                        <div class="mt-4 flex items-center gap-2">
                            <div
                                class="h-1.5 flex-1 rounded-full bg-muted overflow-hidden"
                            >
                                <div
                                    class="h-full rounded-full bg-gradient-to-r from-primary to-accent transition-all duration-500"
                                    style="width: {mp.pct}%"
                                ></div>
                            </div>
                            <span
                                class="text-xs text-muted-foreground shrink-0 tabular"
                                >{mp.done}/{mp.total}</span
                            >
                        </div>
                    </div>

                    {#if assessmentMix.length > 0}
                        <div
                            class="mb-4 rounded-2xl border border-border bg-card p-4 lift"
                        >
                            <div class="mb-3 flex items-center justify-between">
                                <h3
                                    class="text-sm font-semibold text-foreground"
                                >
                                    Assessment mix
                                </h3>
                                <span
                                    class="text-[10px] uppercase tracking-wider text-muted-foreground"
                                >
                                    {assessmentMix.reduce(
                                        (sum, item) => sum + item.count,
                                        0,
                                    )} items
                                </span>
                            </div>
                            <div class="grid grid-cols-2 gap-2 md:grid-cols-4">
                                {#each assessmentMix.slice(0, 8) as item}
                                    <div
                                        class="rounded-xl border border-border bg-surface-1 p-3"
                                    >
                                        <div
                                            class="flex items-center justify-between gap-2"
                                        >
                                            <span
                                                class="truncate text-xs font-medium text-foreground"
                                            >
                                                {item.label}
                                            </span>
                                            <span
                                                class="text-xs font-semibold tabular text-primary"
                                            >
                                                {item.count}
                                            </span>
                                        </div>
                                        <div
                                            class="mt-2 h-1.5 overflow-hidden rounded-full bg-muted"
                                        >
                                            <div
                                                class="h-full rounded-full bg-gradient-to-r from-primary to-accent"
                                                style:width="{item.pct}%"
                                            ></div>
                                        </div>
                                    </div>
                                {/each}
                            </div>
                        </div>
                    {/if}

                    <div class="flex flex-col gap-4">
                        {#each currentModule.items as item, ii}
                            {#if item.item_type === "content"}
                                <article
                                    class="rounded-2xl border border-border bg-card p-6 md:p-8 motion-rise-in"
                                    style="animation-delay: {Math.min(
                                        ii * 30,
                                        200,
                                    )}ms"
                                >
                                    <span
                                        class="text-[10px] font-semibold uppercase tracking-[0.18em] text-muted-foreground"
                                    >
                                        Learning material
                                    </span>
                                    <Markdown
                                        source={item.data?.body ?? ""}
                                        class="mt-3 max-w-prose text-[15px]"
                                    />
                                </article>
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
                                                    checked={answers[
                                                        item.id
                                                    ] === oi}
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
                                    {#if !showResults && !checked[item.id]}
                                        <button
                                            class="mt-3 inline-flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-muted-foreground hover:border-primary hover:text-primary transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                                            disabled={answers[item.id] ===
                                                undefined}
                                            onclick={() =>
                                                (checked[item.id] = true)}
                                        >
                                            <CheckCircle class="size-3" />
                                            Check
                                        </button>
                                    {/if}
                                    {#if !showResults && checked[item.id]}
                                        <AnswerFeedback
                                            status={answers[item.id] ===
                                            undefined
                                                ? "unanswered"
                                                : isCorrect(
                                                        item,
                                                        answers[item.id],
                                                    )
                                                  ? "correct"
                                                  : "incorrect"}
                                            correctAnswer={correctAnswerLabel(
                                                item,
                                            )}
                                            revealAnswer={isGraded ||
                                                revealed[item.id]}
                                        >
                                            {#snippet explanation()}
                                                {#if item.data?.explanation && (isGraded || revealed[item.id])}
                                                    {item.data.explanation}
                                                {/if}
                                            {/snippet}
                                        </AnswerFeedback>
                                        {#if !isGraded && !revealed[item.id] && !isCorrect(item, answers[item.id])}
                                            <button
                                                class="mt-2 inline-flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-muted-foreground hover:border-info hover:text-info transition-colors"
                                                onclick={() =>
                                                    (revealed[item.id] = true)}
                                            >
                                                <Eye class="size-3" />
                                                Reveal answer
                                            </button>
                                        {/if}
                                    {/if}
                                    {#if showResults}
                                        <AnswerFeedback
                                            status={answers[item.id] ===
                                            undefined
                                                ? "unanswered"
                                                : isCorrect(
                                                        item,
                                                        answers[item.id],
                                                    )
                                                  ? "correct"
                                                  : "incorrect"}
                                            correctAnswer={correctAnswerLabel(
                                                item,
                                            )}
                                            revealAnswer={true}
                                        >
                                            {#snippet explanation()}
                                                {#if item.data?.explanation}
                                                    {item.data.explanation}
                                                {/if}
                                            {/snippet}
                                        </AnswerFeedback>
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
                                        <span
                                            class="text-xs text-muted-foreground"
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
                                                            answers[item.id] ??
                                                            [];
                                                        const target =
                                                            e.target as HTMLInputElement;
                                                        setAnswer(
                                                            item.id,
                                                            target.checked
                                                                ? [...cur, oi]
                                                                : cur.filter(
                                                                      (
                                                                          v: number,
                                                                      ) =>
                                                                          v !==
                                                                          oi,
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
                                    {#if !showResults && !checked[item.id]}
                                        <button
                                            class="mt-3 inline-flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-muted-foreground hover:border-primary hover:text-primary transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                                            disabled={(answers[item.id] ?? [])
                                                .length === 0}
                                            onclick={() =>
                                                (checked[item.id] = true)}
                                        >
                                            <CheckCircle class="size-3" />
                                            Check
                                        </button>
                                    {/if}
                                    {#if !showResults && checked[item.id]}
                                        <AnswerFeedback
                                            status={answers[item.id] ===
                                            undefined
                                                ? "unanswered"
                                                : isCorrect(
                                                        item,
                                                        answers[item.id],
                                                    )
                                                  ? "correct"
                                                  : "incorrect"}
                                            correctAnswer={correctAnswerLabel(
                                                item,
                                            )}
                                            revealAnswer={isGraded ||
                                                revealed[item.id]}
                                        />
                                        {#if !isGraded && !revealed[item.id] && !isCorrect(item, answers[item.id])}
                                            <button
                                                class="mt-2 inline-flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-muted-foreground hover:border-info hover:text-info transition-colors"
                                                onclick={() =>
                                                    (revealed[item.id] = true)}
                                            >
                                                <Eye class="size-3" />
                                                Reveal answer
                                            </button>
                                        {/if}
                                    {/if}
                                    {#if showResults}
                                        <AnswerFeedback
                                            status={answers[item.id] ===
                                            undefined
                                                ? "unanswered"
                                                : isCorrect(
                                                        item,
                                                        answers[item.id],
                                                    )
                                                  ? "correct"
                                                  : "incorrect"}
                                            correctAnswer={correctAnswerLabel(
                                                item,
                                            )}
                                            revealAnswer={true}
                                        />
                                    {/if}
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
                                    {#if !showResults && !checked[item.id]}
                                        <button
                                            class="mt-3 inline-flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-muted-foreground hover:border-primary hover:text-primary transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                                            disabled={answers[item.id] ===
                                                undefined}
                                            onclick={() =>
                                                (checked[item.id] = true)}
                                        >
                                            <CheckCircle class="size-3" />
                                            Check
                                        </button>
                                    {/if}
                                    {#if !showResults && checked[item.id]}
                                        <AnswerFeedback
                                            status={answers[item.id] ===
                                            undefined
                                                ? "unanswered"
                                                : isCorrect(
                                                        item,
                                                        answers[item.id],
                                                    )
                                                  ? "correct"
                                                  : "incorrect"}
                                            correctAnswer={correctAnswerLabel(
                                                item,
                                            )}
                                            revealAnswer={isGraded ||
                                                revealed[item.id]}
                                        />
                                        {#if !isGraded && !revealed[item.id] && !isCorrect(item, answers[item.id])}
                                            <button
                                                class="mt-2 inline-flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-muted-foreground hover:border-info hover:text-info transition-colors"
                                                onclick={() =>
                                                    (revealed[item.id] = true)}
                                            >
                                                <Eye class="size-3" />
                                                Reveal answer
                                            </button>
                                        {/if}
                                    {/if}
                                    {#if showResults}
                                        <AnswerFeedback
                                            status={answers[item.id] ===
                                            undefined
                                                ? "unanswered"
                                                : isCorrect(
                                                        item,
                                                        answers[item.id],
                                                    )
                                                  ? "correct"
                                                  : "incorrect"}
                                            correctAnswer={correctAnswerLabel(
                                                item,
                                            )}
                                            revealAnswer={true}
                                        />
                                    {/if}
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
                                                value={answers[item.id]?.[bi] ??
                                                    ""}
                                                oninput={(e) => {
                                                    const cur = [
                                                        ...(answers[item.id] ??
                                                            []),
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
                                    {#if !showResults && !checked[item.id]}
                                        <button
                                            class="mt-3 inline-flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-muted-foreground hover:border-primary hover:text-primary transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                                            disabled={!answers[item.id] ||
                                                (answers[item.id] ?? [])
                                                    .length === 0}
                                            onclick={() =>
                                                (checked[item.id] = true)}
                                        >
                                            <CheckCircle class="size-3" />
                                            Check
                                        </button>
                                    {/if}
                                    {#if !showResults && checked[item.id]}
                                        <AnswerFeedback
                                            status={!answers[item.id]
                                                ? "unanswered"
                                                : isCorrect(
                                                        item,
                                                        answers[item.id],
                                                    )
                                                  ? "correct"
                                                  : "incorrect"}
                                            correctAnswer={correctAnswerLabel(
                                                item,
                                            )}
                                            revealAnswer={isGraded ||
                                                revealed[item.id]}
                                        >
                                            {#snippet explanation()}
                                                {#if item.data?.explanation && (isGraded || revealed[item.id])}
                                                    {item.data.explanation}
                                                {/if}
                                            {/snippet}
                                        </AnswerFeedback>
                                        {#if !isGraded && !revealed[item.id] && !isCorrect(item, answers[item.id])}
                                            <button
                                                class="mt-2 inline-flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-muted-foreground hover:border-info hover:text-info transition-colors"
                                                onclick={() =>
                                                    (revealed[item.id] = true)}
                                            >
                                                <Eye class="size-3" />
                                                Reveal answer
                                            </button>
                                        {/if}
                                    {/if}
                                    {#if showResults}
                                        <AnswerFeedback
                                            status={!answers[item.id]
                                                ? "unanswered"
                                                : isCorrect(
                                                        item,
                                                        answers[item.id],
                                                    )
                                                  ? "correct"
                                                  : "incorrect"}
                                            correctAnswer={correctAnswerLabel(
                                                item,
                                            )}
                                            revealAnswer={true}
                                        >
                                            {#snippet explanation()}
                                                {#if item.data?.explanation}
                                                    {item.data.explanation}
                                                {/if}
                                            {/snippet}
                                        </AnswerFeedback>
                                    {/if}
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
                                                (
                                                    e.target as HTMLTextAreaElement
                                                ).value,
                                            )}
                                        placeholder="Type your answer..."
                                        class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring resize-none"
                                    ></textarea>
                                    {#if !showResults && !checked[item.id] && item.data?.sample_answer}
                                        <button
                                            class="mt-2 inline-flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-muted-foreground hover:border-info hover:text-info transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                                            disabled={!answers[item.id]}
                                            onclick={() =>
                                                (checked[item.id] = true)}
                                        >
                                            <Eye class="size-3" />
                                            Reveal sample
                                        </button>
                                    {/if}
                                    {#if showResults || checked[item.id]}
                                        <p
                                            class="mt-2 text-xs text-muted-foreground"
                                        >
                                            Sample answer: {item.data
                                                .sample_answer}
                                        </p>
                                    {/if}
                                </div>
                            {:else if item.item_type === "matching"}
                                <div
                                    class="rounded-xl border border-border bg-card p-5"
                                >
                                    <p
                                        class="mb-3 text-sm font-medium text-foreground"
                                    >
                                        {ii + 1}. {item.data?.question ??
                                            "Match each item with its definition."}
                                    </p>
                                    <QuestionMatching
                                        data={item.data}
                                        value={answers[item.id]}
                                        reveal={showResults ||
                                            (isGraded && checked[item.id]) ||
                                            revealed[item.id]}
                                        onChange={(v) => setAnswer(item.id, v)}
                                    />
                                    {#if !showResults && !checked[item.id]}
                                        <button
                                            class="mt-3 inline-flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-muted-foreground hover:border-primary hover:text-primary transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                                            disabled={answers[item.id] ===
                                                undefined}
                                            onclick={() =>
                                                (checked[item.id] = true)}
                                        >
                                            <CheckCircle class="size-3" />
                                            Check
                                        </button>
                                    {/if}
                                    {#if !showResults && checked[item.id]}
                                        <AnswerFeedback
                                            status={answers[item.id] ===
                                            undefined
                                                ? "unanswered"
                                                : isCorrect(
                                                        item,
                                                        answers[item.id],
                                                    )
                                                  ? "correct"
                                                  : "incorrect"}
                                            correctAnswer={correctAnswerLabel(
                                                item,
                                            )}
                                            revealAnswer={isGraded ||
                                                revealed[item.id]}
                                        >
                                            {#snippet explanation()}
                                                {#if item.data?.explanation && (isGraded || revealed[item.id])}
                                                    {item.data.explanation}
                                                {/if}
                                            {/snippet}
                                        </AnswerFeedback>
                                        {#if !isGraded && !revealed[item.id] && !isCorrect(item, answers[item.id])}
                                            <button
                                                class="mt-2 inline-flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-muted-foreground hover:border-info hover:text-info transition-colors"
                                                onclick={() =>
                                                    (revealed[item.id] = true)}
                                            >
                                                <Eye class="size-3" />
                                                Reveal answer
                                            </button>
                                        {/if}
                                    {/if}
                                    {#if showResults}
                                        <AnswerFeedback
                                            status={answers[item.id] ===
                                            undefined
                                                ? "unanswered"
                                                : isCorrect(
                                                        item,
                                                        answers[item.id],
                                                    )
                                                  ? "correct"
                                                  : "incorrect"}
                                            correctAnswer={correctAnswerLabel(
                                                item,
                                            )}
                                            revealAnswer={true}
                                        >
                                            {#snippet explanation()}
                                                {#if item.data?.explanation}
                                                    {item.data.explanation}
                                                {/if}
                                            {/snippet}
                                        </AnswerFeedback>
                                    {/if}
                                </div>
                            {:else if item.item_type === "drag_sort"}
                                <div
                                    class="rounded-xl border border-border bg-card p-5"
                                >
                                    <p
                                        class="mb-3 text-sm font-medium text-foreground"
                                    >
                                        {ii + 1}. {item.data?.question ??
                                            "Put these items in the correct order."}
                                    </p>
                                    <QuestionOrdering
                                        data={item.data}
                                        value={answers[item.id]}
                                        reveal={showResults ||
                                            (isGraded && checked[item.id]) ||
                                            revealed[item.id]}
                                        onChange={(v) => setAnswer(item.id, v)}
                                    />
                                    {#if !showResults && !checked[item.id]}
                                        <button
                                            class="mt-3 inline-flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-muted-foreground hover:border-primary hover:text-primary transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                                            disabled={answers[item.id] ===
                                                undefined}
                                            onclick={() =>
                                                (checked[item.id] = true)}
                                        >
                                            <CheckCircle class="size-3" />
                                            Check
                                        </button>
                                    {/if}
                                    {#if !showResults && checked[item.id]}
                                        <AnswerFeedback
                                            status={answers[item.id] ===
                                            undefined
                                                ? "unanswered"
                                                : isCorrect(
                                                        item,
                                                        answers[item.id],
                                                    )
                                                  ? "correct"
                                                  : "incorrect"}
                                            correctAnswer={correctAnswerLabel(
                                                item,
                                            )}
                                            revealAnswer={isGraded ||
                                                revealed[item.id]}
                                        >
                                            {#snippet explanation()}
                                                {#if item.data?.explanation && (isGraded || revealed[item.id])}
                                                    {item.data.explanation}
                                                {/if}
                                            {/snippet}
                                        </AnswerFeedback>
                                        {#if !isGraded && !revealed[item.id] && !isCorrect(item, answers[item.id])}
                                            <button
                                                class="mt-2 inline-flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-muted-foreground hover:border-info hover:text-info transition-colors"
                                                onclick={() =>
                                                    (revealed[item.id] = true)}
                                            >
                                                <Eye class="size-3" />
                                                Reveal answer
                                            </button>
                                        {/if}
                                    {/if}
                                    {#if showResults}
                                        <AnswerFeedback
                                            status={answers[item.id] ===
                                            undefined
                                                ? "unanswered"
                                                : isCorrect(
                                                        item,
                                                        answers[item.id],
                                                    )
                                                  ? "correct"
                                                  : "incorrect"}
                                            correctAnswer={correctAnswerLabel(
                                                item,
                                            )}
                                            revealAnswer={true}
                                        >
                                            {#snippet explanation()}
                                                {#if item.data?.explanation}
                                                    {item.data.explanation}
                                                {/if}
                                            {/snippet}
                                        </AnswerFeedback>
                                    {/if}
                                </div>
                            {:else if item.item_type === "hotspot"}
                                <div
                                    class="rounded-xl border border-border bg-card p-5"
                                >
                                    <p
                                        class="mb-3 text-sm font-medium text-foreground"
                                    >
                                        {ii + 1}. {item.data?.question ??
                                            "Select the correct hotspot."}
                                    </p>
                                    <QuestionHotspot
                                        data={item.data}
                                        value={answers[item.id]}
                                        reveal={showResults ||
                                            (isGraded && checked[item.id]) ||
                                            revealed[item.id]}
                                        onChange={(v) => setAnswer(item.id, v)}
                                    />
                                    {#if !showResults && !checked[item.id]}
                                        <button
                                            class="mt-3 inline-flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-muted-foreground hover:border-primary hover:text-primary transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                                            disabled={answers[item.id] ===
                                                undefined}
                                            onclick={() =>
                                                (checked[item.id] = true)}
                                        >
                                            <CheckCircle class="size-3" />
                                            Check
                                        </button>
                                    {/if}
                                    {#if !showResults && checked[item.id]}
                                        <AnswerFeedback
                                            status={answers[item.id] ===
                                            undefined
                                                ? "unanswered"
                                                : isCorrect(
                                                        item,
                                                        answers[item.id],
                                                    )
                                                  ? "correct"
                                                  : "incorrect"}
                                            correctAnswer={correctAnswerLabel(
                                                item,
                                            )}
                                            revealAnswer={isGraded ||
                                                revealed[item.id]}
                                        >
                                            {#snippet explanation()}
                                                {#if item.data?.explanation && (isGraded || revealed[item.id])}
                                                    {item.data.explanation}
                                                {/if}
                                            {/snippet}
                                        </AnswerFeedback>
                                        {#if !isGraded && !revealed[item.id] && !isCorrect(item, answers[item.id])}
                                            <button
                                                class="mt-2 inline-flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-muted-foreground hover:border-info hover:text-info transition-colors"
                                                onclick={() =>
                                                    (revealed[item.id] = true)}
                                            >
                                                <Eye class="size-3" />
                                                Reveal answer
                                            </button>
                                        {/if}
                                    {/if}
                                    {#if showResults}
                                        <AnswerFeedback
                                            status={answers[item.id] ===
                                            undefined
                                                ? "unanswered"
                                                : isCorrect(
                                                        item,
                                                        answers[item.id],
                                                    )
                                                  ? "correct"
                                                  : "incorrect"}
                                            correctAnswer={correctAnswerLabel(
                                                item,
                                            )}
                                            revealAnswer={true}
                                        >
                                            {#snippet explanation()}
                                                {#if item.data?.explanation}
                                                    {item.data.explanation}
                                                {/if}
                                            {/snippet}
                                        </AnswerFeedback>
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
    </div>
{/if}
