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
        Settings,
    } from "@lucide/svelte";
    import { PageHeader, showToast } from "$lib/components/brand";
    import * as Button from "$lib/components/ui/button";

    type ContentType = "all" | "document" | "course";

    let { data } = $props();

    // Admin/manager-only capabilities (enrollment viewing + department enrollment
    // both hit role-gated admin endpoints).
    let canViewEnrollments = $derived(
        data?.user?.roles?.some((r: string) =>
            ["admin", "manager"].includes(r),
        ) ?? false,
    );

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

    // Delete confirmation
    let deleteTarget = $state<any>(null);
    let deleteDialogOpen = $state(false);
    let deleteEnrollmentCount = $state(0);
    let deleteChecking = $state(false);

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
        deleteTarget = item;
        deleteDialogOpen = true;
        deleteEnrollmentCount = 0;

        // For courses, check enrollment count
        if (item.content_type === "course") {
            deleteChecking = true;
            try {
                const res = await fetch(
                    `/api/content-repository/courses/${item.id}/enrollment-count`,
                    { credentials: "include" },
                );
                if (res.ok) {
                    const data = await res.json();
                    deleteEnrollmentCount = data.enrollment_count ?? 0;
                }
            } catch {
                /* ignore */
            }
            deleteChecking = false;
        }
    }

    function closeDeleteDialog() {
        deleteTarget = null;
        deleteDialogOpen = false;
        deleteEnrollmentCount = 0;
    }

    async function confirmDelete() {
        if (!deleteTarget || deletingId !== null) return;
        const item = deleteTarget;
        closeDeleteDialog();

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
            } else {
                const err = await res.json();
                showToast(err.error ?? "Failed to delete", {
                    title: "Error",
                    variant: "error",
                });
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

    // Horizontal scroll indicators
    let scrollContainer = $state<HTMLElement | null>(null);
    let canScrollLeft = $state(false);
    let canScrollRight = $state(false);

    function checkScroll() {
        const el = scrollContainer;
        if (!el) {
            canScrollLeft = false;
            canScrollRight = false;
            return;
        }
        canScrollLeft = el.scrollLeft > 1;
        canScrollRight =
            el.scrollLeft < el.scrollWidth - el.clientWidth - 1;
    }

    function scrollTable(dir: "left" | "right") {
        scrollContainer?.scrollBy({
            left: dir === "left" ? -280 : 280,
            behavior: "smooth",
        });
    }

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

    // ---- Course Enrollments Viewer (admin/manager) ----
    type EnrollmentRow = {
        id: number;
        user_id: number;
        user_name: string;
        user_email: string;
        status: string;
        progress_pct: number;
        enrolled_at: string;
        completed_at: string | null;
        dropped_at: string | null;
    };
    type EnrollmentSummary = {
        active_count: number;
        completed_count: number;
        dropped_count: number;
        total_count: number;
    };

    let enrollmentsCourse = $state<any>(null);
    let enrollmentsOpen = $state(false);
    let enrollments = $state<EnrollmentRow[]>([]);
    let enrollmentSummary = $state<EnrollmentSummary | null>(null);
    let enrollmentsLoading = $state(false);
    let enrollmentsError = $state("");
    let enrollmentSearch = $state("");

    let filteredEnrollments = $derived(
        enrollmentSearch.trim()
            ? enrollments.filter((e) => {
                  const q = enrollmentSearch.toLowerCase();
                  return (
                      e.user_name.toLowerCase().includes(q) ||
                      e.user_email.toLowerCase().includes(q)
                  );
              })
            : enrollments,
    );

    let avgProgress = $derived(
        enrollmentSummary && enrollmentSummary.total_count > 0
            ? Math.round(
                  enrollments
                      .filter((e) => e.status !== "dropped")
                      .reduce((sum, e) => sum + (e.progress_pct ?? 0), 0) /
                      Math.max(
                          1,
                          enrollments.filter((e) => e.status !== "dropped")
                              .length,
                      ),
              )
            : 0,
    );

    async function openEnrollments(item: any) {
        enrollmentsCourse = item;
        enrollmentsOpen = true;
        enrollments = [];
        enrollmentSummary = null;
        enrollmentSearch = "";
        enrollmentsError = "";
        enrollmentsLoading = true;
        try {
            const res = await fetch(
                `/api/admin/enrollments/course/${item.id}`,
                { credentials: "include" },
            );
            if (!res.ok) {
                const err = await res.json().catch(() => ({}));
                enrollmentsError =
                    err.error ?? "Failed to load enrollments";
                enrollmentsLoading = false;
                return;
            }
            const data = await res.json();
            enrollments = (data.enrollments ?? []) as EnrollmentRow[];
            enrollmentSummary = {
                active_count: data.active_count ?? 0,
                completed_count: data.completed_count ?? 0,
                dropped_count: data.dropped_count ?? 0,
                total_count: data.total_count ?? 0,
            };
        } catch {
            enrollmentsError = "Network error";
        }
        enrollmentsLoading = false;
    }

    function closeEnrollments() {
        enrollmentsOpen = false;
        enrollmentsCourse = null;
        enrollments = [];
        enrollmentSummary = null;
        enrollmentsError = "";
        enrollmentSearch = "";
    }

    function enrollmentStatusBadge(status: string): {
        label: string;
        cls: string;
    } {
        switch (status) {
            case "completed":
                return {
                    label: "Completed",
                    cls: "bg-success/10 text-success border border-success/20",
                };
            case "dropped":
                return {
                    label: "Dropped",
                    cls: "bg-muted text-muted-foreground border border-border",
                };
            case "active":
            case "pending":
            default:
                return {
                    label: "In progress",
                    cls: "bg-info/10 text-info border border-info/20",
                };
        }
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

    // ---- Course Settings Dialog ----
    let settingsItem = $state<any>(null);
    let settingsOpen = $state(false);
    let settingsMaxAttempts = $state<number | null>(null);
    let settingsDaysToComplete = $state<number | null>(null);
    let settingsSaving = $state(false);
    let settingsError = $state("");

    function parseItemSettings(raw: any): any {
        if (!raw || raw === "{}") return {};
        try {
            return typeof raw === "string" ? JSON.parse(raw) : raw;
        } catch {
            return {};
        }
    }

    function openSettings(item: any) {
        settingsItem = item;
        const s = parseItemSettings(item.settings);
        settingsMaxAttempts = s.max_attempts ?? null;
        settingsDaysToComplete = s.days_to_complete ?? null;
        settingsError = "";
        settingsOpen = true;
    }

    function closeSettings() {
        settingsOpen = false;
        settingsItem = null;
        settingsMaxAttempts = null;
        settingsDaysToComplete = null;
        settingsError = "";
    }

    async function saveSettings() {
        if (!settingsItem) return;
        settingsSaving = true;
        settingsError = "";
        try {
            const clean: any = {};
            if (settingsMaxAttempts && settingsMaxAttempts > 0)
                clean.max_attempts = settingsMaxAttempts;
            if (settingsDaysToComplete && settingsDaysToComplete > 0)
                clean.days_to_complete = settingsDaysToComplete;
            const res = await fetch(
                `/api/courses/${settingsItem.id}/settings`,
                {
                    method: "PUT",
                    credentials: "include",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify({ settings: clean }),
                },
            );
            if (!res.ok) {
                const err = await res.json();
                settingsError = err.error ?? "Failed to save settings";
                return;
            }
            showToast("Course settings updated.", {
                title: "Settings",
                variant: "success",
            });
            closeSettings();
            loadContent();
        } catch {
            settingsError = "Network error";
        } finally {
            settingsSaving = false;
        }
    }

    $effect(() => {
        loadContent();
    });

    $effect(() => {
        const el = scrollContainer;
        if (!el) return;

        checkScroll();

        const handler = () => checkScroll();
        el.addEventListener("scroll", handler, { passive: true });
        const ro = new ResizeObserver(handler);
        ro.observe(el);

        return () => {
            el.removeEventListener("scroll", handler);
            ro.disconnect();
        };
    });

    // Reset scroll when page or filter changes
    $effect(() => {
        page; typeFilter; searchQuery;
        scrollContainer?.scrollTo({ left: 0 });
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

    <div class="rounded-xl border border-border bg-card overflow-hidden">
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
            <div class="relative">
                <div class="overflow-x-auto" bind:this={scrollContainer}>
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
                                    {#if item.content_type === "course"}
                                        <Button.Root
                                            variant="ghost"
                                            size="icon-sm"
                                            onclick={() => openSettings(item)}
                                            class="text-muted-foreground hover:text-primary"
                                            title="Course settings"
                                        >
                                            <Settings class="size-4" />
                                        </Button.Root>
                                    {/if}
                                    {#if canViewEnrollments &&
                                        item.content_type === "course" &&
                                        item.status === "published"}
                                        <Button.Root
                                            variant="ghost"
                                            size="icon-sm"
                                            onclick={() =>
                                                openEnrollments(item)}
                                            class="text-muted-foreground hover:text-primary"
                                            title="View enrollments & progress"
                                        >
                                            <GraduationCap class="size-4" />
                                        </Button.Root>
                                    {/if}
                                    {#if canViewEnrollments &&
                                        item.content_type === "course" &&
                                        item.status === "published"}
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
                </div>

                <!-- Scroll indicators -->
                {#if canScrollLeft}
                    <div class="pointer-events-none absolute inset-y-0 left-0 w-16 bg-linear-to-r from-card via-card/80 to-transparent"></div>
                    <button
                        class="absolute left-2 top-6 size-10 flex items-center justify-center rounded-full bg-card border border-border shadow-lg text-foreground hover:bg-muted hover:scale-110 transition-all"
                        onclick={() => scrollTable("left")}
                        title="Scroll left"
                    >
                        <ChevronLeft class="size-5" />
                    </button>
                {/if}
                {#if canScrollRight}
                    <div class="pointer-events-none absolute inset-y-0 right-0 w-16 bg-linear-to-l from-card via-card/80 to-transparent"></div>
                    <button
                        class="absolute right-2 top-6 size-10 flex items-center justify-center rounded-full bg-card border border-border shadow-lg text-foreground hover:bg-muted hover:scale-110 transition-all"
                        onclick={() => scrollTable("right")}
                        title="Scroll right"
                    >
                        <ChevronRight class="size-5" />
                    </button>
                {/if}
            </div>
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

<!-- Course Enrollments Viewer Dialog -->
{#if enrollmentsOpen && enrollmentsCourse}
    {@const summary = enrollmentSummary}
    <div
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
        onclick={closeEnrollments}
        role="dialog"
        aria-modal="true"
    >
        <div
            class="bg-card border border-border rounded-xl shadow-xl w-full max-w-3xl mx-4 max-h-[85vh] flex flex-col"
            onclick={(e: MouseEvent) => e.stopPropagation()}
        >
            <div
                class="flex items-center justify-between border-b border-border px-6 py-4 gap-4"
            >
                <div class="min-w-0">
                    <h3
                        class="text-base font-semibold text-foreground truncate"
                    >
                        {enrollmentsCourse.title}
                    </h3>
                    <p class="text-xs text-muted-foreground mt-0.5">
                        Enrollment & progress overview
                    </p>
                </div>
                <Button.Root
                    variant="ghost"
                    size="icon"
                    class="size-8 text-muted-foreground hover:text-foreground shrink-0"
                    onclick={closeEnrollments}
                >
                    <XCircle class="size-5" />
                </Button.Root>
            </div>

            <div
                class="px-6 py-4 flex flex-col gap-4 overflow-y-auto"
            >
                {#if enrollmentsLoading}
                    <div class="flex justify-center py-12">
                        <LoaderCircle
                            class="size-5 text-muted-foreground animate-spin"
                        />
                    </div>
                {:else if enrollmentsError}
                    <div
                        class="flex items-center gap-2 rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-xs text-destructive"
                    >
                        <AlertTriangle class="size-3.5 shrink-0" />
                        <span>{enrollmentsError}</span>
                    </div>
                {:else}
                    <!-- Summary stats -->
                    {#if summary}
                        <div
                            class="grid grid-cols-2 sm:grid-cols-4 gap-3"
                        >
                            <div
                                class="rounded-lg border border-border bg-muted/20 p-3"
                            >
                                <p
                                    class="text-xs text-muted-foreground"
                                >
                                    Enrolled
                                </p>
                                <p
                                    class="text-xl font-semibold text-foreground"
                                >
                                    {summary.total_count}
                                </p>
                            </div>
                            <div
                                class="rounded-lg border border-border bg-muted/20 p-3"
                            >
                                <p
                                    class="text-xs text-muted-foreground"
                                >
                                    In progress
                                </p>
                                <p
                                    class="text-xl font-semibold text-info"
                                >
                                    {summary.active_count}
                                </p>
                            </div>
                            <div
                                class="rounded-lg border border-border bg-muted/20 p-3"
                            >
                                <p
                                    class="text-xs text-muted-foreground"
                                >
                                    Completed
                                </p>
                                <p
                                    class="text-xl font-semibold text-success"
                                >
                                    {summary.completed_count}
                                </p>
                            </div>
                            <div
                                class="rounded-lg border border-border bg-muted/20 p-3"
                            >
                                <p
                                    class="text-xs text-muted-foreground"
                                >
                                    Avg progress
                                </p>
                                <p
                                    class="text-xl font-semibold text-foreground"
                                >
                                    {avgProgress}%
                                </p>
                            </div>
                        </div>
                    {/if}

                    {#if enrollments.length === 0}
                        <div
                            class="flex flex-col items-center justify-center text-center py-10 gap-2"
                        >
                            <Users class="size-8 text-muted-foreground/50" />
                            <p class="text-sm font-medium text-foreground">
                                No enrollments yet
                            </p>
                            <p class="text-xs text-muted-foreground">
                                Enroll users or a department to see progress
                                here.
                            </p>
                        </div>
                    {:else}
                        <!-- Search -->
                        <div class="relative">
                            <Search
                                class="size-4 absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground pointer-events-none"
                            />
                            <input
                                type="text"
                                placeholder="Search by name or email…"
                                class="w-full rounded-lg border border-border bg-background pl-9 pr-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/30"
                                bind:value={enrollmentSearch}
                            />
                        </div>

                        <!-- Enrollment table -->
                        <div
                            class="rounded-lg border border-border overflow-hidden"
                        >
                            <div class="max-h-80 overflow-y-auto">
                                <table class="w-full text-sm">
                                    <thead
                                        class="bg-muted/30 text-left text-xs text-muted-foreground sticky top-0"
                                    >
                                        <tr>
                                            <th
                                                class="px-3 py-2 font-medium"
                                            >
                                                Learner
                                            </th>
                                            <th
                                                class="px-3 py-2 font-medium"
                                            >
                                                Status
                                            </th>
                                            <th
                                                class="px-3 py-2 font-medium"
                                            >
                                                Progress
                                            </th>
                                            <th
                                                class="px-3 py-2 font-medium hidden sm:table-cell"
                                            >
                                                Enrolled
                                            </th>
                                        </tr>
                                    </thead>
                                    <tbody>
                                        {#each filteredEnrollments as e (e.id)}
                                            {@const badge =
                                                enrollmentStatusBadge(e.status)}
                                            <tr
                                                class="border-t border-border/40 hover:bg-muted/20"
                                            >
                                                <td
                                                    class="px-3 py-2"
                                                >
                                                    <p
                                                        class="font-medium text-foreground"
                                                    >
                                                        {e.user_name}
                                                    </p>
                                                    <p
                                                        class="text-xs text-muted-foreground"
                                                    >
                                                        {e.user_email}
                                                    </p>
                                                </td>
                                                <td
                                                    class="px-3 py-2"
                                                >
                                                    <span
                                                        class="inline-flex items-center rounded px-1.5 py-0.5 text-[10px] {badge.cls}"
                                                    >
                                                        {badge.label}
                                                    </span>
                                                </td>
                                                <td
                                                    class="px-3 py-2"
                                                >
                                                    {#if e.status === "completed"}
                                                        <span
                                                            class="text-success font-medium"
                                                        >
                                                            100%
                                                        </span>
                                                    {:else if e.status === "dropped"}
                                                        <span
                                                            class="text-muted-foreground"
                                                        >
                                                            —
                                                        </span>
                                                    {:else}
                                                        <div
                                                            class="flex items-center gap-2"
                                                        >
                                                            <div
                                                                class="w-20 h-1.5 rounded-full bg-muted overflow-hidden"
                                                            >
                                                                <div
                                                                    class="h-full bg-primary"
                                                                    style="width: {Math.min(100, Math.max(0, e.progress_pct ?? 0))}%"
                                                                ></div>
                                                            </div>
                                                            <span
                                                                class="text-xs text-muted-foreground tabular-nums"
                                                            >
                                                                {Math.round(e.progress_pct ?? 0)}%
                                                            </span>
                                                        </div>
                                                    {/if}
                                                </td>
                                                <td
                                                    class="px-3 py-2 text-xs text-muted-foreground hidden sm:table-cell"
                                                >
                                                    {e.enrolled_at || "—"}
                                                </td>
                                            </tr>
                                        {/each}
                                    </tbody>
                                </table>
                            </div>
                        </div>
                        {#if filteredEnrollments.length === 0}
                            <p
                                class="text-xs text-muted-foreground text-center py-2"
                            >
                                No learners match “{enrollmentSearch}”.
                            </p>
                        {/if}
                    {/if}
                {/if}
            </div>

            <div
                class="flex items-center justify-end border-t border-border px-6 py-3 gap-2"
            >
                <Button.Root variant="outline" size="sm" onclick={closeEnrollments}>
                    Close
                </Button.Root>
            </div>
        </div>
    </div>
{/if}

<!-- Course Settings Dialog -->
{#if settingsOpen && settingsItem}
    <div
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
        onclick={closeSettings}
        role="dialog"
        aria-modal="true"
    >
        <div
            class="bg-card border border-border rounded-xl shadow-xl w-full max-w-md mx-4 flex flex-col"
            onclick={(e: MouseEvent) => e.stopPropagation()}
        >
            <div
                class="flex items-center justify-between px-6 py-4 border-b border-border shrink-0"
            >
                <div class="min-w-0">
                    <h3 class="text-base font-semibold text-foreground">
                        Course Settings
                    </h3>
                    <p class="text-sm text-muted-foreground mt-0.5 truncate">
                        {settingsItem.title}
                    </p>
                </div>
                <Button.Root
                    variant="ghost"
                    size="icon-sm"
                    onclick={closeSettings}
                    class="shrink-0 ml-3"
                >
                    <XCircle class="size-5" />
                </Button.Root>
            </div>

            <div class="px-6 py-4 flex flex-col gap-4">
                {#if settingsError}
                    <div
                        class="flex items-center gap-2 rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-xs text-destructive"
                    >
                        <AlertTriangle class="size-3.5 shrink-0" />
                        <span>{settingsError}</span>
                    </div>
                {/if}

                <p class="text-xs text-muted-foreground">
                    Control how learners can take this course. Leave a field
                    empty to remove that limit.
                </p>

                <label class="flex flex-col gap-1.5">
                    <span class="text-xs font-medium text-muted-foreground">
                        Max Attempts
                        <span class="text-muted-foreground/50 font-normal">
                            (times a learner can take it)</span
                        >
                    </span>
                    <input
                        type="number"
                        min="1"
                        bind:value={settingsMaxAttempts}
                        placeholder="Unlimited"
                        class="rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                    />
                </label>

                <label class="flex flex-col gap-1.5">
                    <span class="text-xs font-medium text-muted-foreground">
                        Days to Complete
                        <span class="text-muted-foreground/50 font-normal">
                            (deadline from enrollment)</span
                        >
                    </span>
                    <input
                        type="number"
                        min="1"
                        bind:value={settingsDaysToComplete}
                        placeholder="No deadline"
                        class="rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                    />
                </label>
            </div>

            <div
                class="flex items-center justify-end gap-2 px-6 py-4 border-t border-border shrink-0"
            >
                <Button.Root
                    variant="ghost"
                    size="sm"
                    onclick={closeSettings}
                    disabled={settingsSaving}
                >
                    Cancel
                </Button.Root>
                <Button.Root
                    size="sm"
                    onclick={saveSettings}
                    disabled={settingsSaving}
                >
                    {settingsSaving ? "Saving…" : "Save Settings"}
                </Button.Root>
            </div>
        </div>
    </div>
{/if}

<!-- Delete Confirmation Dialog -->
{#if deleteDialogOpen && deleteTarget}
    <div
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
        onclick={closeDeleteDialog}
        role="dialog"
        aria-modal="true"
    >
        <div
            class="bg-card border border-border rounded-xl shadow-xl w-full max-w-md mx-4 flex flex-col"
            onclick={(e: MouseEvent) => e.stopPropagation()}
        >
            <div
                class="flex items-center justify-between px-6 py-4 border-b border-border shrink-0"
            >
                <div class="min-w-0">
                    <h3 class="text-base font-semibold text-foreground">
                        Delete {deleteTarget.content_type === "document" ? "Document" : "Course"}
                    </h3>
                    <p class="text-sm text-muted-foreground mt-0.5 truncate">
                        {deleteTarget.title}
                    </p>
                </div>
                <Button.Root
                    variant="ghost"
                    size="icon-sm"
                    onclick={closeDeleteDialog}
                    class="shrink-0 ml-3"
                >
                    <XCircle class="size-5" />
                </Button.Root>
            </div>

            <div class="px-6 py-4 flex flex-col gap-3">
                <p class="text-sm text-foreground">
                    Are you sure you want to permanently delete this {deleteTarget.content_type === "document" ? "document" : "course"}? This action cannot be undone.
                </p>

                {#if deleteTarget.content_type === "course"}
                    {#if deleteChecking}
                        <div class="flex items-center gap-2 text-xs text-muted-foreground">
                            <LoaderCircle class="size-3.5 animate-spin" />
                            Checking enrollments…
                        </div>
                    {:else if deleteEnrollmentCount > 0}
                        <div
                            class="flex items-center gap-2 rounded-lg border border-warning/30 bg-warning/10 p-3 text-xs text-warning"
                        >
                            <AlertTriangle class="size-4 shrink-0" />
                            <span>
                                {deleteEnrollmentCount} student{deleteEnrollmentCount === 1 ? " is" : "s are"} enrolled in this course. Deleting will also remove their enrollment records.
                            </span>
                        </div>
                    {/if}
                {/if}
            </div>

            <div
                class="flex items-center justify-end gap-2 px-6 py-4 border-t border-border shrink-0"
            >
                <Button.Root
                    variant="ghost"
                    size="sm"
                    onclick={closeDeleteDialog}
                >
                    Cancel
                </Button.Root>
                <Button.Root
                    variant="ghost"
                    size="sm"
                    onclick={confirmDelete}
                    class="text-destructive hover:bg-destructive/10"
                >
                    Delete
                </Button.Root>
            </div>
        </div>
    </div>
{/if}
