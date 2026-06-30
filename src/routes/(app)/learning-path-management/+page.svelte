<script lang="ts">
    import type {
        LearningPath,
        PathCourse,
        PublishedCourse,
    } from "./+page.server";
    import {
        BookOpen,
        Plus,
        Trash2,
        Eye,
        Pencil,
        X,
        Layers,
        ArrowUpDown,
    } from "@lucide/svelte";
    import * as Button from "$lib/components/ui/button";
    import {
        PageHeader,
        StatCard,
        EmptyState,
        PremiumTable,
    } from "$lib/components/brand";
    import { showToast } from "$lib/components/brand/toast.svelte";

    let { data } = $props();

    let paths: LearningPath[] = $state([]);
    let publishedCourses: PublishedCourse[] = $state([]);

    $effect(() => {
        paths = data.paths ?? [];
        publishedCourses = data.publishedCourses ?? [];
    });

    // --- Create state ---
    let showCreate = $state(false);
    let newTitle = $state("");
    let newDescription = $state("");
    let createError = $state("");
    let creating = $state(false);

    // --- Edit state ---
    let editingPath: LearningPath | null = $state(null);
    let editTitle = $state("");
    let editDescription = $state("");
    let editError = $state("");
    let saving = $state(false);

    // --- Manage courses modal ---
    let showManageModal = $state(false);
    let managingPath: LearningPath | null = $state(null);
    let pathCourses: PathCourse[] = $state([]);
    let loadingCourses = $state(false);

    // --- Add course state ---
    let showAddCourse = $state(false);
    let selectedCourseId: number | null = $state(null);
    let requiredCourse = $state(true);
    let addCourseError = $state("");

    // --- Preview state ---
    let previewPath: LearningPath | null = $state(null);
    let previewCourses: PathCourse[] = $state([]);

    // Stats
    let totalPaths = $derived(paths.length);
    let publishedPaths = $derived(
        paths.filter((p) => p.status === "published").length,
    );

    // Helpers
    function getStatusBadge(status: string) {
        switch (status) {
            case "published":
                return {
                    label: "Published",
                    cls: "bg-success/10 text-success",
                };
            case "draft":
                return {
                    label: "Draft",
                    cls: "bg-muted text-muted-foreground",
                };
            case "archived":
                return {
                    label: "Archived",
                    cls: "bg-destructive/10 text-destructive",
                };
            default:
                return { label: status, cls: "bg-muted text-muted-foreground" };
        }
    }

    function availableCourses() {
        const existingIds = new Set(pathCourses.map((pc) => pc.course_id));
        return publishedCourses.filter(
            (c) => !existingIds.has(c.id) && c.status === "published",
        );
    }

    // --- CRUD actions ---
    async function handleCreate() {
        createError = "";
        if (!newTitle.trim()) {
            createError = "Title is required.";
            return;
        }
        creating = true;
        try {
            const res = await fetch("/api/admin/learning-paths", {
                method: "POST",
                credentials: "include",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    title: newTitle.trim(),
                    description: newDescription.trim(),
                }),
            });
            if (!res.ok) {
                const err = await res.json();
                createError = err.error ?? "Failed to create";
                return;
            }
            const created = await res.json();
            paths = [created, ...paths];
            newTitle = "";
            newDescription = "";
            showCreate = false;
            showToast("Learning path created.", {
                title: created.title,
                variant: "success",
            });
        } finally {
            creating = false;
        }
    }

    function startEdit(p: LearningPath) {
        editingPath = p;
        editTitle = p.title;
        editDescription = p.description;
        editError = "";
    }

    async function handleSaveEdit() {
        if (!editingPath) return;
        editError = "";
        saving = true;
        try {
            const res = await fetch(
                `/api/admin/learning-paths/${editingPath.id}`,
                {
                    method: "PUT",
                    credentials: "include",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify({
                        title: editTitle.trim(),
                        description: editDescription.trim(),
                    }),
                },
            );
            if (!res.ok) {
                const err = await res.json();
                editError = err.error ?? "Failed to save";
                return;
            }
            const updated = await res.json();
            paths = paths.map((p) => (p.id === updated.id ? updated : p));
            editingPath = null;
            showToast("Learning path updated.", { variant: "success" });
        } finally {
            saving = false;
        }
    }

    async function handleDelete(p: LearningPath) {
        if (!confirm(`Delete "${p.title}"? This cannot be undone.`)) return;
        const res = await fetch(`/api/admin/learning-paths/${p.id}`, {
            method: "DELETE",
            credentials: "include",
        });
        if (!res.ok) {
            const err = await res.json();
            showToast(err.error ?? "Failed to delete", { variant: "error" });
            return;
        }
        paths = paths.filter((x) => x.id !== p.id);
        showToast("Learning path deleted.", { variant: "success" });
    }

    async function handleStatusChange(p: LearningPath, newStatus: string) {
        const res = await fetch(`/api/admin/learning-paths/${p.id}`, {
            method: "PUT",
            credentials: "include",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ status: newStatus }),
        });
        if (!res.ok) {
            const err = await res.json();
            showToast(err.error ?? "Failed to update status", {
                variant: "error",
            });
            return;
        }
        const updated = await res.json();
        paths = paths.map((x) => (x.id === updated.id ? updated : x));
        showToast(`Status changed to ${newStatus}.`, { variant: "success" });
    }

    // --- Course management ---
    async function openManageModal(p: LearningPath) {
        managingPath = p;
        showManageModal = true;
        loadingCourses = true;
        showAddCourse = false;
        addCourseError = "";
        selectedCourseId = null;
        try {
            const res = await fetch(
                `/api/admin/learning-paths/${p.id}/courses`,
                { credentials: "include" },
            );
            if (res.ok) pathCourses = await res.json();
            else pathCourses = [];
        } finally {
            loadingCourses = false;
        }
    }

    function closeManageModal() {
        showManageModal = false;
        managingPath = null;
        pathCourses = [];
        showAddCourse = false;
    }

    async function handleAddCourse() {
        if (!managingPath || selectedCourseId === null) return;
        const pathId = managingPath.id;
        addCourseError = "";
        const maxOrder =
            pathCourses.length > 0
                ? Math.max(...pathCourses.map((c) => c.sort_order))
                : -1;
        try {
            const res = await fetch(
                `/api/admin/learning-paths/${pathId}/courses`,
                {
                    method: "POST",
                    credentials: "include",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify({
                        course_id: selectedCourseId,
                        sort_order: maxOrder + 1,
                        is_required: requiredCourse,
                    }),
                },
            );
            if (!res.ok) {
                const err = await res.json();
                addCourseError = err.error ?? "Failed to add course";
                return;
            }
            // Refresh course list
            const coursesRes = await fetch(
                `/api/admin/learning-paths/${pathId}/courses`,
                { credentials: "include" },
            );
            if (coursesRes.ok) pathCourses = await coursesRes.json();
            selectedCourseId = null;
            showAddCourse = false;
            showToast("Course added to path.", { variant: "success" });
        } catch (e: any) {
            addCourseError = e.message;
        }
    }

    async function handleRemoveCourse(courseId: number) {
        if (!managingPath) return;
        const pathId = managingPath.id;
        const res = await fetch(
            `/api/admin/learning-paths/${pathId}/courses/${courseId}`,
            {
                method: "DELETE",
                credentials: "include",
            },
        );
        if (!res.ok) {
            showToast("Failed to remove course.", { variant: "error" });
            return;
        }
        pathCourses = pathCourses.filter((c) => c.course_id !== courseId);
        showToast("Course removed from path.", { variant: "success" });
    }

    async function handleReorder(courseId: number, direction: "up" | "down") {
        if (!managingPath) return;
        const pathId = managingPath.id;
        const idx = pathCourses.findIndex((c) => c.course_id === courseId);
        if (idx === -1) return;
        const swapIdx = direction === "up" ? idx - 1 : idx + 1;
        if (swapIdx < 0 || swapIdx >= pathCourses.length) return;

        const a = pathCourses[idx];
        const b = pathCourses[swapIdx];
        const aOrder = b.sort_order;
        const bOrder = a.sort_order;

        // Optimistic update
        pathCourses = pathCourses.map((c) => {
            if (c.course_id === a.course_id)
                return { ...c, sort_order: aOrder };
            if (c.course_id === b.course_id)
                return { ...c, sort_order: bOrder };
            return c;
        });
        pathCourses.sort((x, y) => x.sort_order - y.sort_order);

        // Persist both
        await fetch(`/api/admin/learning-paths/${pathId}/reorder`, {
            method: "PUT",
            credentials: "include",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
                course_id: a.course_id,
                sort_order: aOrder,
                is_required: a.is_required,
            }),
        });
        await fetch(`/api/admin/learning-paths/${pathId}/reorder`, {
            method: "PUT",
            credentials: "include",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
                course_id: b.course_id,
                sort_order: bOrder,
                is_required: b.is_required,
            }),
        });
    }

    async function openPreview(p: LearningPath) {
        previewPath = p;
        const res = await fetch(`/api/admin/learning-paths/${p.id}/courses`, {
            credentials: "include",
        });
        previewCourses = res.ok ? await res.json() : [];
    }
