<script lang="ts">
    import type {
        LearningPath,
        PathEnrollment,
        PathDetail,
    } from "./+page.server";
    import {
        BookOpen,
        Plus,
        Play,
        CheckCircle,
        XCircle,
        Layers,
        ChevronRight,
        GraduationCap,
        ArrowRight,
        Clock,
    } from "@lucide/svelte";
    import * as Button from "$lib/components/ui/button";
    import * as Tabs from "$lib/components/ui/tabs";
    import { page } from "$app/stores";
    import { goto } from "$app/navigation";
    import { Pencil, Settings } from "@lucide/svelte";
    import { PageHeader, EmptyState, StatCard } from "$lib/components/brand";
    import { showToast } from "$lib/components/brand/toast.svelte";

    let { data } = $props();

    let user = $derived(($page.data as any)?.user);
    let canManage = $derived(
        user?.roles?.includes("admin") ||
            user?.roles?.includes("manager") ||
            user?.roles?.includes("content creator"),
    );

    let paths: LearningPath[] = $state([]);
    let myEnrollments: PathEnrollment[] = $state([]);

    $effect(() => {
        paths = data.paths ?? [];
        myEnrollments = data.myEnrollments ?? [];
    });

    let activeTab = $state("available");

    // Computed
    let enrolledIds = $derived(
        new Set(myEnrollments.map((e) => e.learning_path_id)),
    );
    let availablePaths = $derived(paths.filter((p) => !enrolledIds.has(p.id)));
    let activeEnrollments = $derived(
        myEnrollments.filter((e) => e.status === "active"),
    );
    let completedEnrollments = $derived(
        myEnrollments.filter((e) => e.status === "completed"),
    );

    // Detail view
    let selectedPath: PathDetail | null = $state(null);
    let loadingDetail = $state(false);
    let enrollingId: number | null = $state(null);

    let inProgressCount = $derived(activeEnrollments.length);
    let completedCount = $derived(completedEnrollments.length);

    function progressPercent(e: PathEnrollment): number {
        const p = parseFloat(e.progress_pct);
        return isNaN(p) ? 0 : p;
    }

    async function loadDetail(pathId: number) {
        loadingDetail = true;
        try {
            const res = await fetch(`/api/learning-paths/${pathId}`, {
                credentials: "include",
            });
            if (res.ok) selectedPath = await res.json();
        } finally {
            loadingDetail = false;
        }
    }

    async function handleEnroll(pathId: number) {
        enrollingId = pathId;
        try {
            const res = await fetch(`/api/learning-paths/${pathId}/enroll`, {
                method: "POST",
                credentials: "include",
            });
            if (!res.ok) {
                const err = await res.json();
                showToast(err.error ?? "Failed to enroll", {
                    variant: "error",
                });
                return;
            }
            const enr = await res.json();
            // Add to local state
            const path = paths.find((p) => p.id === pathId);
            myEnrollments = [
                ...myEnrollments,
                {
                    ...enr,
                    path_title: path?.title ?? "",
                    path_description: path?.description ?? "",
                },
            ];
            showToast("Enrolled in learning path!", { variant: "success" });
        } finally {
            enrollingId = null;
        }
    }

    async function handleDrop(enrollmentId: number) {
        if (!confirm("Drop this learning path? Your progress will be lost."))
            return;
        try {
            const res = await fetch(
                `/api/learning-paths/enrollments/${enrollmentId}/drop`,
                {
                    method: "PUT",
                    credentials: "include",
                },
            );
            if (!res.ok) {
                showToast("Failed to drop enrollment", { variant: "error" });
                return;
            }
            myEnrollments = myEnrollments.filter((e) => e.id !== enrollmentId);
            if (selectedPath?.enrollment?.id === enrollmentId) {
                selectedPath = { ...selectedPath, enrollment: null };
            }
            showToast("Enrollment dropped.", { variant: "success" });
        } catch (e: any) {
            showToast(e.message ?? "Error", { variant: "error" });
        }
    }
</script>

<svelte:head><title>Learning Paths - FGB Academy</title></svelte:head>

