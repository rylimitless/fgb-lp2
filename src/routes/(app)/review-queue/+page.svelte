<script lang="ts">
    import {
        ClipboardCheck,
        FileText,
        BookOpen,
        CheckCircle,
        XCircle,
        RotateCcw,
        ChevronLeft,
        ChevronRight,
        LoaderCircle,
        Eye,
    } from "@lucide/svelte";
    import * as Button from "$lib/components/ui/button";
    import { PageHeader, ModulePreview } from "$lib/components/brand";

    type TabType = "documents" | "courses";

    let activeTab = $state<TabType>("documents");

    let docs = $state<any[]>([]);
    let docsTotal = $state(0);
    let docsPage = $state(0);
    let docsLoading = $state(true);
    const pageSize = 15;

    let courses = $state<any[]>([]);
    let coursesTotal = $state(0);
    let coursesPage = $state(0);
    let coursesLoading = $state(true);

    let reviewItem = $state<any>(null);
    let reviewType = $state<"document" | "course">("document");
    let reviewNotes = $state("");

    // Document viewer state
    let viewingDoc = $state<any>(null);
    let viewingChunks = $state<any[]>([]);
    let viewingLoading = $state(false);

    // Course preview state — uses the same ModulePreview component as the
    // course builder, so reviewers see exactly what learners will see.
    let previewingCourse = $state<any>(null);
    let previewLoading = $state(false);

    async function loadDocuments(page: number) {
        docsLoading = true;
        docsPage = page;
        try {
            const res = await fetch(
                `/api/review/documents?limit=${pageSize}&offset=${page * pageSize}`,
                { credentials: "include" },
            );
            if (res.ok) {
                const data = await res.json();
                docs = data.items ?? [];
                docsTotal = data.total ?? 0;
            }
        } catch {
            /* ignore */
        }
        docsLoading = false;
    }

    async function loadCourses(page: number) {
        coursesLoading = true;
        coursesPage = page;
        try {
            const res = await fetch(
                `/api/review/courses?limit=${pageSize}&offset=${page * pageSize}`,
                { credentials: "include" },
            );
            if (res.ok) {
                const data = await res.json();
                courses = data.items ?? [];
                coursesTotal = data.total ?? 0;
            }
        } catch {
            /* ignore */
        }
        coursesLoading = false;
    }

    function openReview(item: any, type: "document" | "course") {
        reviewItem = item;
        reviewType = type;
        reviewNotes = "";
    }

    function closeReview() {
        reviewItem = null;
    }

    async function submitReviewDirect(
        id: number,
        type: "document" | "course",
        status: string,
    ) {
        const endpoint =
            type === "document"
                ? `/api/review/documents/${id}`
                : `/api/review/courses/${id}`;
        try {
            await fetch(endpoint, {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                credentials: "include",
                body: JSON.stringify({
                    review_status: status,
                    review_notes: "",
                }),
            });
            if (type === "document") loadDocuments(docsPage);
            else loadCourses(coursesPage);
        } catch {
            /* ignore */
        }
    }

    async function submitReview(status: string) {
        if (!reviewItem) return;
        const endpoint =
            reviewType === "document"
                ? `/api/review/documents/${reviewItem.id}`
                : `/api/review/courses/${reviewItem.id}`;

        try {
            await fetch(endpoint, {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                credentials: "include",
                body: JSON.stringify({
                    review_status: status,
                    review_notes: reviewNotes,
                }),
            });
            closeReview();
            if (reviewType === "document") loadDocuments(docsPage);
            else loadCourses(coursesPage);
        } catch {
            /* ignore */
        }
    }

    async function viewDocument(doc: any) {
        viewingDoc = doc;
        viewingChunks = [];
        viewingLoading = true;
        try {
            const res = await fetch(`/api/documents/${doc.id}/chunks`, {
                credentials: "include",
            });
            if (res.ok) {
                viewingChunks = await res.json();
            }
        } catch {
            /* ignore */
        }
        viewingLoading = false;
    }

    function closeViewer() {
        viewingDoc = null;
        viewingChunks = [];
    }

    // Fetch full course data (modules + items) and open the learner preview.
    // Reuses ModulePreview so reviewers see the same interactive experience
    // as the course builder.
    async function viewCourse(course: any) {
        previewLoading = true;
        try {
            const res = await fetch(`/api/courses/${course.id}`, {
                credentials: "include",
            });
            if (res.ok) {
                previewingCourse = await res.json();
            }
        } catch {
            /* ignore */
        }
        previewLoading = false;
    }

    function formatDate(d: string) {
        if (!d) return "";
        return new Date(d).toLocaleDateString("en-US", {
            year: "numeric",
            month: "short",
            day: "numeric",
        });
    }

    let totalPages = $derived(
        activeTab === "documents"
            ? Math.max(1, Math.ceil(docsTotal / pageSize))
            : Math.max(1, Math.ceil(coursesTotal / pageSize)),
    );
    let currentPage = $derived(
        activeTab === "documents" ? docsPage : coursesPage,
    );

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

    $effect(() => {
        loadDocuments(0);
        loadCourses(0);
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

    // Reset scroll when switching tabs or pages
    $effect(() => {
        activeTab; docsPage; coursesPage;
        scrollContainer?.scrollTo({ left: 0 });
    });
</script>

<div class="flex w-full max-w-6xl mx-auto flex-col gap-6">
    <PageHeader
        title="Review queue"
        eyebrow="Approvals"
        description="Approve, reject, or request changes for newly submitted documents and courses. Every decision is recorded."
    >
        {#snippet icon()}
            <ClipboardCheck class="size-6 text-primary" />
        {/snippet}
    </PageHeader>

    <div class="flex gap-1 rounded-lg bg-muted p-1 w-fit">
        <button
            class="rounded-md px-4 py-2 text-sm font-medium transition-colors {activeTab ===
            'documents'
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground'}"
            onclick={() => (activeTab = "documents")}
        >
            <FileText class="size-4 inline mr-1.5" />
            Documents ({docsTotal})
        </button>
        <button
            class="rounded-md px-4 py-2 text-sm font-medium transition-colors {activeTab ===
            'courses'
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground'}"
            onclick={() => (activeTab = "courses")}
        >
            <BookOpen class="size-4 inline mr-1.5" />
            Courses ({coursesTotal})
        </button>
    </div>

    <div class="rounded-xl border border-border bg-card overflow-hidden">
        {#if (activeTab === "documents" && docsLoading) || (activeTab === "courses" && coursesLoading)}
            <div class="flex items-center justify-center py-16">
                <LoaderCircle
                    class="size-6 text-muted-foreground animate-spin"
                />
            </div>
        {:else}
            <div class="relative">
                <div class="overflow-x-auto" bind:this={scrollContainer}>
                    <table class="w-full min-w-[640px]">
                <thead>
                    <tr class="border-b border-border bg-muted/50">
                        <th
                            class="px-5 py-3.5 text-left text-sm font-medium text-muted-foreground"
                            >Title</th
                        >
                        <th
                            class="px-5 py-3.5 text-left text-sm font-medium text-muted-foreground hidden sm:table-cell"
                            >Status</th
                        >
                        <th
                            class="px-5 py-3.5 text-left text-sm font-medium text-muted-foreground hidden md:table-cell"
                            >Created</th
                        >
                        <th
                            class="px-5 py-3.5 text-left text-sm font-medium text-muted-foreground whitespace-nowrap"
                            >Actions</th
                        >
                    </tr>
                </thead>
                <tbody>
                    {#if activeTab === "documents"}
                        {#each docs as doc (doc.id)}
                            <tr
                                class="border-b border-border last:border-0 hover:bg-muted/20"
                            >
                                <td class="px-5 py-3.5">
                                    <div
                                        class="flex items-center gap-2.5 min-w-0"
                                    >
                                        <FileText
                                            class="size-4 text-muted-foreground shrink-0"
                                        />
                                        <span
                                            class="text-sm font-medium truncate block"
                                            >{doc.title}</span
                                        >
                                    </div>
                                </td>
                                <td class="px-5 py-3.5 hidden sm:table-cell">
                                    <span class="text-sm text-muted-foreground"
                                        >{doc.status}</span
                                    >
                                </td>
                                <td
                                    class="px-5 py-3.5 hidden md:table-cell text-sm text-muted-foreground"
                                    >{formatDate(doc.created_at)}</td
                                >
                                <td class="px-5 py-3.5">
                                    <div class="flex items-center gap-2">
                                        <Button.Root
                                            variant="outline"
                                            size="sm"
                                            onclick={() => viewDocument(doc)}
                                            class="text-info border-info/20 hover:bg-info/10"
                                        >
                                            <Eye class="size-3.5 mr-1.5" />
                                            View
                                        </Button.Root>
                                        <Button.Root
                                            variant="outline"
                                            size="sm"
                                            onclick={() =>
                                                submitReviewDirect(
                                                    doc.id,
                                                    "document",
                                                    "approved",
                                                )}
                                            class="text-success border-success/20 hover:bg-success/10"
                                        >
                                            <CheckCircle
                                                class="size-3.5 mr-1.5"
                                            />
                                            Approve
                                        </Button.Root>
                                        <Button.Root
                                            variant="outline"
                                            size="sm"
                                            onclick={() =>
                                                openReview(doc, "document")}
                                            class="text-info border-info/20 hover:bg-info/10"
                                        >
                                            <RotateCcw
                                                class="size-3.5 mr-1.5"
                                            />
                                            Request Changes
                                        </Button.Root>
                                        <Button.Root
                                            variant="outline"
                                            size="sm"
                                            onclick={() =>
                                                openReview(doc, "document")}
                                            class="text-destructive border-destructive/20 hover:bg-destructive/10"
                                        >
                                            <XCircle class="size-3.5 mr-1.5" />
                                            Reject
                                        </Button.Root>
                                    </div>
                                </td>
                            </tr>
                        {/each}
                    {:else}
                        {#each courses as course (course.id)}
                            <tr
                                class="border-b border-border last:border-0 hover:bg-muted/20"
                            >
                                <td class="px-5 py-3.5">
                                    <div
                                        class="flex items-center gap-2.5 min-w-0"
                                    >
                                        <BookOpen
                                            class="size-4 text-muted-foreground shrink-0"
                                        />
                                        <span
                                            class="text-sm font-medium truncate block"
                                            >{course.title}</span
                                        >
                                    </div>
                                </td>
                                <td class="px-5 py-3.5 hidden sm:table-cell">
                                    <span class="text-sm text-muted-foreground"
                                        >{course.status}</span
                                    >
                                </td>
                                <td
                                    class="px-5 py-3.5 hidden md:table-cell text-sm text-muted-foreground"
                                    >{formatDate(course.created_at)}</td
                                >
                                <td class="px-5 py-3.5">
                                    <div class="flex items-center gap-2">
                                        <Button.Root
                                            variant="outline"
                                            size="sm"
                                            onclick={() => viewCourse(course)}
                                            disabled={previewLoading}
                                            class="text-info border-info/20 hover:bg-info/10"
                                        >
                                            {#if previewLoading}
                                                <LoaderCircle class="size-3.5 mr-1.5 animate-spin" />
                                            {:else}
                                                <Eye class="size-3.5 mr-1.5" />
                                            {/if}
                                            View
                                        </Button.Root>
                                        <Button.Root
                                            variant="outline"
                                            size="sm"
                                            onclick={() =>
                                                submitReviewDirect(
                                                    course.id,
                                                    "course",
                                                    "approved",
                                                )}
                                            class="text-success border-success/20 hover:bg-success/10"
                                        >
                                            <CheckCircle
                                                class="size-3.5 mr-1.5"
                                            />
                                            Approve
                                        </Button.Root>
                                        <Button.Root
                                            variant="outline"
                                            size="sm"
                                            onclick={() =>
                                                openReview(course, "course")}
                                            class="text-info border-info/20 hover:bg-info/10"
                                        >
                                            <RotateCcw
                                                class="size-3.5 mr-1.5"
                                            />
                                            Request Changes
                                        </Button.Root>
                                        <Button.Root
                                            variant="outline"
                                            size="sm"
                                            onclick={() =>
                                                openReview(course, "course")}
                                            class="text-destructive border-destructive/20 hover:bg-destructive/10"
                                        >
                                            <XCircle class="size-3.5 mr-1.5" />
                                            Reject
                                        </Button.Root>
                                    </div>
                                </td>
                            </tr>
                        {/each}
                    {/if}
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

            <div
                class="flex items-center justify-between px-5 py-3 border-t border-border bg-muted/20"
            >
                <span class="text-sm text-muted-foreground">
                    Page {currentPage + 1} of {totalPages}
                </span>
                <div class="flex gap-1">
                    <Button.Root
                        variant="ghost"
                        size="sm"
                        disabled={currentPage === 0}
                        onclick={() => {
                            if (activeTab === "documents")
                                loadDocuments(docsPage - 1);
                            else loadCourses(coursesPage - 1);
                        }}
                    >
                        <ChevronLeft class="size-4" />
                        Prev
                    </Button.Root>
                    <Button.Root
                        variant="ghost"
                        size="sm"
                        disabled={currentPage >= totalPages - 1}
                        onclick={() => {
                            if (activeTab === "documents")
                                loadDocuments(docsPage + 1);
                            else loadCourses(coursesPage + 1);
                        }}
                    >
                        Next
                        <ChevronRight class="size-4" />
                    </Button.Root>
                </div>
            </div>
        {/if}
    </div>
</div>

{#if reviewItem}
    <div
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
        onclick={closeReview}
    >
        <div
            class="bg-card border border-border rounded-xl shadow-xl w-full max-w-md p-6"
            onclick={(e: MouseEvent) => e.stopPropagation()}
        >
            <h3 class="text-base font-semibold text-foreground mb-1">
                Review {reviewType === "document" ? "Document" : "Course"}
            </h3>
            <p class="text-sm text-muted-foreground mb-4 truncate">
                {reviewItem.title}
            </p>

            <textarea
                bind:value={reviewNotes}
                rows={3}
                placeholder="Add review notes (optional)..."
                class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring resize-none mb-4"
            ></textarea>

            <div class="flex gap-2 justify-end">
                <Button.Root variant="ghost" size="sm" onclick={closeReview}
                    >Cancel</Button.Root
                >
                <Button.Root
                    variant="ghost"
                    size="sm"
                    onclick={() => submitReview("changes_requested")}
                    class="text-info"
                >
                    <RotateCcw class="size-4 mr-1.5" />
                    Request Changes
                </Button.Root>
                <Button.Root
                    variant="ghost"
                    size="sm"
                    onclick={() => submitReview("rejected")}
                    class="text-destructive"
                >
                    <XCircle class="size-4 mr-1.5" />
                    Reject
                </Button.Root>
                <Button.Root size="sm" onclick={() => submitReview("approved")}>
                    <CheckCircle class="size-4 mr-1.5" />
                    Approve
                </Button.Root>
            </div>
        </div>
    </div>
{/if}

{#if viewingDoc}
    <div
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
        onclick={closeViewer}
    >
        <div
            class="bg-card border border-border rounded-xl shadow-xl w-full max-w-2xl max-h-[85vh] flex flex-col"
            onclick={(e: MouseEvent) => e.stopPropagation()}
        >
            <div
                class="flex items-center justify-between px-6 py-4 border-b border-border"
            >
                <div class="min-w-0">
                    <h3
                        class="text-base font-semibold text-foreground truncate"
                    >
                        {viewingDoc.title}
                    </h3>
                    <p class="text-xs text-muted-foreground mt-0.5">
                        Document &middot; {viewingDoc.total_chunks ?? 0} chunks &middot;
                        Status: {viewingDoc.status}
                    </p>
                </div>
                <Button.Root
                    variant="ghost"
                    size="icon-sm"
                    onclick={closeViewer}
                    class="shrink-0 ml-3"
                >
                    <XCircle class="size-5" />
                </Button.Root>
            </div>
            <div class="overflow-y-auto px-6 py-4 flex-1">
                {#if viewingLoading}
                    <div class="flex items-center justify-center py-12">
                        <LoaderCircle
                            class="size-6 text-muted-foreground animate-spin"
                        />
                    </div>
                {:else if viewingChunks.length === 0}
                    <p class="text-sm text-muted-foreground text-center py-12">
                        No content chunks available for this document.
                    </p>
                {:else}
                    <div class="flex flex-col gap-4">
                        {#each viewingChunks as chunk, i}
                            <div
                                class="rounded-lg border border-border bg-muted/30 px-4 py-3"
                            >
                                <div class="flex items-center gap-2 mb-2">
                                    <span
                                        class="text-xs font-medium text-muted-foreground"
                                    >
                                        Chunk {chunk.chunk_index + 1}
                                    </span>
                                    {#if chunk.page_number}
                                        <span
                                            class="text-xs text-muted-foreground/60"
                                        >
                                            &middot; Page {chunk.page_number}
                                        </span>
                                    {/if}
                                    {#if chunk.source_label}
                                        <span
                                            class="text-xs text-muted-foreground/60"
                                        >
                                            &middot; {chunk.source_label}
                                        </span>
                                    {/if}
                                </div>
                                <p
                                    class="text-sm text-foreground/85 leading-relaxed whitespace-pre-wrap"
                                >
                                    {chunk.content}
                                </p>
                            </div>
                        {/each}
                    </div>
                {/if}
            </div>
        </div>
    </div>
{/if}

<!-- Course preview modal (same component as the course builder) -->
{#if previewingCourse}
    {@const firstModule = (previewingCourse.modules ?? []).find((m: any) => (m.items?.length ?? 0) > 0) ?? (previewingCourse.modules ?? [])[0]}
    {#if firstModule}
        <ModulePreview
            module={firstModule}
            courseTitle={previewingCourse.title}
            allModules={previewingCourse.modules ?? []}
            courseSources={previewingCourse.sources ?? []}
            onClose={() => (previewingCourse = null)}
        />
    {:else}
        <div class="fixed inset-0 z-50 bg-background/95 backdrop-blur-sm flex items-center justify-center" role="dialog" aria-modal="true">
            <div class="text-center">
                <p class="text-sm text-muted-foreground mb-4">This course has no modules to preview.</p>
                <Button.Root size="sm" onclick={() => (previewingCourse = null)}>Close</Button.Root>
            </div>
        </div>
    {/if}
{/if}
