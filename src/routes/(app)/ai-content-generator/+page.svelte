<script lang="ts">
    import {
        Sparkles,
        BookOpen,
        LoaderCircle,
        ChevronDown,
        ChevronLeft,
        ChevronRight,
        ChevronUp,
        FileText,
        CheckCircle,
        XCircle,
        Clock,
        ArrowRight,
        Layers,
        Bookmark,
        ExternalLink,
        Play,
    } from "@lucide/svelte";
    import * as Button from "$lib/components/ui/button";
    import Markdown from "$lib/components/brand/Markdown.svelte";
    import AiItemEditWrapper from "$lib/components/brand/AiItemEditWrapper.svelte";

    let { data } = $props();

    let courses = $state<any[]>([]);
    let allDocs = $state<any[]>([]);
    let approvedDocs = $derived(
        allDocs.filter((d: any) => d.approved && d.status === "ready"),
    );
    let processingDocs = $derived(
        allDocs.filter(
            (d: any) => d.status === "uploaded" || d.status === "processing",
        ),
    );

    $effect(() => {
        courses = data.courses ?? [];
        allDocs = data.documents ?? [];
    });

    async function refreshDocs() {
        try {
            const res = await fetch("/api/documents", {
                credentials: "include",
            });
            if (res.ok) {
                allDocs = await res.json();
            }
        } catch {
            // ignore transient errors; next tick will retry
        }
    }

    $effect(() => {
        const interval = setInterval(() => {
            if (processingDocs.length > 0) {
                refreshDocs();
            }
        }, 3000);
        return () => clearInterval(interval);
    });

    // ---- Form state ----
    let title = $state("");
    let description = $state("");
    let selectedDocIds = $state<number[]>([]);
    let generating = $state(false);
    let genError = $state("");
    let activeJobId = $state<string | null>(null);

    // ---- Generation progress (SSE streaming) ----
    let generationSteps = $state<{ step: string; detail: string }[]>([]);
    let generationDone = $state(false);
    let streamedModules = $state<any[]>([]);
    let streamProgress = $state("");

    // ---- Selected course view ----
    let selectedCourseId = $state<number | null>(null);
    let viewingCourse = $state<any>(null);
    let viewingLoading = $state(false);

    // ---- Expanded modules ----
    let expandedModules = $state<Record<number, boolean>>({});
    let showSources = $state(false);

    // ---- AI Edit ----
    let editInstructions = $state("");
    let editing = $state(false);
    let sidebarCollapsed = $state(false);
    let generateCollapsed = $state(false);
    let editError = $state("");

    // ---- Per-item AI Edit ----
    let editingItemId = $state<number | null>(null);
    let itemEditInstructions = $state("");
    let itemEditLoading = $state(false);
    let itemEditError = $state("");
    // Preview: { item_id, item_type, current_data, suggested_data }
    let itemEditPreview = $state<any>(null);
    let itemEditAcceptedIds = $state<Set<number>>(new Set());

    function toggleModule(id: number) {
        expandedModules[id] = !expandedModules[id];
    }

    function toggleDoc(docId: number) {
        if (selectedDocIds.includes(docId)) {
            selectedDocIds = selectedDocIds.filter((id) => id !== docId);
        } else {
            selectedDocIds = [...selectedDocIds, docId];
        }
    }

    async function handleGenerate() {
        genError = "";
        generationSteps = [];
        generationDone = false;
        streamedModules = [];
        streamProgress = "";
        viewingCourse = null;
        generating = true;

        try {
            const enqueueRes = await fetch("/api/courses/generate", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                credentials: "include",
                body: JSON.stringify({
                    title: title || "Untitled Course",
                    description,
                    source_doc_ids: selectedDocIds,
                }),
            });

            if (!enqueueRes.ok) {
                genError = "Failed to start generation";
                generating = false;
                return;
            }

            const { job_id } = await enqueueRes.json();
            activeJobId = job_id;

            await streamJobEvents(job_id);
        } catch {
            if (!genError) genError = "Network error. Is the backend running?";
        } finally {
            generating = false;
            activeJobId = null;
        }
    }

    async function streamJobEvents(jobId: string) {
        const streamRes = await fetch(`/api/courses/generate/${jobId}/stream`, {
            credentials: "include",
        });

        if (!streamRes.ok) {
            genError = "Failed to connect to stream";
            return;
        }

        const reader = streamRes.body?.getReader();
        if (!reader) {
            genError = "Streaming not supported";
            return;
        }

        const decoder = new TextDecoder();
        let buffer = "";
        let eventType = "";

        while (true) {
            const { done, value } = await reader.read();
            if (done) break;
            buffer += decoder.decode(value, { stream: true });
            const lines = buffer.split("\n");
            buffer = lines.pop() || "";

            for (const line of lines) {
                if (line.startsWith("event: ")) {
                    eventType = line.slice(7).trim();
                } else if (line.startsWith("data: ")) {
                    try {
                        const payload = JSON.parse(line.slice(6));
                        if (eventType === "step") {
                            generationSteps = [...generationSteps, payload];
                            if (
                                payload.step === "writing" ||
                                payload.step === "packaging"
                            ) {
                                streamProgress = payload.detail;
                            }
                        } else if (eventType === "module") {
                            streamedModules = [
                                ...streamedModules,
                                payload.module,
                            ];
                            if (payload.progress) {
                                streamProgress = `Module ${payload.progress} generated`;
                            }
                        } else if (eventType === "error") {
                            genError =
                                payload.message ||
                                "Generation failed — check backend logs for details";
                            return;
                        } else if (eventType === "done") {
                            generationDone = true;
                            viewingCourse = payload;
                            selectedCourseId = payload.id;
                            expandedModules = {};
                            courses = [payload, ...courses];
                        }
                        eventType = "";
                    } catch (e) {
                        console.error("SSE parse error:", e, line);
                    }
                }
            }
        }
    }

    // Auto-reconnect to in-progress generation on page load.
    $effect(() => {
        pollForActiveJob();
    });

    let autoReconnectDone = false;

    async function pollForActiveJob() {
        if (autoReconnectDone || generating) return;
        try {
            const res = await fetch("/api/courses/generate/active", {
                credentials: "include",
            });
            if (!res.ok) return;
            const jobs = await res.json();
            if (jobs.length > 0) {
                autoReconnectDone = true;
                const job = jobs[0];
                generating = true;
                activeJobId = job.id;
                generationSteps = job.steps ?? [];
                generationDone = false;
                // Don't pre-seed streamedModules — the SSE replay will
                // push them, and we want to avoid duplicates.
                streamedModules = [];
                streamProgress = "";
                genError = "";

                try {
                    await streamJobEvents(job.id);
                } catch {
                    if (!genError) genError = "Stream connection lost";
                } finally {
                    generating = false;
                    activeJobId = null;
                }
            }
        } catch {
            // Backend may not be ready yet
        }
    }

    async function handleEdit(courseId: number) {
        if (!editInstructions.trim()) return;
        editError = "";
        editing = true;
        try {
            const res = await fetch(`/api/courses/${courseId}/edit`, {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                credentials: "include",
                body: JSON.stringify({ instructions: editInstructions }),
            });
            if (!res.ok) {
                editError = "Edit failed";
                return;
            }
            const updated = await res.json();
            viewingCourse = updated;
            editInstructions = "";
            expandedModules = {};
        } catch {
            editError = "Network error";
        } finally {
            editing = false;
        }
    }

    // ---- Per-item AI Edit handlers ----

    function startItemEdit(itemId: number) {
        editingItemId = itemId;
        itemEditInstructions = "";
        itemEditError = "";
        itemEditPreview = null;
    }

    function cancelItemEdit() {
        editingItemId = null;
        itemEditInstructions = "";
        itemEditError = "";
        itemEditPreview = null;
        itemEditLoading = false;
    }

    async function handleItemAiEdit() {
        if (
            !itemEditInstructions.trim() ||
            editingItemId === null ||
            !viewingCourse
        )
            return;
        itemEditError = "";
        itemEditLoading = true;
        itemEditPreview = null;
        try {
            const res = await fetch(`/api/items/${editingItemId}/ai-edit`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                credentials: "include",
                body: JSON.stringify({
                    instructions: itemEditInstructions,
                }),
            });
            if (!res.ok) {
                const err = await res.json().catch(() => ({}));
                itemEditError = err.error || "AI edit failed";
                return;
            }
            itemEditPreview = await res.json();
        } catch {
            itemEditError = "Network error";
        } finally {
            itemEditLoading = false;
        }
    }

    async function acceptItemEdit() {
        if (!itemEditPreview || !viewingCourse) return;
        try {
            const res = await fetch(`/api/items/${itemEditPreview.item_id}`, {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                credentials: "include",
                body: JSON.stringify({
                    data: itemEditPreview.suggested_data,
                }),
            });
            if (!res.ok) {
                itemEditError = "Failed to save edit";
                return;
            }
            // Update local state
            const updated = await res.json();
            viewingCourse = {
                ...viewingCourse,
                modules: viewingCourse.modules.map((mod: any) => ({
                    ...mod,
                    items: mod.items.map((item: any) =>
                        item.id === updated.id
                            ? { ...item, data: updated.data }
                            : item,
                    ),
                })),
            };
            itemEditAcceptedIds = new Set([
                ...itemEditAcceptedIds,
                itemEditPreview.item_id,
            ]);
            cancelItemEdit();
        } catch {
            itemEditError = "Network error while saving";
        }
    }

    function discardItemEdit() {
        itemEditPreview = null;
        itemEditError = "";
    }

    function onItemInstructionsInput(e: Event) {
        itemEditInstructions = (e.target as HTMLInputElement).value;
    }

    async function deleteItem(itemId: number) {
        if (!viewingCourse) return;
        if (!confirm("Delete this question? This cannot be undone.")) return;
        try {
            const res = await fetch(`/api/items/${itemId}`, {
                method: "DELETE",
                credentials: "include",
            });
            if (!res.ok) {
                itemEditError = "Failed to delete item";
                return;
            }
            // Remove from local state
            viewingCourse = {
                ...viewingCourse,
                modules: viewingCourse.modules.map((mod: any) => ({
                    ...mod,
                    items: (mod.items ?? []).filter(
                        (item: any) => item.id !== itemId,
                    ),
                })),
            };
            cancelItemEdit();
        } catch {
            itemEditError = "Network error while deleting";
        }
    }

    async function viewCourseDetail(course: any) {
        selectedCourseId = course.id;
        viewingLoading = true;
        expandedModules = {};
        showSources = false;

        try {
            const res = await fetch(`/api/courses/${course.id}`, {
                credentials: "include",
            });
            if (res.ok) {
                viewingCourse = await res.json();
            }
        } catch {
            // ignore
        } finally {
            viewingLoading = false;
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

    function itemPreview(item: any): string {
        const d = itemData(item);
        switch (item.item_type) {
            case "content":
                return d.body ?? "";
            case "mc":
            case "ma":
                return d.question ?? "";
            case "tf":
                return d.statement ?? "";
            case "fb":
                return d.text ?? "";
            case "sa":
                return d.question ?? "";
            case "matching":
                return `${(d.pairs ?? []).length} pairs`;
            case "drag_sort":
                return `${(d.items ?? []).length} items`;
            case "sequence":
                return `${(d.steps ?? []).length} steps`;
            default:
                return "";
        }
    }

    function moduleCount(mod: any): number {
        return mod.items?.length ?? 0;
    }
</script>

<div class="flex w-full max-w-7xl mx-auto gap-6">
    <!-- Collapse toggle button -->
    <button
        class="mt-2 size-6 rounded-md border border-border bg-card flex items-center justify-center shrink-0 hover:bg-muted transition-colors press"
        onclick={() => (sidebarCollapsed = !sidebarCollapsed)}
        title={sidebarCollapsed ? "Show sidebar" : "Hide sidebar"}
    >
        {#if sidebarCollapsed}
            <ChevronRight class="size-3.5 text-muted-foreground" />
        {:else}
            <ChevronLeft class="size-3.5 text-muted-foreground" />
        {/if}
    </button>

    <!-- Left: Form + Course List -->
    <section
        class="flex flex-col gap-4 transition-all duration-200 ease-in-out overflow-hidden {sidebarCollapsed
            ? 'w-0 min-w-0 opacity-0 p-0 m-0'
            : 'w-[400px] shrink-0'}"
    >
        <!-- Generate Form -->
        <div
            class="rounded-xl border border-border bg-card p-6 whitespace-nowrap"
        >
            <button
                class="w-full flex items-center justify-between gap-2 text-sm font-semibold tracking-tight text-foreground mb-0"
                onclick={() => (generateCollapsed = !generateCollapsed)}
            >
                <span class="flex items-center gap-2">
                    <Sparkles class="size-4 text-primary" />
                    Generate Course
                </span>
                {#if generateCollapsed}
                    <ChevronDown
                        class="size-4 text-muted-foreground shrink-0"
                    />
                {:else}
                    <ChevronUp class="size-4 text-muted-foreground shrink-0" />
                {/if}
            </button>
            {#if !generateCollapsed}
                <div class="flex flex-col gap-4 mt-4">
                    <div class="flex flex-col gap-1.5">
                        <label
                            for="course-title"
                            class="text-xs font-medium text-muted-foreground"
                        >
                            Title (optional)
                        </label>
                        <input
                            id="course-title"
                            type="text"
                            bind:value={title}
                            placeholder="e.g., Introduction to Machine Learning"
                            class="rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                        />
                    </div>
                    <div class="flex flex-col gap-1.5">
                        <label
                            for="course-desc"
                            class="text-xs font-medium text-muted-foreground"
                        >
                            Description or content to base the course on
                        </label>
                        <textarea
                            id="course-desc"
                            bind:value={description}
                            rows={4}
                            placeholder="Paste paper text, article content, or describe what the course should cover..."
                            class="rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring resize-none"
                        ></textarea>
                    </div>

                    {#if approvedDocs.length > 0 || processingDocs.length > 0}
                        <div class="flex flex-col gap-1.5">
                            <p
                                class="text-xs font-medium text-muted-foreground"
                            >
                                Source documents ({approvedDocs.length} approved{processingDocs.length
                                    ? `, ${processingDocs.length} processing`
                                    : ""})
                            </p>
                            <div
                                class="flex flex-col gap-1 max-h-32 overflow-y-auto"
                            >
                                {#each approvedDocs as doc}
                                    <button
                                        class="flex items-center gap-2 rounded-md px-2.5 py-1.5 text-xs transition-colors {selectedDocIds.includes(
                                            doc.id,
                                        )
                                            ? 'bg-primary/10 text-foreground'
                                            : 'text-muted-foreground hover:bg-muted hover:text-foreground'}"
                                        onclick={() => toggleDoc(doc.id)}
                                    >
                                        <div
                                            class="size-3.5 rounded border flex items-center justify-center shrink-0 {selectedDocIds.includes(
                                                doc.id,
                                            )
                                                ? 'bg-primary border-primary text-primary-foreground'
                                                : 'border-border'}"
                                        >
                                            {#if selectedDocIds.includes(doc.id)}
                                                <CheckCircle class="size-2.5" />
                                            {/if}
                                        </div>
                                        <FileText class="size-3 shrink-0" />
                                        <span class="truncate">{doc.title}</span
                                        >
                                    </button>
                                {/each}
                                {#each processingDocs as doc}
                                    <div
                                        class="flex items-center gap-2 rounded-md px-2.5 py-1.5 text-xs text-muted-foreground/70"
                                        title="Document is still being processed"
                                    >
                                        <LoaderCircle
                                            class="size-3 animate-spin shrink-0"
                                        />
                                        <FileText class="size-3 shrink-0" />
                                        <span class="truncate">{doc.title}</span
                                        >
                                        {#if doc.total_chunks > 0}
                                            <span
                                                class="ml-auto text-[10px] shrink-0"
                                            >
                                                {doc.chunks_done}/{doc.total_chunks}
                                            </span>
                                        {:else}
                                            <span
                                                class="ml-auto text-[10px] shrink-0"
                                            >
                                                {doc.status}
                                            </span>
                                        {/if}
                                    </div>
                                {/each}
                            </div>
                        </div>
                    {/if}

                    {#if genError}
                        <div
                            class="rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 flex items-center gap-2"
                        >
                            <XCircle class="size-4 text-destructive shrink-0" />
                            <p class="text-xs text-destructive">
                                {genError}
                            </p>
                        </div>
                    {/if}

                    <Button.Root
                        class="w-full"
                        disabled={generating || !description.trim()}
                        onclick={handleGenerate}
                    >
                        {#if generating}
                            <LoaderCircle class="size-4 mr-2 animate-spin" />
                            Generating course…
                        {:else}
                            <Sparkles class="size-4 mr-2" />
                            Generate Course
                        {/if}
                    </Button.Root>
                </div>
            {/if}
        </div>

        <!-- Course List -->
        <div class="rounded-xl border border-border bg-card p-6">
            <h3
                class="text-sm font-semibold tracking-tight text-foreground mb-3 flex items-center gap-2"
            >
                <BookOpen class="size-4 text-muted-foreground" />
                Courses ({courses.length})
            </h3>

            {#if courses.length === 0}
                <p class="text-xs text-muted-foreground text-center py-4">
                    No courses yet. Generate one to get started.
                </p>
            {:else}
                <div class="flex flex-col gap-1">
                    {#each courses as course}
                        <button
                            class="flex items-center gap-2.5 rounded-md px-2.5 py-2 text-left transition-colors {selectedCourseId ===
                            course.id
                                ? 'bg-primary/10 text-foreground'
                                : 'text-muted-foreground hover:bg-muted hover:text-foreground'}"
                            onclick={() => viewCourseDetail(course)}
                        >
                            <BookOpen class="size-3.5 shrink-0" />
                            <div class="min-w-0 flex-1">
                                <p class="text-xs font-medium truncate">
                                    {course.title}
                                </p>
                                <p class="text-xs text-muted-foreground">
                                    {course.status}
                                    {#if course.source_doc_ids?.length}
                                        · {course.source_doc_ids.length}
                                        source(s)
                                    {/if}
                                </p>
                            </div>
                            <ArrowRight class="size-3 shrink-0" />
                        </button>
                    {/each}
                </div>
            {/if}
        </div>
    </section>

    <!-- Right: Course Detail / Progress -->
    <section class="flex-1 min-w-0">
        {#if generating}
            <div class="rounded-xl border border-border bg-card p-6">
                <h3
                    class="text-sm font-semibold text-foreground mb-3 flex items-center gap-2"
                >
                    <Sparkles class="size-4 text-primary" />
                    Building your course
                </h3>

                <!-- Step checklist: show last 3, collapse rest -->
                {#if generationSteps.length > 0}
                    <div class="flex flex-col gap-1 mb-3">
                        {#each generationSteps.slice(-3) as step, i}
                            {@const globalIdx = generationSteps.length - 3 + i}
                            <div class="flex items-center gap-2">
                                <div
                                    class="size-3.5 rounded-full shrink-0 flex items-center justify-center {globalIdx <
                                        generationSteps.length - 1 ||
                                    generationDone
                                        ? 'bg-success/20 text-success'
                                        : 'bg-info/20 text-info'}"
                                >
                                    {#if globalIdx < generationSteps.length - 1 || generationDone}
                                        <CheckCircle class="size-2" />
                                    {:else}
                                        <LoaderCircle
                                            class="size-2 animate-spin"
                                        />
                                    {/if}
                                </div>
                                <p
                                    class="text-xs text-muted-foreground truncate"
                                >
                                    {step.detail}
                                </p>
                            </div>
                        {/each}
                        {#if generationSteps.length > 3}
                            <p
                                class="text-xs text-muted-foreground/50 pl-[22px]"
                            >
                                +{generationSteps.length - 3} previous steps
                            </p>
                        {/if}
                    </div>
                {:else}
                    <div class="flex items-center gap-2 mb-3">
                        <LoaderCircle
                            class="size-3.5 text-muted-foreground animate-spin shrink-0"
                        />
                        <p class="text-xs text-muted-foreground">
                            Connecting to generation service...
                        </p>
                    </div>
                {/if}

                <!-- Error display -->
                {#if genError}
                    <div
                        class="rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 mb-3 flex items-start gap-2"
                    >
                        <XCircle
                            class="size-3.5 text-destructive shrink-0 mt-0.5"
                        />
                        <div>
                            <p class="text-xs font-medium text-destructive">
                                Generation failed
                            </p>
                            <p class="text-xs text-destructive/80 mt-0.5">
                                {genError}
                            </p>
                        </div>
                    </div>
                {/if}

                <!-- Streamed modules -->
                {#if streamedModules.length > 0}
                    <div class="border-t border-border pt-3">
                        <p
                            class="text-xs font-semibold text-muted-foreground mb-2"
                        >
                            {streamProgress || ""}
                        </p>
                        <div class="flex flex-col gap-1.5">
                            {#each streamedModules as mod, mi}
                                {#if mi === streamedModules.length - 1}
                                    <div
                                        class="rounded-lg border border-success/20 bg-success/5 px-3 py-2"
                                    >
                                        <div
                                            class="flex items-center gap-2 mb-1"
                                        >
                                            <CheckCircle
                                                class="size-3 text-success shrink-0"
                                            />
                                            <p
                                                class="text-xs font-semibold text-foreground"
                                            >
                                                Module {mi + 1}: {mod.title}
                                            </p>
                                        </div>
                                        <p
                                            class="text-xs text-muted-foreground/80"
                                        >
                                            {mod.description}
                                        </p>
                                    </div>
                                {:else}
                                    <div
                                        class="flex items-center gap-1.5 text-xs text-muted-foreground/60"
                                    >
                                        <CheckCircle
                                            class="size-2.5 text-success/60 shrink-0"
                                        />
                                        <span class="truncate"
                                            >Module {mi + 1}: {mod.title}</span
                                        >
                                    </div>
                                {/if}
                            {/each}
                        </div>
                    </div>
                {/if}

                {#if !generationDone && !genError}
                    <div class="flex items-center gap-2 mt-3">
                        <LoaderCircle
                            class="size-3 text-muted-foreground animate-spin shrink-0"
                        />
                        <p class="text-xs text-muted-foreground animate-pulse">
                            {streamProgress || "Working..."}
                        </p>
                    </div>
                {/if}
            </div>
        {:else if !viewingCourse && !viewingLoading}
            <div
                class="rounded-xl border border-border bg-card p-12 text-center"
            >
                <Layers class="size-10 text-muted-foreground/40 mx-auto mb-3" />
                <p class="text-sm text-muted-foreground">
                    Select a course or generate a new one to view its structure.
                </p>
            </div>
        {:else if viewingLoading}
            <div class="flex items-center justify-center py-12">
                <LoaderCircle
                    class="size-6 text-muted-foreground animate-spin"
                />
            </div>
        {:else}
            <!-- Course header -->
            <div class="mb-6">
                <h1
                    class="text-2xl font-semibold tracking-tight text-foreground"
                >
                    {viewingCourse.title}
                </h1>
                <p class="mt-1.5 text-sm text-muted-foreground leading-relaxed">
                    {viewingCourse.description}
                </p>
                <div class="flex items-center gap-3 mt-3">
                    <span
                        class="inline-flex items-center gap-1 rounded-full border border-border px-2.5 py-0.5 text-xs font-medium {viewingCourse.status ===
                        'published'
                            ? 'bg-success/10 text-success border-success/20'
                            : viewingCourse.status === 'draft'
                              ? 'bg-warning/10 text-warning border-warning/20'
                              : 'bg-muted text-muted-foreground'}"
                    >
                        {#if viewingCourse.status === "published"}
                            <CheckCircle class="size-2.5" />
                        {:else if viewingCourse.status === "draft"}
                            <Clock class="size-2.5" />
                        {/if}
                        {viewingCourse.status}
                    </span>
                    <span class="text-xs text-muted-foreground">
                        {viewingCourse.modules?.length ?? 0} module(s)
                    </span>
                    {#if viewingCourse.sources?.length > 0}
                        <button
                            class="inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-medium transition-colors {showSources
                                ? 'bg-primary/10 text-primary border-primary/20'
                                : 'border-border text-muted-foreground hover:text-foreground hover:border-foreground/20'}"
                            onclick={() => (showSources = !showSources)}
                        >
                            <Bookmark class="size-3" />
                            {viewingCourse.sources.length} source(s)
                        </button>
                    {/if}

                    <a
                        href="/lesson-player?preview={viewingCourse.id}"
                        target="_blank"
                        rel="noopener noreferrer"
                        class="inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-medium transition-colors border-info/20 bg-info/5 text-info hover:bg-info/10"
                    >
                        <Play class="size-2.5" />
                        Preview as Student
                    </a>
                </div>

                {#if showSources && viewingCourse.sources}
                    <div
                        class="mt-4 rounded-lg border border-border bg-card p-4"
                    >
                        <p
                            class="text-xs font-semibold text-muted-foreground mb-2"
                        >
                            Source Documents
                        </p>
                        <div class="flex flex-col gap-1.5">
                            {#each viewingCourse.sources as src}
                                <div class="rounded-md bg-muted/20 px-3 py-2">
                                    <p class="text-xs font-medium">
                                        {src.document_title}
                                    </p>
                                    <p
                                        class="text-xs text-muted-foreground mt-0.5"
                                    >
                                        Chunk {src.chunk_index}
                                    </p>
                                    <p
                                        class="text-xs text-muted-foreground mt-1 italic"
                                    >
                                        "{src.excerpt}"
                                    </p>
                                </div>
                            {/each}
                        </div>
                    </div>
                {/if}
            </div>

            <!-- AI Edit bar -->
            <div
                class="rounded-lg border border-border bg-card p-3 mb-6 flex items-center gap-2"
            >
                <input
                    type="text"
                    bind:value={editInstructions}
                    placeholder="Ask AI to edit this course..."
                    class="flex-1 rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                    onkeydown={(e: KeyboardEvent) => {
                        if (e.key === "Enter" && !editing) {
                            handleEdit(viewingCourse.id);
                        }
                    }}
                />
                <Button.Root
                    size="sm"
                    disabled={editing || !editInstructions.trim()}
                    onclick={() => handleEdit(viewingCourse.id)}
                >
                    {#if editing}
                        <LoaderCircle class="size-3.5 mr-1.5 animate-spin" />
                        Editing…
                    {:else}
                        Edit
                    {/if}
                </Button.Root>
            </div>

            {#if editError}
                <div
                    class="rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 mb-6 flex items-center gap-2"
                >
                    <XCircle class="size-4 text-destructive shrink-0" />
                    <p class="text-xs text-destructive">
                        {editError}
                    </p>
                </div>
            {/if}

            <!-- Modules -->
            <div class="flex flex-col gap-3">
                {#each viewingCourse.modules ?? [] as mod, mi}
                    <div
                        class="rounded-xl border border-border bg-card overflow-hidden"
                    >
                        <button
                            class="w-full flex items-center gap-3 px-5 py-4 text-left hover:bg-muted/30 transition-colors"
                            onclick={() => toggleModule(mod.id)}
                        >
                            {#if expandedModules[mod.id]}
                                <ChevronDown
                                    class="size-4 text-muted-foreground shrink-0"
                                />
                            {:else}
                                <ChevronRight
                                    class="size-4 text-muted-foreground shrink-0"
                                />
                            {/if}
                            <div class="flex-1 min-w-0">
                                <div class="flex items-center gap-2">
                                    <span
                                        class="text-xs font-semibold text-muted-foreground"
                                    >
                                        Module {mi + 1}
                                    </span>
                                    <span class="text-xs text-muted-foreground">
                                        {mod.items?.length ?? 0} items
                                    </span>
                                </div>
                                <p
                                    class="text-sm font-semibold text-foreground"
                                >
                                    {mod.title}
                                </p>
                            </div>
                        </button>
                        {#if expandedModules[mod.id]}
                            <div class="border-t border-border px-5 pb-4 pt-3">
                                <p class="text-xs text-muted-foreground mb-3">
                                    {mod.description}
                                </p>
                                <div class="flex flex-col gap-4">
                                    {#each mod.items ?? [] as item, ii (item.id)}
                                        {@const d = itemData(item)}
                                        {#if item.item_type === "content"}
                                            <article
                                                class="rounded-2xl border border-border bg-card p-6 md:p-8 relative group/item"
                                            >
                                                <AiItemEditWrapper
                                                    {item}
                                                    {editingItemId}
                                                    {itemEditInstructions}
                                                    {itemEditLoading}
                                                    {itemEditError}
                                                    {itemEditPreview}
                                                    onStartEdit={startItemEdit}
                                                    onCancelEdit={cancelItemEdit}
                                                    onAiEdit={handleItemAiEdit}
                                                    onAccept={acceptItemEdit}
                                                    onDiscard={discardItemEdit}
                                                    onInstructionsInput={onItemInstructionsInput}
                                                    onDelete={deleteItem}
                                                />
                                                <span
                                                    class="text-[10px] font-semibold uppercase tracking-[0.18em] text-muted-foreground"
                                                >
                                                    Learning material
                                                </span>
                                                {#if d.body}
                                                    <Markdown
                                                        source={d.body}
                                                        class="mt-3 max-w-prose text-[15px]"
                                                    />
                                                {:else}
                                                    <p
                                                        class="mt-3 text-sm italic text-muted-foreground"
                                                    >
                                                        Empty content
                                                    </p>
                                                {/if}
                                            </article>
                                        {:else if item.item_type === "mc"}
                                            <div
                                                class="rounded-xl border border-border bg-card p-5 relative group/item"
                                            >
                                                <AiItemEditWrapper
                                                    {item}
                                                    {editingItemId}
                                                    {itemEditInstructions}
                                                    {itemEditLoading}
                                                    {itemEditError}
                                                    {itemEditPreview}
                                                    onStartEdit={startItemEdit}
                                                    onCancelEdit={cancelItemEdit}
                                                    onAiEdit={handleItemAiEdit}
                                                    onAccept={acceptItemEdit}
                                                    onDiscard={discardItemEdit}
                                                    onInstructionsInput={onItemInstructionsInput}
                                                    onDelete={deleteItem}
                                                />
                                                <p
                                                    class="text-[10px] font-semibold uppercase tracking-[0.18em] text-muted-foreground mb-2"
                                                >
                                                    Multiple Choice
                                                </p>
                                                <p
                                                    class="text-sm font-medium text-foreground mb-3"
                                                >
                                                    {ii + 1}. {d.question ?? ""}
                                                </p>
                                                <div
                                                    class="flex flex-col gap-2"
                                                >
                                                    {#each d.options ?? [] as opt, oi}
                                                        <div
                                                            class="flex items-center gap-2.5 rounded-lg border px-3.5 py-2.5 {oi ===
                                                            d.correct
                                                                ? 'border-primary bg-primary/5'
                                                                : 'border-border'}"
                                                        >
                                                            <div
                                                                class="size-4 rounded-full border-2 flex items-center justify-center shrink-0 {oi ===
                                                                d.correct
                                                                    ? 'border-primary'
                                                                    : 'border-muted-foreground/30'}"
                                                            >
                                                                {#if oi === d.correct}
                                                                    <div
                                                                        class="size-2 rounded-full bg-primary"
                                                                    ></div>
                                                                {/if}
                                                            </div>
                                                            <span
                                                                class="text-sm text-foreground"
                                                                >{opt}</span
                                                            >
                                                        </div>
                                                    {/each}
                                                </div>
                                                {#if !d.options?.length}
                                                    <p
                                                        class="text-sm italic text-muted-foreground"
                                                    >
                                                        Empty question
                                                    </p>
                                                {/if}
                                            </div>
                                        {:else if item.item_type === "ma"}
                                            <div
                                                class="rounded-xl border border-border bg-card p-5 relative group/item"
                                            >
                                                <AiItemEditWrapper
                                                    {item}
                                                    {editingItemId}
                                                    {itemEditInstructions}
                                                    {itemEditLoading}
                                                    {itemEditError}
                                                    {itemEditPreview}
                                                    onStartEdit={startItemEdit}
                                                    onCancelEdit={cancelItemEdit}
                                                    onAiEdit={handleItemAiEdit}
                                                    onAccept={acceptItemEdit}
                                                    onDiscard={discardItemEdit}
                                                    onInstructionsInput={onItemInstructionsInput}
                                                    onDelete={deleteItem}
                                                />
                                                <p
                                                    class="text-[10px] font-semibold uppercase tracking-[0.18em] text-muted-foreground mb-2"
                                                >
                                                    Multiple Answer
                                                </p>
                                                <p
                                                    class="text-sm font-medium text-foreground mb-3"
                                                >
                                                    {ii + 1}. {d.question ?? ""}
                                                    <span
                                                        class="text-xs text-muted-foreground"
                                                        >(select all that apply)</span
                                                    >
                                                </p>
                                                <div
                                                    class="flex flex-col gap-2"
                                                >
                                                    {#each d.options ?? [] as opt, oi}
                                                        <div
                                                            class="flex items-center gap-2.5 rounded-lg border px-3.5 py-2.5 {(
                                                                d.correct ?? []
                                                            ).includes(oi)
                                                                ? 'border-primary bg-primary/5'
                                                                : 'border-border'}"
                                                        >
                                                            <div
                                                                class="size-4 rounded border-2 flex items-center justify-center shrink-0 {(
                                                                    d.correct ??
                                                                    []
                                                                ).includes(oi)
                                                                    ? 'bg-primary border-primary text-primary-foreground'
                                                                    : 'border-muted-foreground/30'}"
                                                            >
                                                                {#if (d.correct ?? []).includes(oi)}
                                                                    <CheckCircle
                                                                        class="size-3"
                                                                    />
                                                                {/if}
                                                            </div>
                                                            <span
                                                                class="text-sm text-foreground"
                                                                >{opt}</span
                                                            >
                                                        </div>
                                                    {/each}
                                                </div>
                                                {#if !d.options?.length}
                                                    <p
                                                        class="text-sm italic text-muted-foreground"
                                                    >
                                                        Empty question
                                                    </p>
                                                {/if}
                                            </div>
                                        {:else if item.item_type === "tf"}
                                            <div
                                                class="rounded-xl border border-border bg-card p-5 relative group/item"
                                            >
                                                <AiItemEditWrapper
                                                    {item}
                                                    {editingItemId}
                                                    {itemEditInstructions}
                                                    {itemEditLoading}
                                                    {itemEditError}
                                                    {itemEditPreview}
                                                    onStartEdit={startItemEdit}
                                                    onCancelEdit={cancelItemEdit}
                                                    onAiEdit={handleItemAiEdit}
                                                    onAccept={acceptItemEdit}
                                                    onDiscard={discardItemEdit}
                                                    onInstructionsInput={onItemInstructionsInput}
                                                    onDelete={deleteItem}
                                                />
                                                <p
                                                    class="text-[10px] font-semibold uppercase tracking-[0.18em] text-muted-foreground mb-2"
                                                >
                                                    True / False
                                                </p>
                                                <p
                                                    class="text-sm font-medium text-foreground mb-3"
                                                >
                                                    {ii + 1}. {d.statement ??
                                                        ""}
                                                </p>
                                                <div class="flex gap-3">
                                                    {#each [true, false] as val}
                                                        <div
                                                            class="flex-1 rounded-lg border px-4 py-2.5 text-sm font-medium {val ===
                                                            d.answer
                                                                ? 'border-primary bg-primary/5 text-primary'
                                                                : 'border-border text-muted-foreground'}"
                                                        >
                                                            {val
                                                                ? "True"
                                                                : "False"}
                                                        </div>
                                                    {/each}
                                                </div>
                                            </div>
                                        {:else if item.item_type === "fb"}
                                            <div
                                                class="rounded-xl border border-border bg-card p-5 relative group/item"
                                            >
                                                <AiItemEditWrapper
                                                    {item}
                                                    {editingItemId}
                                                    {itemEditInstructions}
                                                    {itemEditLoading}
                                                    {itemEditError}
                                                    {itemEditPreview}
                                                    onStartEdit={startItemEdit}
                                                    onCancelEdit={cancelItemEdit}
                                                    onAiEdit={handleItemAiEdit}
                                                    onAccept={acceptItemEdit}
                                                    onDiscard={discardItemEdit}
                                                    onInstructionsInput={onItemInstructionsInput}
                                                    onDelete={deleteItem}
                                                />
                                                <p
                                                    class="text-[10px] font-semibold uppercase tracking-[0.18em] text-muted-foreground mb-2"
                                                >
                                                    Fill in the Blanks
                                                </p>
                                                <p
                                                    class="text-sm font-medium text-foreground mb-3"
                                                >
                                                    {ii + 1}. Fill in the
                                                    blanks:
                                                </p>
                                                <p
                                                    class="text-sm text-foreground leading-relaxed mb-3 whitespace-pre-line"
                                                >
                                                    {d.text ?? ""}
                                                </p>
                                                {#if d.blanks?.length}
                                                    <div
                                                        class="flex flex-col gap-1.5"
                                                    >
                                                        {#each d.blanks as blank, bi}
                                                            <p
                                                                class="text-xs text-muted-foreground"
                                                            >
                                                                <span
                                                                    class="font-medium text-foreground"
                                                                    >{bi +
                                                                        1}.</span
                                                                >
                                                                {blank}
                                                            </p>
                                                        {/each}
                                                    </div>
                                                {/if}
                                            </div>
                                        {:else if item.item_type === "sa"}
                                            <div
                                                class="rounded-xl border border-border bg-card p-5 relative group/item"
                                            >
                                                <AiItemEditWrapper
                                                    {item}
                                                    {editingItemId}
                                                    {itemEditInstructions}
                                                    {itemEditLoading}
                                                    {itemEditError}
                                                    {itemEditPreview}
                                                    onStartEdit={startItemEdit}
                                                    onCancelEdit={cancelItemEdit}
                                                    onAiEdit={handleItemAiEdit}
                                                    onAccept={acceptItemEdit}
                                                    onDiscard={discardItemEdit}
                                                    onInstructionsInput={onItemInstructionsInput}
                                                    onDelete={deleteItem}
                                                />
                                                <p
                                                    class="text-[10px] font-semibold uppercase tracking-[0.18em] text-muted-foreground mb-2"
                                                >
                                                    Short Answer
                                                </p>
                                                <p
                                                    class="text-sm font-medium text-foreground mb-3"
                                                >
                                                    {ii + 1}. {d.question ?? ""}
                                                </p>
                                                {#if d.sample_answer}
                                                    <p
                                                        class="mt-2 text-xs text-muted-foreground"
                                                    >
                                                        Sample answer: {d.sample_answer}
                                                    </p>
                                                {/if}
                                            </div>
                                        {:else if item.item_type === "matching"}
                                            <div
                                                class="rounded-xl border border-border bg-card p-5 relative group/item"
                                            >
                                                <AiItemEditWrapper
                                                    {item}
                                                    {editingItemId}
                                                    {itemEditInstructions}
                                                    {itemEditLoading}
                                                    {itemEditError}
                                                    {itemEditPreview}
                                                    onStartEdit={startItemEdit}
                                                    onCancelEdit={cancelItemEdit}
                                                    onAiEdit={handleItemAiEdit}
                                                    onAccept={acceptItemEdit}
                                                    onDiscard={discardItemEdit}
                                                    onInstructionsInput={onItemInstructionsInput}
                                                    onDelete={deleteItem}
                                                />
                                                <p
                                                    class="text-[10px] font-semibold uppercase tracking-[0.18em] text-muted-foreground mb-2"
                                                >
                                                    Matching
                                                </p>
                                                <p
                                                    class="mb-3 text-sm font-medium text-foreground"
                                                >
                                                    {ii + 1}. {d.question ??
                                                        "Match each item with its definition."}
                                                </p>
                                                <div
                                                    class="flex flex-col gap-1.5"
                                                >
                                                    {#each d.pairs ?? [] as pair}
                                                        <p
                                                            class="text-sm text-foreground"
                                                        >
                                                            <span
                                                                class="font-medium"
                                                                >{pair.left}</span
                                                            >
                                                            <span
                                                                class="text-muted-foreground"
                                                            >
                                                                — {pair.right}</span
                                                            >
                                                        </p>
                                                    {/each}
                                                </div>
                                                {#if !d.pairs?.length}
                                                    <p
                                                        class="text-sm italic text-muted-foreground"
                                                    >
                                                        Empty
                                                    </p>
                                                {/if}
                                            </div>
                                        {:else if item.item_type === "drag_sort"}
                                            <div
                                                class="rounded-xl border border-border bg-card p-5 relative group/item"
                                            >
                                                <AiItemEditWrapper
                                                    {item}
                                                    {editingItemId}
                                                    {itemEditInstructions}
                                                    {itemEditLoading}
                                                    {itemEditError}
                                                    {itemEditPreview}
                                                    onStartEdit={startItemEdit}
                                                    onCancelEdit={cancelItemEdit}
                                                    onAiEdit={handleItemAiEdit}
                                                    onAccept={acceptItemEdit}
                                                    onDiscard={discardItemEdit}
                                                    onInstructionsInput={onItemInstructionsInput}
                                                    onDelete={deleteItem}
                                                />
                                                <p
                                                    class="text-[10px] font-semibold uppercase tracking-[0.18em] text-muted-foreground mb-2"
                                                >
                                                    Drag &amp; Sort
                                                </p>
                                                <p
                                                    class="mb-3 text-sm font-medium text-foreground"
                                                >
                                                    {ii + 1}. {d.question ??
                                                        "Put these items in the correct order."}
                                                </p>
                                                <ol
                                                    class="flex flex-col gap-1.5 ml-5 list-decimal marker:text-muted-foreground"
                                                >
                                                    {#each d.items ?? [] as it}
                                                        <li
                                                            class="text-sm text-foreground"
                                                        >
                                                            {it}
                                                        </li>
                                                    {/each}
                                                </ol>
                                                {#if !d.items?.length}
                                                    <p
                                                        class="text-sm italic text-muted-foreground"
                                                    >
                                                        Empty
                                                    </p>
                                                {/if}
                                            </div>
                                        {:else if item.item_type === "sequence"}
                                            <div
                                                class="rounded-xl border border-border bg-card p-5 relative group/item"
                                            >
                                                <AiItemEditWrapper
                                                    {item}
                                                    {editingItemId}
                                                    {itemEditInstructions}
                                                    {itemEditLoading}
                                                    {itemEditError}
                                                    {itemEditPreview}
                                                    onStartEdit={startItemEdit}
                                                    onCancelEdit={cancelItemEdit}
                                                    onAiEdit={handleItemAiEdit}
                                                    onAccept={acceptItemEdit}
                                                    onDiscard={discardItemEdit}
                                                    onInstructionsInput={onItemInstructionsInput}
                                                    onDelete={deleteItem}
                                                />
                                                <p
                                                    class="text-[10px] font-semibold uppercase tracking-[0.18em] text-muted-foreground mb-2"
                                                >
                                                    Sequence
                                                </p>
                                                <p
                                                    class="mb-3 text-sm font-medium text-foreground"
                                                >
                                                    {ii + 1}. {d.question ??
                                                        "Arrange the steps in order."}
                                                </p>
                                                <ol
                                                    class="flex flex-col gap-1.5 ml-5 list-decimal marker:text-muted-foreground"
                                                >
                                                    {#each d.steps ?? [] as step}
                                                        <li
                                                            class="text-sm text-foreground"
                                                        >
                                                            {step}
                                                        </li>
                                                    {/each}
                                                </ol>
                                                {#if !d.steps?.length}
                                                    <p
                                                        class="text-sm italic text-muted-foreground"
                                                    >
                                                        Empty
                                                    </p>
                                                {/if}
                                            </div>
                                        {:else}
                                            <div
                                                class="rounded-xl border border-border bg-card p-5 relative group/item"
                                            >
                                                <AiItemEditWrapper
                                                    {item}
                                                    {editingItemId}
                                                    {itemEditInstructions}
                                                    {itemEditLoading}
                                                    {itemEditError}
                                                    {itemEditPreview}
                                                    onStartEdit={startItemEdit}
                                                    onCancelEdit={cancelItemEdit}
                                                    onAiEdit={handleItemAiEdit}
                                                    onAccept={acceptItemEdit}
                                                    onDiscard={discardItemEdit}
                                                    onInstructionsInput={onItemInstructionsInput}
                                                    onDelete={deleteItem}
                                                />
                                                <p
                                                    class="text-[10px] font-semibold uppercase tracking-[0.18em] text-muted-foreground mb-2"
                                                >
                                                    {item.item_type}
                                                </p>
                                                {#if itemPreview(item)}
                                                    <p
                                                        class="text-sm text-foreground"
                                                    >
                                                        {itemPreview(item)}
                                                    </p>
                                                {:else}
                                                    <p
                                                        class="text-sm italic text-muted-foreground"
                                                    >
                                                        Empty
                                                    </p>
                                                {/if}
                                            </div>
                                        {/if}
                                    {/each}
                                </div>
                            </div>
                        {/if}
                    </div>
                {/each}
            </div>
        {/if}
    </section>
</div>
