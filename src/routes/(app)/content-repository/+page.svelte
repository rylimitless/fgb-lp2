<script lang="ts">
    import {
        Search,
        FileText,
        BookOpen,
        Trash2,
        CheckCircle,
        XCircle,
        LoaderCircle,
        Clock,
        ChevronLeft,
        ChevronRight,
        Library,
        MessageSquareText,
        RotateCcw,
        AlertTriangle,
        Users,
        Building2,
        GraduationCap,
    } from "@lucide/svelte";
    import { PageHeader, showToast } from "$lib/components/brand";
    import * as Button from "$lib/components/ui/button";

    type ContentType = "all" | "document" | "course";

    let items = $state<any[]>([]);
    let total = $state(0);
    let totalDocs = $state(0);
    let totalCourses = $state(0);
    let page = $state(0);
    let loading = $state(true);
    let searchQuery = $state("");
    let typeFilter = $state<ContentType>("all");
    let deletingId = $state<number | null>(null);
    let resubmittingId = $state<number | null>(null);
    const pageSize = 20;

    // Notes viewer
    let notesItem = $state<any>(null);

    let searchTimeout: ReturnType<typeof setTimeout>;

    async function loadContent() {
        loading = true;
        try {
            const params = new URLSearchParams({
                limit: String(pageSize),
                offset: String(page * pageSize),
            });
            if (searchQuery.trim()) params.set("q", searchQuery.trim());
            if (typeFilter !== "all") params.set("type", typeFilter);

            const res = await fetch(
                `/api/content-repository?${params.toString()}`,
                { credentials: "include" },
            );
            if (res.ok) {
                const data = await res.json();
                items = data.items ?? [];
                total = data.total ?? 0;
                totalDocs = data.total_docs ?? 0;
                totalCourses = data.total_courses ?? 0;
            }
        } catch {
            /* ignore */
        }
        loading = false;
    }

    function onSearchInput() {
        clearTimeout(searchTimeout);
        searchTimeout = setTimeout(() => {
            page = 0;
            loadContent();
        }, 300);
    }

    function setTypeFilter(t: ContentType) {
        typeFilter = t;
        page = 0;
        loadContent();
    }

    function goPage(p: number) {
        page = p;
        loadContent();
    }

    async function handleDelete(item: any) {
        if (deletingId !== null) return;
        deletingId = item.id;
        try {
            const endpoint =
                item.content_type === "document"
                    ? `/api/content-repository/documents/${item.id}`
                    : `/api/content-repository/courses/${item.id}`;
            const res = await fetch(endpoint, {
                method: "DELETE",
                credentials: "include",
            });
            if (res.ok) {
                items = items.filter(
                    (i: any) =>
                        i.id !== item.id ||
                        i.content_type !== item.content_type,
                );
                total--;
                if (item.content_type === "document") totalDocs--;
                else totalCourses--;
            }
        } catch {
            /* ignore */
        }
        deletingId = null;
    }

    async function handleResubmit(item: any) {
        if (resubmittingId !== null) return;
        resubmittingId = item.id;
        try {
            const endpoint =
                item.content_type === "document"
                    ? `/api/content-repository/documents/${item.id}/resubmit`
                    : `/api/content-repository/courses/${item.id}/resubmit`;
            const res = await fetch(endpoint, {
                method: "PUT",
                credentials: "include",
            });
            if (res.ok) {
                // Refresh to show updated status
                loadContent();
            }
        } catch {
            /* ignore */
        }
        resubmittingId = null;
    }

    function viewNotes(item: any) {
        notesItem = item;
    }

    function closeNotes() {
        notesItem = null;
    }

    function statusIcon(item: any) {
        if (item.content_type === "document") {
            switch (item.status) {
                case "uploaded":
                    return Clock;
                case "processing":
                    return LoaderCircle;
                case "ready":
                    return CheckCircle;
                case "failed":
                    return XCircle;
                default:
                    return FileText;
            }
        } else {
            switch (item.status) {
                case "published":
                    return CheckCircle;
                case "draft":
                    return Clock;
                case "archived":
                    return XCircle;
                default:
                    return BookOpen;
            }
        }
    }

    function statusColor(item: any) {
        if (item.content_type === "document") {
            switch (item.status) {
                case "uploaded":
                    return "text-warning";
                case "processing":
                    return "text-info";
                case "ready":
                    return "text-success";
                case "failed":
                    return "text-destructive";
                default:
                    return "text-muted-foreground";
            }
        } else {
            switch (item.status) {
                case "published":
                    return "text-success";
                case "draft":
                    return "text-warning";
                case "archived":
                    return "text-muted-foreground";
                default:
                    return "text-muted-foreground";
            }
        }
    }

    function statusLabel(item: any) {
        if (item.content_type === "document") {
            switch (item.status) {
                case "uploaded":
                    return "Queued";
                case "processing":
                    return "Processing";
                case "ready":
                    return "Ready";
                case "failed":
                    return "Failed";
                default:
                    return item.status;
            }
        } else {
            switch (item.status) {
                case "published":
                    return "Published";
                case "draft":
                    return "Draft";
                case "archived":
                    return "Archived";
                default:
                    return item.status;
            }
        }
    }

    function reviewBadge(item: any) {
        if (!item.review_status) return null;
        switch (item.review_status) {
            case "approved":
                return {
                    label: "Approved",
                    cls: "bg-success/10 text-success border border-success/20",
                };
            case "rejected":
                return {
                    label: "Rejected",
                    cls: "bg-destructive/10 text-destructive border border-destructive/20",
                };
            case "pending":
                return {
                    label: "Pending",
                    cls: "bg-warning/10 text-warning border border-warning/20",
                };
            case "changes_requested":
                return {
                    label: "Changes Requested",
                    cls: "bg-info/10 text-info border border-info/20",
                };
            default:
                return null;
        }
    }

    function contentTypeLabel(ct: string) {
        return ct === "document" ? "Document" : "Course";
    }

    function formatDate(d: string) {
        if (!d) return "";
        return new Date(d).toLocaleDateString("en-US", {
            year: "numeric",
            month: "short",
            day: "numeric",
            hour: "2-digit",
            minute: "2-digit",
        });
    }

    let totalPages = $derived(Math.max(1, Math.ceil(total / pageSize)));

    // ---- Enroll Department in Course ----
    let enrollCourse = $state<any>(null);
    let enrollDialogOpen = $state(false);
    let departments = $state<any[]>([]);
    let deptLoading = $state(false);
    let selectedDeptId = $state<number | null>(null);
    let enrollError = $state("");
    let enrollSaving = $state(false);

    async function openEnroll(item: any) {
        enrollCourse = item;
        enrollDialogOpen = true;
        selectedDeptId = null;
        enrollError = "";
        departments = [];
        deptLoading = true;
        try {
            const res = await fetch("/api/admin/departments", {
                credentials: "include",
            });
            if (res.ok) {
                const deps = await res.json();
                // Load user count per department
                const withCounts = await Promise.all(
                    deps.map(async (d: any) => {
                        const ur = await fetch(
                            `/api/admin/departments/${d.id}/users`,
                            { credentials: "include" },
                        );
                        const users = ur.ok ? await ur.json() : [];
                        return { ...d, user_count: users.length };
                    }),
                );
                departments = withCounts;
            }
        } catch {
            /* ignore */
        }
        deptLoading = false;
    }

    function closeEnroll() {
        enrollCourse = null;
        enrollDialogOpen = false;
        selectedDeptId = null;
        enrollError = "";
    }

    async function handleBulkEnroll() {
        if (!enrollCourse || !selectedDeptId) return;
        enrollSaving = true;
        enrollError = "";
        try {
            const res = await fetch("/api/admin/departments/bulk-enroll", {
                method: "POST",
                credentials: "include",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    department_id: selectedDeptId,
                    course_id: enrollCourse.id,
                }),
            });
            if (!res.ok) {
                const err = await res.json();
                enrollError = err.error ?? "Enrollment failed";
                return;
            }
            const result = await res.json();
            showToast(
                `${result.enrolled_count} user(s) enrolled in "${result.course_title}".`,
                { title: "Bulk enrollment", variant: "success" },
            );
            closeEnroll();
        } catch {
            enrollError = "Network error";
        } finally {
            enrollSaving = false;
        }
    }

    $effect(() => {
        loadContent();
    });
