<script lang="ts">
    import type {
        LearningPath,
        PathCourse,
        PublishedCourse,
        User,
    } from "../+page.server";
    import type { PathEnrollment, Department } from "./+page.server";
    import {
        Layers,
        Plus,
        ArrowLeft,
        Trash2,
        X,
        Users,
        CheckCircle,
        Search,
        ArrowUpDown,
        Save,
        Building2,
    } from "@lucide/svelte";
    import * as Button from "$lib/components/ui/button";
    import * as Tabs from "$lib/components/ui/tabs";
    import { PageHeader, CourseSearchPicker } from "$lib/components/brand";
    import { showToast } from "$lib/components/brand/toast.svelte";
    import { goto } from "$app/navigation";

    let { data } = $props();

    let activeTab = $state("details");

    // --- Path detail state ---
    let path = $state<LearningPath | null>(null);
    let editTitle = $state("");
    let editDescription = $state("");
    let saving = $state(false);

    // --- Courses state ---
    let pathCourses = $state<PathCourse[]>([]);
    let publishedCourses = $state<PublishedCourse[]>([]);
    let selectedCourseId = $state<number | null>(null);
    let requiredCourse = $state(true);
    let addCourseError = $state("");
    let showAddCourse = $state(false);
    let reordering = $state(false);

    // --- Enrollments state ---
    let enrollments = $state<PathEnrollment[]>([]);
    let users = $state<User[]>([]);
    let departments = $state<Department[]>([]);
    let enrollSearch = $state("");
    let selectedUserIds = $state<Set<number>>(new Set());
    let bulkEnrolling = $state(false);
    let selectedDepartmentId = $state<number | null>(null);
    let deptEnrolling = $state(false);

    // Sync server data → local state whenever the loaded path changes.
    $effect(() => {
        const p = data.path;
        path = p;
        editTitle = p?.title ?? "";
        editDescription = p?.description ?? "";
        pathCourses = [...(data.courses ?? [])].sort(
            (a, b) => a.sort_order - b.sort_order,
        );
        publishedCourses = data.publishedCourses ?? [];
        enrollments = data.enrollments ?? [];
        users = data.users ?? [];
        departments = data.departments ?? [];
        selectedUserIds = new Set();
        enrollSearch = "";
        selectedCourseId = null;
        requiredCourse = true;
        showAddCourse = false;
        activeTab = "details";
    });

    let enrolledUserIds = $derived(new Set(enrollments.map((e) => e.user_id)));

    let availableCourses = $derived.by(() => {
        const existingIds = new Set(pathCourses.map((pc) => pc.course_id));
        return publishedCourses.filter(
            (c) => !existingIds.has(c.id) && c.status === "published",
        );
    });

    let availableUsers = $derived.by(() => {
        const q = enrollSearch.trim().toLowerCase();
        return users
            .filter(
                (u) =>
                    !enrolledUserIds.has(u.id) &&
                    (q === "" ||
                        u.name.toLowerCase().includes(q) ||
                        u.email.toLowerCase().includes(q)),
            )
            .slice(0, 50);
    });

    function getStatusBadge(status: string) {
        switch (status) {
            case "published":
                return { label: "Published", cls: "bg-success/10 text-success" };
            case "draft":
                return { label: "Draft", cls: "bg-muted text-muted-foreground" };
            case "archived":
                return {
                    label: "Archived",
                    cls: "bg-destructive/10 text-destructive",
                };
            default:
                return { label: status, cls: "bg-muted text-muted-foreground" };
        }
    }

    function statusLabel(s: string) {
        return s === "published"
            ? "Publish"
            : s === "archived"
              ? "Archive"
              : "Unpublish";
    }

    // --- Details tab ---
    async function handleSaveDetails() {
        if (!path) return;
        saving = true;
        try {
            const res = await fetch(`/api/admin/learning-paths/${path.id}`, {
                method: "PUT",
                credentials: "include",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    title: editTitle.trim(),
                    description: editDescription.trim(),
                }),
            });
            if (!res.ok) {
                const err = await res.json().catch(() => ({}));
                showToast(err.error ?? "Failed to save", { variant: "error" });
                return;
            }
            const updated: LearningPath = await res.json();
            path = updated;
            showToast("Learning path updated.", { variant: "success" });
        } finally {
            saving = false;
        }
    }

    async function handleStatusChange(newStatus: string) {
        if (!path) return;
        const res = await fetch(`/api/admin/learning-paths/${path.id}`, {
            method: "PUT",
            credentials: "include",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ status: newStatus }),
        });
        if (!res.ok) {
            const err = await res.json().catch(() => ({}));
            showToast(err.error ?? "Failed to update status", {
                variant: "error",
            });
            return;
        }
        const updated: LearningPath = await res.json();
        path = updated;
        showToast(`Status changed to ${newStatus}.`, { variant: "success" });
    }

    async function handleDelete() {
        if (!path) return;
        if (!confirm(`Delete "${path.title}"? This cannot be undone.`)) return;
        const res = await fetch(`/api/admin/learning-paths/${path.id}`, {
            method: "DELETE",
            credentials: "include",
        });
        if (!res.ok) {
            const err = await res.json().catch(() => ({}));
            showToast(err.error ?? "Failed to delete", { variant: "error" });
            return;
        }
        showToast("Learning path deleted.", { variant: "success" });
        await goto("/learning-path-management");
    }

    // --- Courses tab ---
    async function refreshCourses() {
        if (!path) return;
        const res = await fetch(`/api/admin/learning-paths/${path.id}/courses`, {
            credentials: "include",
        });
        if (res.ok) {
            const list: PathCourse[] = await res.json();
            pathCourses = list.sort((a, b) => a.sort_order - b.sort_order);
        }
    }

    async function handleAddCourse() {
        if (!path || selectedCourseId === null) return;
        addCourseError = "";
        const maxOrder =
            pathCourses.length > 0
                ? Math.max(...pathCourses.map((c) => c.sort_order))
                : -1;
        try {
            const res = await fetch(`/api/admin/learning-paths/${path.id}/courses`, {
                method: "POST",
                credentials: "include",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    course_id: selectedCourseId,
                    sort_order: maxOrder + 1,
                    is_required: requiredCourse,
                }),
            });
            if (!res.ok) {
                const err = await res.json().catch(() => ({}));
                addCourseError = err.error ?? "Failed to add course";
                return;
            }
            selectedCourseId = null;
            requiredCourse = true;
            showAddCourse = false;
            await refreshCourses();
            showToast("Course added to path.", { variant: "success" });
        } catch (e: any) {
            addCourseError = e?.message ?? "Failed to add course";
        }
    }

    async function handleRemoveCourse(courseId: number) {
        if (!path) return;
        const res = await fetch(
            `/api/admin/learning-paths/${path.id}/courses/${courseId}`,
            { method: "DELETE", credentials: "include" },
        );
        if (!res.ok) {
            showToast("Failed to remove course.", { variant: "error" });
            return;
        }
        await refreshCourses();
        showToast("Course removed from path.", { variant: "success" });
    }

    async function handleReorder(courseId: number, direction: "up" | "down") {
        if (!path || reordering) return;
        const idx = pathCourses.findIndex((c) => c.course_id === courseId);
        if (idx === -1) return;
        const swapIdx = direction === "up" ? idx - 1 : idx + 1;
        if (swapIdx < 0 || swapIdx >= pathCourses.length) return;

        const a = pathCourses[idx];
        const b = pathCourses[swapIdx];
        const aOrder = b.sort_order;
        const bOrder = a.sort_order;

        // Optimistic local reorder
        pathCourses = pathCourses
            .map((c) => {
                if (c.course_id === a.course_id)
                    return { ...c, sort_order: aOrder };
                if (c.course_id === b.course_id)
                    return { ...c, sort_order: bOrder };
                return c;
            })
            .sort((x, y) => x.sort_order - y.sort_order);

        reordering = true;
        try {
            await fetch(`/api/admin/learning-paths/${path.id}/reorder`, {
                method: "PUT",
                credentials: "include",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    course_id: a.course_id,
                    sort_order: aOrder,
                    is_required: a.is_required,
                }),
            });
            await fetch(`/api/admin/learning-paths/${path.id}/reorder`, {
                method: "PUT",
                credentials: "include",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    course_id: b.course_id,
                    sort_order: bOrder,
                    is_required: b.is_required,
                }),
            });
        } finally {
            reordering = false;
        }
    }

    // --- Learners tab ---
    async function refreshEnrollments() {
        if (!path) return;
        const res = await fetch(
            `/api/admin/learning-paths/${path.id}/enrollments`,
            { credentials: "include" },
        );
        enrollments = res.ok ? await res.json() : [];
    }

    function toggleUserSelect(id: number) {
        const next = new Set(selectedUserIds);
        if (next.has(id)) next.delete(id);
        else next.add(id);
        selectedUserIds = next;
    }

    async function handleBulkEnroll() {
        if (!path || selectedUserIds.size === 0) return;
        bulkEnrolling = true;
        try {
            const res = await fetch(
                `/api/admin/learning-paths/${path.id}/enroll/bulk`,
                {
                    method: "POST",
                    credentials: "include",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify({
                        user_ids: Array.from(selectedUserIds),
                    }),
                },
            );
            if (!res.ok) {
                const err = await res.json().catch(() => ({}));
                showToast(err.error || "Failed to enroll learners", {
                    variant: "error",
                });
                return;
            }
            const result = await res.json();
            showToast(
                `Enrolled ${result.enrolled_count}, skipped ${result.skipped_count}`,
                { variant: "success" },
            );
            selectedUserIds = new Set();
            await refreshEnrollments();
        } finally {
            bulkEnrolling = false;
        }
    }

    async function handleEnrollDepartment() {
        if (!path || selectedDepartmentId === null) return;
        deptEnrolling = true;
        try {
            const res = await fetch(
                `/api/admin/learning-paths/${path.id}/enroll/department`,
                {
                    method: "POST",
                    credentials: "include",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify({ department_id: selectedDepartmentId }),
                },
            );
            if (!res.ok) {
                const err = await res.json().catch(() => ({}));
                showToast(err.error || "Failed to enroll department", {
                    variant: "error",
                });
                return;
            }
            const result = await res.json();
            showToast(
                `Enrolled ${result.enrolled_count} of ${result.total_in_dept} from ${result.department}.`,
                { variant: "success" },
            );
            selectedDepartmentId = null;
            await refreshEnrollments();
        } finally {
            deptEnrolling = false;
        }
    }