</script>

<svelte:head><title>Learning Paths - FGB Academy</title></svelte:head>

<div class="mx-auto flex w-full max-w-5xl flex-col gap-6">
    <PageHeader
        title="Learning Paths"
        eyebrow="Content"
        description="Create sequenced curricula that guide learners through multiple courses."
    >
        {#snippet icon()}
            <Layers class="size-6 text-primary" />
        {/snippet}
        {#snippet actions()}
            <Button.Root
                variant="default"
                size="sm"
                onclick={() => (showCreate = true)}
            >
                <Plus class="size-4" /> New Path
            </Button.Root>
        {/snippet}
    </PageHeader>

    <div
        class="grid grid-cols-1 gap-4 md:grid-cols-3 motion-rise-in motion-stagger-1"
    >
        <StatCard label="Total Paths" value={totalPaths}>
            {#snippet icon()}<Layers class="size-5 text-primary" />{/snippet}
        </StatCard>
        <StatCard label="Published" value={publishedPaths}>
            {#snippet icon()}<BookOpen class="size-5 text-success" />{/snippet}
        </StatCard>
        <StatCard label="Courses Available" value={publishedCourses.length}>
            {#snippet icon()}<Layers
                    class="size-5 text-accent-foreground"
                />{/snippet}
        </StatCard>
    </div>

    <!-- Create modal -->
    {#if showCreate}
        <div
            class="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
            onclick={() => (showCreate = false)}
        >
            <div
                class="w-full max-w-lg rounded-2xl border border-border bg-card p-6 shadow-xl motion-rise-in"
                onclick={(e) => e.stopPropagation()}
            >
                <h3 class="text-lg font-semibold text-foreground">
                    Create Learning Path
                </h3>
                <div class="mt-4 flex flex-col gap-4">
                    <div>
                        <label class="text-sm font-medium text-foreground"
                            >Title</label
                        >
                        <input
                            type="text"
                            bind:value={newTitle}
                            class="mt-1 w-full rounded-lg border border-border bg-muted px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                            placeholder="e.g. Leadership Fundamentals"
                        />
                    </div>
                    <div>
                        <label class="text-sm font-medium text-foreground"
                            >Description</label
                        >
                        <textarea
                            bind:value={newDescription}
                            rows="3"
                            class="mt-1 w-full rounded-lg border border-border bg-muted px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                            placeholder="Optional description of the learning journey..."
                        ></textarea>
                    </div>
                    {#if createError}<p class="text-sm text-destructive">
                            {createError}
                        </p>{/if}
                    <div class="flex justify-end gap-2">
                        <Button.Root
                            variant="outline"
                            size="sm"
                            onclick={() => (showCreate = false)}
                            >Cancel</Button.Root
                        >
                        <Button.Root
                            variant="default"
                            size="sm"
                            onclick={handleCreate}
                            disabled={creating}
                        >
                            {creating ? "Creating..." : "Create Path"}
                        </Button.Root>
                    </div>
                </div>
            </div>
        </div>
    {/if}

    <!-- Edit modal -->
    {#if editingPath}
        <div
            class="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
            onclick={() => (editingPath = null)}
        >
            <div
                class="w-full max-w-lg rounded-2xl border border-border bg-card p-6 shadow-xl motion-rise-in"
                onclick={(e) => e.stopPropagation()}
            >
                <h3 class="text-lg font-semibold text-foreground">
                    Edit Learning Path
                </h3>
                <div class="mt-4 flex flex-col gap-4">
                    <div>
                        <label class="text-sm font-medium text-foreground"
                            >Title</label
                        >
                        <input
                            type="text"
                            bind:value={editTitle}
                            class="mt-1 w-full rounded-lg border border-border bg-muted px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                        />
                    </div>
                    <div>
                        <label class="text-sm font-medium text-foreground"
                            >Description</label
                        >
                        <textarea
                            bind:value={editDescription}
                            rows="3"
                            class="mt-1 w-full rounded-lg border border-border bg-muted px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                        ></textarea>
                    </div>
                    {#if editError}<p class="text-sm text-destructive">
                            {editError}
                        </p>{/if}
                    <div class="flex justify-end gap-2">
                        <Button.Root
                            variant="outline"
                            size="sm"
                            onclick={() => (editingPath = null)}
                            >Cancel</Button.Root
                        >
                        <Button.Root
                            variant="default"
                            size="sm"
                            onclick={handleSaveEdit}
                            disabled={saving}
                        >
                            {saving ? "Saving..." : "Save"}
                        </Button.Root>
                    </div>
                </div>
            </div>
        </div>
    {/if}

    <!-- Preview modal -->
    {#if previewPath}
        <div
            class="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
            onclick={() => (previewPath = null)}
        >
            <div
                class="w-full max-w-2xl max-h-[80vh] overflow-y-auto rounded-2xl border border-border bg-card p-6 shadow-xl motion-rise-in"
                onclick={(e) => e.stopPropagation()}
            >
                <h3 class="text-lg font-semibold text-foreground">
                    {previewPath.title}
                </h3>
                {#if previewPath.description}<p
                        class="mt-1 text-sm text-muted-foreground"
                    >
                        {previewPath.description}
                    </p>{/if}
                <div class="mt-4">
                    <h4 class="text-sm font-semibold text-foreground">
                        Courses ({previewCourses.length})
                    </h4>
                    {#if previewCourses.length === 0}
                        <p class="mt-2 text-sm text-muted-foreground">
                            No courses added yet.
                        </p>
                    {:else}
                        <ol class="mt-2 space-y-2">
                            {#each previewCourses as pc, i}
                                <li
                                    class="flex items-start gap-3 rounded-lg border border-border bg-muted/50 p-3"
                                >
                                    <span
                                        class="flex size-6 shrink-0 items-center justify-center rounded-full bg-primary text-xs font-bold text-primary-foreground"
                                        >{i + 1}</span
                                    >
                                    <div class="min-w-0">
                                        <p
                                            class="text-sm font-medium text-foreground"
                                        >
                                            {pc.course_title}
                                        </p>
                                        {#if pc.course_description}<p
                                                class="text-xs text-muted-foreground line-clamp-2"
                                            >
                                                {pc.course_description}
                                            </p>{/if}
                                    </div>
                                    {#if !pc.is_required}<span
                                            class="ml-auto shrink-0 text-[10px] text-muted-foreground italic"
                                            >Optional</span
                                        >{/if}
                                </li>
                            {/each}
                        </ol>
                    {/if}
                </div>
                <div class="mt-4 flex justify-end">
                    <Button.Root
                        variant="outline"
                        size="sm"
                        onclick={() => (previewPath = null)}>Close</Button.Root
                    >
                </div>
            </div>
        </div>
    {/if}

    <!-- Paths list -->
    {#if paths.length === 0}
        <EmptyState
            title="No learning paths yet"
            description="Create your first learning path to group courses into a guided curriculum."
        >
            {#snippet icon()}<Layers class="size-12" />{/snippet}
            <Button.Root
                variant="default"
                size="sm"
                onclick={() => (showCreate = true)}
            >
                <Plus class="size-4" /> Create Path
            </Button.Root>
        </EmptyState>
    {:else}
        <div class="space-y-3 motion-rise-in motion-stagger-2">
            {#each paths as p}
                <div
                    class="rounded-2xl border border-border bg-card p-4 shadow-sm"
                >
                    <div class="flex items-start justify-between gap-4">
                        <div class="min-w-0 flex-1">
                            <div class="flex items-center gap-2">
                                <h3
                                    class="text-base font-semibold text-foreground truncate"
                                >
                                    {p.title}
                                </h3>
                                <span
                                    class={`rounded-full px-2 py-0.5 text-[10px] font-semibold uppercase ${getStatusBadge(p.status).cls}`}
                                >
                                    {getStatusBadge(p.status).label}
                                </span>
                            </div>
                            {#if p.description}<p
                                    class="mt-1 text-sm text-muted-foreground line-clamp-2"
                                >
                                    {p.description}
                                </p>{/if}
                        </div>
                        <div class="flex shrink-0 items-center gap-1.5">
                            <Button.Root
                                variant="outline"
                                size="xs"
                                onclick={() => openPreview(p)}
                            >
                                <Eye class="size-3" /> Preview
                            </Button.Root>
                            <Button.Root
                                variant="outline"
                                size="xs"
                                onclick={() => startEdit(p)}
                            >
                                <Pencil class="size-3" /> Edit
                            </Button.Root>
                            <Button.Root
                                variant="outline"
                                size="xs"
                                onclick={() => openManageModal(p)}
                            >
                                <Layers class="size-3" /> Courses
                            </Button.Root>
                            <Button.Root
                                variant="ghost"
                                size="icon-sm"
                                onclick={() => handleDelete(p)}
                                title="Delete"
                            >
                                <Trash2 class="size-4 text-destructive" />
                            </Button.Root>
                        </div>
                    </div>

                    <!-- Status action bar -->
                    <div
                        class="mt-3 flex items-center gap-2 border-t border-border pt-3"
                    >
                        <span class="text-xs text-muted-foreground"
                            >Change status:</span
                        >
                        {#each ["draft", "published", "archived"] as s}
                            {#if s !== p.status}
                                <Button.Root
                                    variant="outline"
                                    size="xs"
                                    onclick={() => handleStatusChange(p, s)}
                                >
                                    {s === "published"
                                        ? "Publish"
                                        : s === "archived"
                                          ? "Archive"
                                          : "Unpublish"}
                                </Button.Root>
                            {/if}
                        {/each}
                    </div>
                </div>
            {/each}
        </div>
    {/if}

    <!-- Manage Courses modal -->
    {#if showManageModal && managingPath}
        <div
            class="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
            onclick={closeManageModal}
        >
            <div
                class="w-full max-w-xl max-h-[85vh] overflow-y-auto rounded-2xl border border-border bg-card p-6 shadow-xl motion-rise-in"
                onclick={(e) => e.stopPropagation()}
            >
                <div class="flex items-center justify-between">
                    <div>
                        <h3 class="text-lg font-semibold text-foreground">
                            {managingPath.title}
                        </h3>
                        <p class="text-sm text-muted-foreground">
                            Manage courses in this path
                        </p>
                    </div>
                    <Button.Root
                        variant="ghost"
                        size="icon-sm"
                        onclick={closeManageModal}
                    >
                        <X class="size-4" />
                    </Button.Root>
                </div>

                <div class="mt-4">
                    <div class="flex items-center justify-between">
                        <h4 class="text-sm font-semibold text-foreground">
                            Courses ({pathCourses.length})
                        </h4>
                        <Button.Root
                            variant="outline"
                            size="xs"
                            onclick={() => (showAddCourse = !showAddCourse)}
                        >
                            <Plus class="size-3" />
                            {showAddCourse ? "Cancel" : "Add Course"}
                        </Button.Root>
                    </div>

                    {#if loadingCourses}
                        <p
                            class="mt-4 text-sm text-muted-foreground text-center py-8"
                        >
                            Loading courses...
                        </p>
                    {:else if pathCourses.length === 0}
                        <p
                            class="mt-4 text-sm text-muted-foreground text-center py-8"
                        >
                            No courses added yet. Click "Add Course" to start
                            building the path.
                        </p>
                    {:else}
                        <div class="mt-3 space-y-1">
                            {#each pathCourses as pc, i}
                                <div
                                    class="flex items-center gap-2 rounded-lg border border-border bg-muted/50 p-2"
                                >
                                    <div
                                        class="flex flex-col items-center gap-0.5"
                                    >
                                        <button
                                            class="flex size-5 items-center justify-center rounded text-muted-foreground hover:bg-muted hover:text-foreground disabled:opacity-30"
                                            disabled={i === 0}
                                            onclick={() =>
                                                handleReorder(
                                                    pc.course_id,
                                                    "up",
                                                )}
                                        >
                                            <ArrowUpDown
                                                class="size-3 rotate-180"
                                            />
                                        </button>
                                        <button
                                            class="flex size-5 items-center justify-center rounded text-muted-foreground hover:bg-muted hover:text-foreground disabled:opacity-30"
                                            disabled={i ===
                                                pathCourses.length - 1}
                                            onclick={() =>
                                                handleReorder(
                                                    pc.course_id,
                                                    "down",
                                                )}
                                        >
                                            <ArrowUpDown class="size-3" />
                                        </button>
                                    </div>
                                    <span
                                        class="flex size-5 shrink-0 items-center justify-center rounded-full bg-primary text-[10px] font-bold text-primary-foreground"
                                        >{i + 1}</span
                                    >
                                    <div class="min-w-0 flex-1">
                                        <p
                                            class="text-sm font-medium text-foreground truncate"
                                        >
                                            {pc.course_title}
                                        </p>
                                    </div>
                                    {#if !pc.is_required}
                                        <span
                                            class="text-[10px] text-muted-foreground italic shrink-0"
                                            >Optional</span
                                        >
                                    {/if}
                                    <Button.Root
                                        variant="ghost"
                                        size="icon-sm"
                                        onclick={() =>
                                            handleRemoveCourse(pc.course_id)}
                                        title="Remove"
                                    >
                                        <X class="size-3 text-destructive" />
                                    </Button.Root>
                                </div>
                            {/each}
                        </div>
                    {/if}

                    <!-- Add course panel inside modal -->
                    {#if showAddCourse}
                        <div
                            class="mt-3 rounded-xl border border-border bg-muted/30 p-4"
                        >
                            <h5 class="text-sm font-semibold text-foreground">
                                Add a course
                            </h5>
                            {#if availableCourses().length === 0}
                                <p class="mt-2 text-sm text-muted-foreground">
                                    All published courses are already in this
                                    path.
                                </p>
                            {:else}
                                <select
                                    bind:value={selectedCourseId}
                                    class="mt-2 w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground"
                                >
                                    <option value={null}
                                        >Select a course...</option
                                    >
                                    {#each availableCourses() as c}
                                        <option value={c.id}>{c.title}</option>
                                    {/each}
                                </select>
                                <label
                                    class="mt-2 flex items-center gap-2 text-sm text-muted-foreground"
                                >
                                    <input
                                        type="checkbox"
                                        bind:checked={requiredCourse}
                                        class="rounded"
                                    />
                                    Required course
                                </label>
                                {#if addCourseError}
                                    <p class="mt-1 text-sm text-destructive">
                                        {addCourseError}
                                    </p>
                                {/if}
                                <div class="mt-3 flex justify-end gap-2">
                                    <Button.Root
                                        variant="outline"
                                        size="xs"
                                        onclick={() => (showAddCourse = false)}
                                    >
                                        Cancel
                                    </Button.Root>
                                    <Button.Root
                                        variant="default"
                                        size="xs"
                                        onclick={handleAddCourse}
                                        disabled={selectedCourseId === null}
                                    >
                                        <Plus class="size-3" /> Add
                                    </Button.Root>
                                </div>
                            {/if}
                        </div>
                    {/if}
                </div>
            </div>
        </div>
    {/if}
</div>