</script>

<div class="flex w-full max-w-6xl mx-auto flex-col gap-6">
    <PageHeader
        title="Content repository"
        eyebrow="Library"
        description="Every document and course in one searchable index. Filter by type, status, and review state."
    >
        {#snippet icon()}
            <Library class="size-6 text-primary" />
        {/snippet}
    </PageHeader>

    <div class="grid grid-cols-3 gap-4">
        <button
            onclick={() => setTypeFilter("all")}
            class="rounded-xl border {typeFilter === 'all'
                ? 'border-primary bg-primary/5'
                : 'border-border'} bg-card p-4 text-left transition-colors hover:bg-muted/50"
        >
            <p class="text-xs text-muted-foreground mb-1">All Content</p>
            <p class="text-2xl font-semibold text-foreground">
                {totalDocs + totalCourses}
            </p>
        </button>
        <button
            onclick={() => setTypeFilter("document")}
            class="rounded-xl border {typeFilter === 'document'
                ? 'border-primary bg-primary/5'
                : 'border-border'} bg-card p-4 text-left transition-colors hover:bg-muted/50"
        >
            <div class="flex items-center gap-2 mb-1">
                <FileText class="size-3.5 text-muted-foreground" />
                <p class="text-xs text-muted-foreground">Documents</p>
            </div>
            <p class="text-2xl font-semibold text-foreground">{totalDocs}</p>
        </button>
        <button
            onclick={() => setTypeFilter("course")}
            class="rounded-xl border {typeFilter === 'course'
                ? 'border-primary bg-primary/5'
                : 'border-border'} bg-card p-4 text-left transition-colors hover:bg-muted/50"
        >
            <div class="flex items-center gap-2 mb-1">
                <BookOpen class="size-3.5 text-muted-foreground" />
                <p class="text-xs text-muted-foreground">Courses</p>
            </div>
            <p class="text-2xl font-semibold text-foreground">{totalCourses}</p>
        </button>
    </div>

    <div class="flex items-center gap-3">
        <div class="relative flex-1 max-w-md">
            <Search
                class="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground pointer-events-none"
            />
            <input
                type="text"
                placeholder="Search by title..."
                bind:value={searchQuery}
                oninput={onSearchInput}
                class="w-full h-9 pl-9 pr-3 rounded-lg border border-border bg-card text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary transition-colors"
            />
        </div>
        <div class="flex gap-1 rounded-lg bg-muted p-1">
            <button
                onclick={() => setTypeFilter("all")}
                class="px-3 py-1.5 rounded-md text-xs font-medium transition-colors {typeFilter ===
                'all'
                    ? 'bg-background text-foreground shadow-sm'
                    : 'text-muted-foreground hover:text-foreground'}"
            >
                All
            </button>
            <button
                onclick={() => setTypeFilter("document")}
                class="px-3 py-1.5 rounded-md text-xs font-medium transition-colors {typeFilter ===
                'document'
                    ? 'bg-background text-foreground shadow-sm'
                    : 'text-muted-foreground hover:text-foreground'}"
            >
                Documents
            </button>
            <button
                onclick={() => setTypeFilter("course")}
                class="px-3 py-1.5 rounded-md text-xs font-medium transition-colors {typeFilter ===
                'course'
                    ? 'bg-background text-foreground shadow-sm'
                    : 'text-muted-foreground hover:text-foreground'}"
            >
                Courses
            </button>
        </div>
    </div>

    <div class="rounded-xl border border-border bg-card overflow-x-auto">
        {#if loading}
            <div class="flex items-center justify-center py-16">
                <LoaderCircle
                    class="size-6 text-muted-foreground animate-spin"
                />
            </div>
        {:else if items.length === 0}
            <div
                class="flex flex-col items-center justify-center py-16 gap-3 text-muted-foreground"
            >
                <Library class="size-10" />
                <p class="text-sm">No content found.</p>
                {#if searchQuery}
                    <p class="text-xs">Try adjusting your search or filters.</p>
                {/if}
            </div>
        {:else}
            <table class="w-full table-auto">
                <thead>
                    <tr class="border-b border-border bg-muted/50">
                        <th
                            class="text-left px-5 py-3 text-xs font-semibold text-muted-foreground uppercase tracking-wider"
                            >Title</th
                        >
                        <th
                            class="text-left px-5 py-3 text-xs font-semibold text-muted-foreground uppercase tracking-wider w-20"
                            >Type</th
                        >
                        <th
                            class="text-left px-5 py-3 text-xs font-semibold text-muted-foreground uppercase tracking-wider w-28"
                            >Status</th
                        >
                        <th
                            class="text-left px-5 py-3 text-xs font-semibold text-muted-foreground uppercase tracking-wider hidden sm:table-cell w-28"
                            >Review</th
                        >
                        <th
                            class="text-left px-5 py-3 text-xs font-semibold text-muted-foreground uppercase tracking-wider hidden sm:table-cell w-24"
                            >Notes</th
                        >
                        <th
                            class="text-left px-5 py-3 text-xs font-semibold text-muted-foreground uppercase tracking-wider hidden md:table-cell w-40"
                            >Date</th
                        >
                        <th
                            class="text-center px-2 py-3 text-xs font-semibold text-muted-foreground uppercase tracking-wider w-24"
                            >Actions</th
                        >
                    </tr>
                </thead>
                <tbody class="divide-y divide-border">
                    {#each items as item (item.content_type + "-" + item.id)}
                        {@const Icon = statusIcon(item)}
                        {@const badge = reviewBadge(item)}
                        {@const needsAction =
                            item.review_status === "changes_requested" ||
                            item.review_status === "rejected"}
                        <tr class="hover:bg-muted/30 transition-colors">
                            <td class="px-5 py-3">
                                <div class="flex items-center gap-2.5 min-w-0">
                                    {#if item.content_type === "document"}
                                        <FileText
                                            class="size-4 text-muted-foreground shrink-0"
                                        />
                                    {:else}
                                        <BookOpen
                                            class="size-4 text-muted-foreground shrink-0"
                                        />
                                    {/if}
                                    <span
                                        class="text-sm font-medium text-foreground truncate"
                                        >{item.title}</span
                                    >
                                </div>
                            </td>
                            <td class="px-5 py-3">
                                <span
                                    class="text-xs text-muted-foreground whitespace-nowrap"
                                    >{contentTypeLabel(item.content_type)}</span
                                >
                            </td>
                            <td class="px-5 py-3">
                                <div class="flex items-center gap-1.5">
                                    <Icon
                                        class="size-3.5 shrink-0 {statusColor(
                                            item,
                                        )}"
                                    />
                                    <span
                                        class="text-xs text-muted-foreground whitespace-nowrap"
                                        >{statusLabel(item)}</span
                                    >
                                </div>
                            </td>
                            <td class="px-5 py-3 hidden sm:table-cell">
                                {#if badge}
                                    <button
                                        onclick={() => viewNotes(item)}
                                        class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium {badge.cls} hover:opacity-80 transition-opacity cursor-pointer"
                                    >
                                        {badge.label}
                                    </button>
                                {:else}
                                    <span class="text-xs text-muted-foreground"
                                        >—</span
                                    >
                                {/if}
                            </td>
                            <td class="px-5 py-3 hidden sm:table-cell">
                                {#if item.review_notes}
                                    <button
                                        onclick={() => viewNotes(item)}
                                        class="inline-flex items-center gap-1 text-xs text-info hover:underline"
                                    >
                                        <MessageSquareText class="size-3.5" />
                                        View
                                    </button>
                                {:else}
                                    <span class="text-xs text-muted-foreground"
                                        >—</span
                                    >
                                {/if}
                            </td>
                            <td class="px-5 py-3 hidden md:table-cell">
                                <span
                                    class="text-xs text-muted-foreground whitespace-nowrap"
                                    >{formatDate(item.created_at)}</span
                                >
                            </td>
                            <td class="px-1 py-3 text-center">
                                <div
                                    class="flex items-center justify-center gap-0.5"
                                >
                                    {#if needsAction}
                                        <Button.Root
                                            variant="ghost"
                                            size="icon-sm"
                                            onclick={() => handleResubmit(item)}
                                            disabled={resubmittingId ===
                                                item.id}
                                            class="text-info hover:text-info/80"
                                            title="Resubmit for review"
                                        >
                                            {#if resubmittingId === item.id}
                                                <LoaderCircle
                                                    class="size-4 animate-spin"
                                                />
                                            {:else}
                                                <RotateCcw class="size-4" />
                                            {/if}
                                        </Button.Root>
                                    {/if}
                                    {#if item.content_type === "course" && item.status === "published"}
                                        <Button.Root
                                            variant="ghost"
                                            size="icon-sm"
                                            onclick={() => openEnroll(item)}
                                            class="text-muted-foreground hover:text-accent"
                                            title="Enroll a department"
                                        >
                                            <Users class="size-4" />
                                        </Button.Root>
                                    {/if}
                                    <Button.Root
                                        variant="ghost"
                                        size="icon-sm"
                                        onclick={() => handleDelete(item)}
                                        disabled={deletingId === item.id}
                                        class="text-muted-foreground hover:text-destructive"
                                    >
                                        {#if deletingId === item.id}
                                            <LoaderCircle
                                                class="size-4 animate-spin"
                                            />
                                        {:else}
                                            <Trash2 class="size-4" />
                                        {/if}
                                    </Button.Root>
                                </div>
                            </td>
                        </tr>
                    {/each}
                </tbody>
            </table>
        {/if}

        {#if !loading && items.length > 0}
            <div
                class="flex items-center justify-between px-5 py-3 border-t border-border bg-muted/30"
            >
                <span class="text-xs text-muted-foreground">
                    Showing {page * pageSize + 1}–{Math.min(
                        (page + 1) * pageSize,
                        total,
                    )} of {total}
                </span>
                <div class="flex gap-1">
                    <Button.Root
                        variant="outline"
                        size="sm"
                        disabled={page === 0}
                        onclick={() => goPage(page - 1)}
                    >
                        <ChevronLeft class="size-4" />
                        Prev
                    </Button.Root>
                    <span
                        class="flex items-center px-3 text-xs text-muted-foreground"
                    >
                        Page {page + 1} of {totalPages}
                    </span>
                    <Button.Root
                        variant="outline"
                        size="sm"
                        disabled={page >= totalPages - 1}
                        onclick={() => goPage(page + 1)}
                    >
                        Next
                        <ChevronRight class="size-4" />
                    </Button.Root>
                </div>
            </div>
        {/if}
    </div>
</div>

{#if notesItem}
    <div
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
        onclick={closeNotes}
    >
        <div
            class="bg-card border border-border rounded-xl shadow-xl w-full max-w-lg max-h-[80vh] flex flex-col"
            onclick={(e: MouseEvent) => e.stopPropagation()}
        >
            <div
                class="flex items-center justify-between px-6 py-4 border-b border-border"
            >
                <div class="min-w-0">
                    <h3
                        class="text-base font-semibold text-foreground truncate"
                    >
                        Review Notes
                    </h3>
                    <p class="text-xs text-muted-foreground mt-0.5 truncate">
                        {notesItem.title}
                    </p>
                </div>
                <Button.Root
                    variant="ghost"
                    size="icon-sm"
                    onclick={closeNotes}
                    class="shrink-0 ml-3"
                >
                    <XCircle class="size-5" />
                </Button.Root>
            </div>
            <div class="overflow-y-auto px-6 py-4 flex-1">
                {#if !notesItem.review_notes}
                    <p class="text-sm text-muted-foreground text-center py-8">
                        No review notes yet.
                    </p>
                {:else}
                    <div class="flex flex-col gap-3">
                        {#each notesItem.review_notes.split("\n---\n") as entry}
                            <div
                                class="rounded-lg border border-border bg-muted/30 px-4 py-3"
                            >
                                <p
                                    class="text-sm text-foreground/85 leading-relaxed whitespace-pre-wrap"
                                >
                                    {entry}
                                </p>
                            </div>
                        {/each}
                    </div>
                {/if}
            </div>
        </div>
    </div>
{/if}

<!-- Enroll Department Dialog -->
{#if enrollCourse && enrollDialogOpen}
    <div
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
        onclick={closeEnroll}
        role="dialog"
        aria-modal="true"
    >
        <div
            class="bg-card border border-border rounded-xl shadow-xl w-full max-w-md mx-4 max-h-[80vh] flex flex-col"
            onclick={(e: MouseEvent) => e.stopPropagation()}
        >
            <div
                class="flex items-center justify-between px-6 py-4 border-b border-border shrink-0"
            >
                <div class="min-w-0">
                    <h3 class="text-base font-semibold text-foreground">
                        Enroll Department
                    </h3>
                    <p class="text-sm text-muted-foreground mt-0.5 truncate">
                        Course: {enrollCourse.title}
                    </p>
                </div>
                <Button.Root
                    variant="ghost"
                    size="icon-sm"
                    onclick={closeEnroll}
                    class="shrink-0 ml-3"
                >
                    <XCircle class="size-5" />
                </Button.Root>
            </div>

            <div class="px-6 py-4 flex flex-col gap-4 overflow-y-auto">
                {#if enrollError}
                    <div
                        class="flex items-center gap-2 rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-xs text-destructive"
                    >
                        <AlertTriangle class="size-3.5 shrink-0" />
                        <span>{enrollError}</span>
                    </div>
                {/if}

                <p class="text-xs text-muted-foreground">
                    Select a department to enroll all of its members in this
                    course.
                </p>

                {#if deptLoading}
                    <div class="flex justify-center py-8">
                        <LoaderCircle
                            class="size-5 text-muted-foreground animate-spin"
                        />
                    </div>
                {:else if departments.length === 0}
                    <div
                        class="flex flex-col items-center gap-2 py-8 text-muted-foreground"
                    >
                        <Building2 class="size-8" />
                        <p class="text-sm">No departments yet.</p>
                        <p class="text-xs">
                            Create departments in Settings or User Management
                            first.
                        </p>
                    </div>
                {:else}
                    <div class="flex flex-col gap-1">
                        {#each departments as dept (dept.id)}
                            <button
                                class="flex items-center gap-3 rounded-lg border px-3.5 py-2.5 text-left transition-colors {selectedDeptId ===
                                dept.id
                                    ? 'border-primary bg-primary/5'
                                    : 'border-border hover:border-muted-foreground/30'}"
                                onclick={() => (selectedDeptId = dept.id)}
                            >
                                <div
                                    class="size-8 rounded-lg bg-accent/10 flex items-center justify-center shrink-0"
                                >
                                    <Building2 class="size-4 text-accent" />
                                </div>
                                <div class="min-w-0 flex-1">
                                    <p
                                        class="text-sm font-medium text-foreground truncate"
                                    >
                                        {dept.name}
                                    </p>
                                    <p class="text-xs text-muted-foreground">
                                        {dept.user_count} member{dept.user_count ===
                                        1
                                            ? ""
                                            : "s"}
                                    </p>
                                </div>
                                <div
                                    class="size-5 rounded-full border-2 flex items-center justify-center shrink-0 {selectedDeptId ===
                                    dept.id
                                        ? 'border-primary'
                                        : 'border-muted-foreground/20'}"
                                >
                                    {#if selectedDeptId === dept.id}
                                        <div
                                            class="size-2.5 rounded-full bg-primary"
                                        ></div>
                                    {/if}
                                </div>
                            </button>
                        {/each}
                    </div>
                {/if}
            </div>

            <div
                class="flex items-center justify-end gap-2 px-6 py-4 border-t border-border shrink-0"
            >
                <Button.Root
                    variant="ghost"
                    size="sm"
                    onclick={closeEnroll}
                    disabled={enrollSaving}
                >
                    Cancel
                </Button.Root>
                <Button.Root
                    size="sm"
                    onclick={handleBulkEnroll}
                    disabled={!selectedDeptId || enrollSaving || deptLoading}
                >
                    {enrollSaving ? "Enrolling…" : "Enroll Department"}
                </Button.Root>
            </div>
        </div>
    </div>
{/if}
