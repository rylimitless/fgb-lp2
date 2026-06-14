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
    } from "@lucide/svelte";
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
                    return "text-amber-500";
                case "processing":
                    return "text-blue-500";
                case "ready":
                    return "text-emerald-500";
                case "failed":
                    return "text-red-500";
                default:
                    return "text-muted-foreground";
            }
        } else {
            switch (item.status) {
                case "published":
                    return "text-emerald-500";
                case "draft":
                    return "text-amber-500";
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
                    cls: "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400",
                };
            case "rejected":
                return {
                    label: "Rejected",
                    cls: "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400",
                };
            case "pending":
                return {
                    label: "Pending",
                    cls: "bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400",
                };
            case "changes_requested":
                return {
                    label: "Changes Requested",
                    cls: "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400",
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

    $effect(() => {
        loadContent();
    });
</script>

<div class="flex w-full max-w-6xl mx-auto flex-col gap-6">
    <div class="flex items-center gap-3">
        <Library class="size-6 text-primary" />
        <h1 class="text-2xl font-semibold tracking-tight text-foreground">
            Content Repository
        </h1>
    </div>

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
                                        class="inline-flex items-center gap-1 text-xs text-blue-600 dark:text-blue-400 hover:underline"
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
                                            class="text-blue-600 hover:text-blue-700"
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
                                    <Button.Root
                                        variant="ghost"
                                        size="icon-sm"
                                        onclick={() => handleDelete(item)}
                                        disabled={deletingId === item.id}
                                        class="text-muted-foreground hover:text-red-500"
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