<div class="mx-auto flex w-full max-w-5xl flex-col gap-6">
    <PageHeader
        title="Learning Paths"
        eyebrow="Curriculum"
        description="Follow guided sequences of courses designed to build mastery step by step."
    >
        {#snippet icon()}
            <Layers class="size-6 text-primary" />
        {/snippet}
        {#snippet actions()}
            {#if canManage}
                <Button.Root
                    variant="outline"
                    size="sm"
                    onclick={() => goto("/learning-path-management")}
                >
                    <Settings class="size-4" /> Manage
                </Button.Root>
            {/if}
        {/snippet}
    </PageHeader>

    <div
        class="grid grid-cols-1 gap-4 md:grid-cols-2 motion-rise-in motion-stagger-1"
    >
        <StatCard label="In Progress" value={inProgressCount}>
            {#snippet icon()}<Play class="size-5 text-info" />{/snippet}
        </StatCard>
        <StatCard label="Completed" value={completedCount}>
            {#snippet icon()}<CheckCircle
                    class="size-5 text-success"
                />{/snippet}
        </StatCard>
    </div>

    <Tabs.Root bind:value={activeTab}>
        <Tabs.List>
            <Tabs.Trigger value="available">
                <Layers class="size-4" /> Available ({availablePaths.length})
            </Tabs.Trigger>
            <Tabs.Trigger value="my-paths">
                <GraduationCap class="size-4" /> My Paths ({myEnrollments.length})
            </Tabs.Trigger>
        </Tabs.List>

        <!-- Available Paths Tab -->
        <Tabs.Content value="available">
            {#if availablePaths.length === 0}
                <EmptyState
                    title="No paths available"
                    description="All published learning paths are either already enrolled or none exist yet."
                >
                    {#snippet icon()}<Layers class="size-12" />{/snippet}
                </EmptyState>
            {:else}
                <div
                    class="mt-4 grid grid-cols-1 gap-4 md:grid-cols-2 motion-rise-in motion-stagger-2"
                >
                    {#each availablePaths as p}
                        <div
                            class="rounded-2xl border border-border bg-card p-5 shadow-sm lift press"
                        >
                            <div class="flex items-start justify-between gap-3">
                                <div class="min-w-0 flex-1">
                                    <h3
                                        class="text-base font-semibold text-foreground truncate"
                                    >
                                        {p.title}
                                    </h3>
                                    {#if p.description}
                                        <p
                                            class="mt-1 text-sm text-muted-foreground line-clamp-2"
                                        >
                                            {p.description}
                                        </p>
                                    {/if}
                                </div>
                            </div>
                            <div class="mt-4 flex items-center gap-2">
                                <Button.Root
                                    variant="outline"
                                    size="sm"
                                    onclick={() => loadDetail(p.id)}
                                >
                                    <BookOpen class="size-4" /> Preview
                                </Button.Root>
                                <Button.Root
                                    variant="default"
                                    size="sm"
                                    onclick={() => handleEnroll(p.id)}
                                    disabled={enrollingId === p.id}
                                >
                                    {enrollingId === p.id
                                        ? "Enrolling..."
                                        : "Enroll"}
                                    <ArrowRight class="size-4" />
                                </Button.Root>
                            </div>
                        </div>
                    {/each}
                </div>
            {/if}
        </Tabs.Content>

        <!-- My Paths Tab -->
        <Tabs.Content value="my-paths">
            {#if myEnrollments.length === 0}
                <EmptyState
                    title="No learning paths yet"
                    description="Enroll in a learning path from the Available tab to start your guided journey."
                >
                    {#snippet icon()}<GraduationCap class="size-12" />{/snippet}
                </EmptyState>
            {:else}
                <div class="mt-4 space-y-3 motion-rise-in motion-stagger-2">
                    {#each myEnrollments as e}
                        {@const pct = progressPercent(e)}
                        <div
                            class="rounded-2xl border border-border bg-card p-4 shadow-sm"
                        >
                            <div class="flex items-start justify-between gap-3">
                                <div class="min-w-0 flex-1">
                                    <div class="flex items-center gap-2">
                                        <h3
                                            class="text-base font-semibold text-foreground truncate"
                                        >
                                            {e.path_title}
                                        </h3>
                                        {#if e.status === "completed"}
                                            <span
                                                class="rounded-full bg-success/10 px-2 py-0.5 text-[10px] font-semibold uppercase text-success"
                                                >Completed</span
                                            >
                                        {:else if e.status === "dropped"}
                                            <span
                                                class="rounded-full bg-destructive/10 px-2 py-0.5 text-[10px] font-semibold uppercase text-destructive"
                                                >Dropped</span
                                            >
                                        {:else}
                                            <span
                                                class="rounded-full bg-info/10 px-2 py-0.5 text-[10px] font-semibold uppercase text-info"
                                                >Active</span
                                            >
                                        {/if}
                                    </div>
                                    {#if e.path_description}
                                        <p
                                            class="mt-1 text-sm text-muted-foreground line-clamp-1"
                                        >
                                            {e.path_description}
                                        </p>
                                    {/if}

                                    <!-- Progress bar -->
                                    {#if e.status !== "dropped"}
                                        <div class="mt-3">
                                            <div
                                                class="flex items-center justify-between text-xs"
                                            >
                                                <span
                                                    class="text-muted-foreground"
                                                    >Progress</span
                                                >
                                                <span
                                                    class="font-semibold text-foreground"
                                                    >{pct.toFixed(0)}%</span
                                                >
                                            </div>
                                            <div
                                                class="mt-1 h-2 w-full overflow-hidden rounded-full bg-muted"
                                            >
                                                <div
                                                    class="h-full rounded-full transition-all duration-500"
                                                    class:bg-success={pct >=
                                                        100}
                                                    class:bg-primary={pct < 100}
                                                    style="width: {pct}%"
                                                ></div>
                                            </div>
                                        </div>
                                    {/if}
                                </div>

                                <div class="flex shrink-0 items-center gap-1">
                                    <Button.Root
                                        variant="ghost"
                                        size="icon-sm"
                                        onclick={() =>
                                            loadDetail(e.learning_path_id)}
                                        title="View details"
                                    >
                                        <ChevronRight class="size-4" />
                                    </Button.Root>
                                    {#if e.status === "active"}
                                        <Button.Root
                                            variant="ghost"
                                            size="icon-sm"
                                            onclick={() => handleDrop(e.id)}
                                            title="Drop"
                                        >
                                            <XCircle
                                                class="size-4 text-destructive"
                                            />
                                        </Button.Root>
                                    {/if}
                                </div>
                            </div>
                        </div>
                    {/each}
                </div>
            {/if}
        </Tabs.Content>
    </Tabs.Root>

    <!-- Detail modal -->
    {#if selectedPath}
        <div
            class="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
            onclick={() => (selectedPath = null)}
        >
            <div
                class="w-full max-w-2xl max-h-[80vh] overflow-y-auto rounded-2xl border border-border bg-card p-6 shadow-xl motion-rise-in"
                onclick={(e) => e.stopPropagation()}
            >
                <h3 class="text-lg font-semibold text-foreground">
                    {selectedPath.path.title}
                </h3>
                {#if selectedPath.path.description}
                    <p class="mt-1 text-sm text-muted-foreground">
                        {selectedPath.path.description}
                    </p>
                {/if}

                {#if selectedPath.enrollment}
                    {@const epct = progressPercent(selectedPath.enrollment)}
                    <div
                        class="mt-3 flex items-center gap-3 rounded-xl border border-border bg-muted/30 p-3"
                    >
                        <div class="flex-1">
                            <div
                                class="flex items-center justify-between text-xs"
                            >
                                <span class="text-muted-foreground"
                                    >Your progress</span
                                >
                                <span class="font-semibold"
                                    >{epct.toFixed(0)}%</span
                                >
                            </div>
                            <div
                                class="mt-1 h-2 w-full overflow-hidden rounded-full bg-muted"
                            >
                                <div
                                    class="h-full rounded-full bg-primary transition-all"
                                    style="width: {epct}%"
                                ></div>
                            </div>
                        </div>
                        {#if selectedPath.enrollment.status === "active"}
                            <Button.Root
                                variant="outline"
                                size="xs"
                                onclick={() =>
                                    handleDrop(selectedPath!.enrollment!.id)}
                            >
                                <XCircle class="size-3" /> Drop
                            </Button.Root>
                        {/if}
                    </div>
                {:else}
                    <div class="mt-3">
                        <Button.Root
                            variant="default"
                            size="sm"
                            onclick={() => handleEnroll(selectedPath!.path.id)}
                            disabled={enrollingId === selectedPath!.path.id}
                        >
                            {enrollingId === selectedPath!.path.id
                                ? "Enrolling..."
                                : "Enroll in this path"}
                            <ArrowRight class="size-4" />
                        </Button.Root>
                    </div>
                {/if}

                <div class="mt-4">
                    <h4 class="text-sm font-semibold text-foreground">
                        Courses ({selectedPath.courses.length})
                    </h4>
                    {#if selectedPath.courses.length === 0}
                        <p class="mt-2 text-sm text-muted-foreground">
                            No courses in this path yet.
                        </p>
                    {:else}
                        <ol class="mt-2 space-y-2">
                            {#each selectedPath.courses as pc, i}
                                <li
                                    class="flex items-start gap-3 rounded-lg border border-border bg-muted/50 p-3"
                                >
                                    <span
                                        class="flex size-6 shrink-0 items-center justify-center rounded-full bg-primary text-xs font-bold text-primary-foreground"
                                        >{i + 1}</span
                                    >
                                    <div class="min-w-0 flex-1">
                                        <p
                                            class="text-sm font-medium text-foreground"
                                        >
                                            {pc.course_title}
                                        </p>
                                        {#if pc.course_description}
                                            <p
                                                class="text-xs text-muted-foreground line-clamp-2"
                                            >
                                                {pc.course_description}
                                            </p>
                                        {/if}
                                    </div>
                                    {#if !pc.is_required}
                                        <span
                                            class="ml-auto shrink-0 text-[10px] text-muted-foreground italic"
                                            >Optional</span
                                        >
                                    {/if}
                                </li>
                            {/each}
                        </ol>
                    {/if}
                </div>

                <div class="mt-4 flex justify-end">
                    <Button.Root
                        variant="outline"
                        size="sm"
                        onclick={() => (selectedPath = null)}>Close</Button.Root
                    >
                </div>
            </div>
        </div>
    {/if}
</div>