</script>

<svelte:head
    ><title>{path?.title ?? "Learning Path"} - FGB Academy</title></svelte:head
>

<div class="mx-auto flex w-full max-w-5xl flex-col gap-6">
    <PageHeader
        title={path?.title ?? "Learning Path"}
        eyebrow="Content"
        description={path?.description ?? ""}
    >
        {#snippet icon()}
            <Layers class="size-6 text-primary" />
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

    {#if path}
        <div class="flex flex-wrap items-center gap-2">
            <span
                class={`rounded-full px-2.5 py-0.5 text-[11px] font-semibold uppercase ${getStatusBadge(path.status).cls}`}
            >
                {getStatusBadge(path.status).label}
            </span>
            <span class="text-xs text-muted-foreground">
                Created {new Date(path.created_at).toLocaleDateString()}
            </span>
        </div>

        <Tabs.Root bind:value={activeTab}>
            <Tabs.List>
                <Tabs.Trigger value="details">Details</Tabs.Trigger>
                <Tabs.Trigger value="courses">
                    Courses ({pathCourses.length})
                </Tabs.Trigger>
                <Tabs.Trigger value="learners">
                    <Users class="size-4" /> Learners ({enrollments.length})
                </Tabs.Trigger>
            </Tabs.List>

            <!-- DETAILS -->
            <Tabs.Content value="details">
                <div
                    class="flex flex-col gap-6 rounded-xl border border-border bg-card p-6"
                >
                    <div class="flex flex-col gap-4">
                        <div class="flex flex-col gap-1.5">
                            <label
                                for="d-title"
                                class="text-sm font-medium text-foreground">Title</label
                            >
                            <input
                                id="d-title"
                                type="text"
                                bind:value={editTitle}
                                class="w-full rounded-lg border border-border bg-muted px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                            />
                        </div>
                        <div class="flex flex-col gap-1.5">
                            <label
                                for="d-desc"
                                class="text-sm font-medium text-foreground">Description</label
                            >
                            <textarea
                                id="d-desc"
                                rows="4"
                                bind:value={editDescription}
                                class="w-full rounded-lg border border-border bg-muted px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                            ></textarea>
                        </div>
                        <div>
                            <Button.Root
                                variant="default"
                                size="sm"
                                onclick={handleSaveDetails}
                                disabled={saving}
                            >
                                <Save class="size-4" />
                                {saving ? "Saving..." : "Save"}
                            </Button.Root>
                        </div>
                    </div>

                    <div class="flex flex-col gap-3 border-t border-border pt-4">
                        <h3 class="text-sm font-semibold text-foreground"
                            >Status</h3
                        >
                        <div class="flex flex-wrap items-center gap-2">
                            {#each ["draft", "published", "archived"] as s}
                                <Button.Root
                                    variant={s === path.status
                                        ? "default"
                                        : "outline"}
                                    size="sm"
                                    disabled={s === path.status}
                                    onclick={() => handleStatusChange(s)}
                                >
                                    {statusLabel(s)}
                                </Button.Root>
                            {/each}
                        </div>
                    </div>

                    <div class="border-t border-border pt-4">
                        <h3 class="text-sm font-semibold text-destructive"
                            >Danger zone</h3
                        >
                        <p class="mt-1 text-xs text-muted-foreground">
                            Deleting a path permanently removes it. Enrollments may
                            be affected.
                        </p>
                        <div class="mt-3">
                            <Button.Root
                                variant="destructive"
                                size="sm"
                                onclick={handleDelete}
                            >
                                <Trash2 class="size-4" /> Delete Path
                            </Button.Root>
                        </div>
                    </div>
                </div>
            </Tabs.Content>

            <!-- COURSES -->
            <Tabs.Content value="courses">
                <div class="rounded-xl border border-border bg-card p-6">
                    <div class="flex items-center justify-between">
                        <h3 class="text-sm font-semibold text-foreground"
                            >Courses in this path</h3
                        >
                        <Button.Root
                            variant="outline"
                            size="xs"
                            onclick={() => (showAddCourse = !showAddCourse)}
                        >
                            <Plus class="size-3" />
                            {showAddCourse ? "Cancel" : "Add Course"}
                        </Button.Root>
                    </div>

                    {#if pathCourses.length === 0}
                        <p class="mt-4 text-sm text-muted-foreground">
                            No courses added yet. Click "Add Course" to start
                            building the path.
                        </p>
                    {:else}
                        <div class="mt-3 space-y-1">
                            {#each pathCourses as pc, i (pc.id)}
                                <div
                                    class="flex items-center gap-2 rounded-lg border border-border bg-muted/50 p-2"
                                >
                                    <div
                                        class="flex flex-col items-center gap-0.5"
                                    >
                                        <button
                                            type="button"
                                            class="flex size-5 items-center justify-center rounded text-muted-foreground hover:bg-muted hover:text-foreground disabled:opacity-30"
                                            disabled={i === 0}
                                            onclick={() =>
                                                handleReorder(pc.course_id, "up")}
                                            aria-label="Move up"
                                        >
                                            <ArrowUpDown
                                                class="size-3 rotate-180"
                                            />
                                        </button>
                                        <button
                                            type="button"
                                            class="flex size-5 items-center justify-center rounded text-muted-foreground hover:bg-muted hover:text-foreground disabled:opacity-30"
                                            disabled={i ===
                                                pathCourses.length - 1}
                                            onclick={() =>
                                                handleReorder(
                                                    pc.course_id,
                                                    "down",
                                                )}
                                            aria-label="Move down"
                                        >
                                            <ArrowUpDown class="size-3" />
                                        </button>
                                    </div>
                                    <span
                                        class="flex size-5 shrink-0 items-center justify-center rounded-full bg-primary text-[10px] font-bold text-primary-foreground"
                                    >
                                        {i + 1}
                                    </span>
                                    <div class="min-w-0 flex-1">
                                        <p
                                            class="truncate text-sm font-medium text-foreground"
                                        >
                                            {pc.course_title}
                                        </p>
                                        {#if pc.course_description}
                                            <p
                                                class="truncate text-xs text-muted-foreground"
                                            >
                                                {pc.course_description}
                                            </p>
                                        {/if}
                                    </div>
                                    {#if !pc.is_required}
                                        <span
                                            class="shrink-0 text-[10px] italic text-muted-foreground"
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

                    {#if showAddCourse}
                        <div
                            class="mt-3 rounded-xl border border-border bg-muted/30 p-4"
                        >
                            <h4 class="text-sm font-semibold text-foreground"
                                >Add a course</h4
                            >
                            {#if availableCourses.length === 0}
                                <p class="mt-2 text-sm text-muted-foreground">
                                    All published courses are already in this path.
                                </p>
                            {:else}
                                <div class="mt-2">
                                    <CourseSearchPicker
                                        items={availableCourses}
                                        excludeIds={new Set(
                                            pathCourses.map((pc) => pc.course_id),
                                        )}
                                        placeholder="Search courses to add..."
                                        emptyText="No matching courses."
                                        onselect={(c) =>
                                            (selectedCourseId = c.id)}
                                    />
                                    {#if selectedCourseId !== null}
                                        <p class="mt-2 text-xs text-muted-foreground">
                                            Selected: <span class="font-medium text-foreground">{availableCourses.find((c) => c.id === selectedCourseId)?.title ?? ""}</span>
                                        </p>
                                    {/if}
                                </div>
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
                                        >Cancel</Button.Root
                                    >
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
            </Tabs.Content>

            <!-- LEARNERS -->
            <Tabs.Content value="learners">
                <div
                    class="flex flex-col gap-6 rounded-xl border border-border bg-card p-6"
                >
                    <div>
                        <h3 class="text-sm font-semibold text-foreground">
                            Enrolled learners ({enrollments.length})
                        </h3>
                        {#if enrollments.length === 0}
                            <p class="mt-2 text-sm text-muted-foreground">
                                No learners enrolled yet.
                            </p>
                        {:else}
                            <div class="mt-2 max-h-72 space-y-1 overflow-y-auto">
                                {#each enrollments as e (e.id)}
                                    <div
                                        class="flex items-center justify-between gap-2 rounded-lg border border-border bg-muted/30 px-3 py-2"
                                    >
                                        <div class="min-w-0 flex-1">
                                            <p
                                                class="truncate text-sm font-medium text-foreground"
                                            >
                                                {e.user_name}
                                            </p>
                                            <p
                                                class="truncate text-xs text-muted-foreground"
                                            >
                                                {e.user_email}
                                            </p>
                                        </div>
                                        <div
                                            class="flex shrink-0 items-center gap-2"
                                        >
                                            {#if e.status === "completed"}
                                                <span
                                                    class="inline-flex items-center gap-1 text-xs text-success"
                                                >
                                                    <CheckCircle
                                                        class="size-3"
                                                    /> Completed
                                                </span>
                                            {:else if e.status === "dropped"}
                                                <span class="text-xs text-destructive"
                                                    >Dropped</span
                                                >
                                            {:else}
                                                <span
                                                    class="text-xs text-muted-foreground"
                                                >
                                                    {Math.round(
                                                        parseFloat(
                                                            e.progress_pct,
                                                        ) || 0,
                                                    )}%
                                                </span>
                                            {/if}
                                        </div>
                                    </div>
                                {/each}
                            </div>
                        {/if}
                    </div>

                    <div class="border-t border-border pt-4">
                        <h3 class="text-sm font-semibold text-foreground"
                            >Add learners</h3
                        >

                        <!-- Enroll by department -->
                        {#if departments.length > 0}
                            <div
                                class="mt-3 flex flex-col gap-2 rounded-lg border border-border bg-muted/30 p-3 sm:flex-row sm:items-center"
                            >
                                <div class="flex items-center gap-2 text-sm text-muted-foreground">
                                    <Building2 class="size-4" />
                                    <span>Department:</span>
                                </div>
                                <select
                                    bind:value={selectedDepartmentId}
                                    class="flex-1 rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-foreground"
                                >
                                    <option value={null}>Select a department...</option>
                                    {#each departments as d}
                                        <option value={d.id}>{d.name}</option>
                                    {/each}
                                </select>
                                <Button.Root
                                    variant="default"
                                    size="sm"
                                    onclick={handleEnrollDepartment}
                                    disabled={selectedDepartmentId === null ||
                                        deptEnrolling}
                                >
                                    {deptEnrolling
                                        ? "Enrolling..."
                                        : "Enroll Dept"}
                                </Button.Root>
                            </div>
                            <div class="my-3 flex items-center gap-3">
                                <div class="h-px flex-1 bg-border"></div>
                                <span class="text-[10px] uppercase tracking-wide text-muted-foreground">or pick individuals</span>
                                <div class="h-px flex-1 bg-border"></div>
                            </div>
                        {/if}

                        <div class="relative mt-2">
                            <Search
                                class="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
                            />
                            <input
                                type="text"
                                bind:value={enrollSearch}
                                placeholder="Search by name or email..."
                                class="w-full rounded-lg border border-border bg-muted py-2 pl-9 pr-3 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                            />
                        </div>

                        {#if availableUsers.length === 0}
                            <p class="mt-2 text-sm text-muted-foreground">
                                {users.length === 0
                                    ? "No users available to enroll."
                                    : "No matching users (or everyone is already enrolled)."}
                            </p>
                        {:else}
                            <div class="mt-2 max-h-56 space-y-1 overflow-y-auto">
                                {#each availableUsers as u (u.id)}
                                    <label
                                        class="flex cursor-pointer items-center gap-3 rounded-lg border border-border bg-muted/30 px-3 py-2 hover:bg-muted"
                                    >
                                        <input
                                            type="checkbox"
                                            checked={selectedUserIds.has(u.id)}
                                            onchange={() =>
                                                toggleUserSelect(u.id)}
                                            class="rounded"
                                        />
                                        <div class="min-w-0 flex-1">
                                            <p
                                                class="truncate text-sm font-medium text-foreground"
                                                >{u.name}</p
                                            >
                                            <p
                                                class="truncate text-xs text-muted-foreground"
                                                >{u.email}</p
                                            >
                                        </div>
                                    </label>
                                {/each}
                            </div>

                            <div
                                class="mt-3 flex items-center justify-between gap-2"
                            >
                                <span class="text-xs text-muted-foreground">
                                    {selectedUserIds.size} selected
                                </span>
                                <Button.Root
                                    variant="default"
                                    size="xs"
                                    onclick={handleBulkEnroll}
                                    disabled={selectedUserIds.size === 0 ||
                                        bulkEnrolling}
                                >
                                    {bulkEnrolling
                                        ? "Enrolling..."
                                        : "Enroll Selected"}
                                </Button.Root>
                            </div>
                        {/if}
                    </div>
                </div>
            </Tabs.Content>
        </Tabs.Root>
    {/if}
</div>
