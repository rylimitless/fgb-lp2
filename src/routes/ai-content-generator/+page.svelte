<script lang="ts">
    import {
        Sparkles,
        BookOpen,
        LoaderCircle,
        ChevronDown,
        ChevronRight,
        FileText,
        CheckCircle,
        XCircle,
        Clock,
        ArrowRight,
        Layers,
        Bookmark,
        ExternalLink,
    } from "@lucide/svelte";
    import * as Button from "$lib/components/ui/button";

    let { data } = $props();

    let courses = $state<any[]>([]);
    let approvedDocs = $state<any[]>([]);
    $effect(() => {
        courses = data.courses ?? [];
        approvedDocs = (data.documents ?? []).filter(
            (d: any) => d.approved && d.status === "ready",
        );
    });

    // ---- Form state ----
    let title = $state("");
    let description = $state("");
    let selectedDocIds = $state<number[]>([]);
    let generating = $state(false);
    let genError = $state("");

    // ---- Generation progress (SSE streaming) ----
    let generationSteps = $state<{ step: string; detail: string }[]>([]);
    let generationDone = $state(false);

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
    let editError = $state("");

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
        viewingCourse = null;
        generating = true;

        try {
            const res = await fetch(
                "/api/courses/generate",
                {
                    method: "POST",
                    headers: { "Content-Type": "application/json" },
                    credentials: "include",
                    body: JSON.stringify({
                        title: title || "Untitled Course",
                        description,
                        source_doc_ids: selectedDocIds,
                    }),
                },
            );

            if (!res.ok) {
                genError = "Generation failed";
                return;
            }

            const reader = res.body?.getReader();
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
                        const payload = JSON.parse(line.slice(6));
                        if (eventType === "step") {
                            generationSteps = [...generationSteps, payload];
                        } else if (eventType === "error") {
                            genError = payload.message || "Error";
                            return;
                        } else if (eventType === "done") {
                            generationDone = true;
                            viewingCourse = payload;
                            selectedCourseId = payload.id;
                            expandedModules = {};
                            courses = [payload, ...courses];
                        }
                        eventType = "";
                    }
                }
            }
        } catch {
            if (!genError) genError = "Network error. Is the backend running?";
        } finally {
            generating = false;
        }
    }

    async function handleEdit(courseId: number) {
        if (!editInstructions.trim()) return;
        editError = "";
        editing = true;
        try {
            const res = await fetch(
                `/api/courses/${courseId}/edit`,
                {
                    method: "PUT",
                    headers: { "Content-Type": "application/json" },
                    credentials: "include",
                    body: JSON.stringify({ instructions: editInstructions }),
                },
            );
            if (!res.ok) {
                const err = await res.json();
                editError = err.error || "Edit failed";
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

    async function viewCourseDetail(course: any) {
        selectedCourseId = course.id;
        viewingLoading = true;
        expandedModules = {};
        showSources = false;

        try {
            const res = await fetch(
                `/api/courses/${course.id}`,
                { credentials: "include" },
            );
            if (res.ok) {
                viewingCourse = await res.json();
            }
        } catch {
            // ignore
        } finally {
            viewingLoading = false;
        }
    }

    function itemTypeLabel(type: string): string {
        const labels: Record<string, string> = {
            content: "Content",
            mc: "Multiple Choice",
            ma: "Multiple Answer",
            tf: "True / False",
            fb: "Fill in the Blank",
            sa: "Short Answer",
            matching: "Matching",
            drag_sort: "Drag & Sort",
            hotspot: "Hotspot",
            sequence: "Sequence",
            scale: "Scale",
        };
        return labels[type] ?? type;
    }

    function itemTypeColor(type: string): string {
        const colors: Record<string, string> = {
            content: "bg-gray-500/10 text-gray-600 border-gray-500/20",
            mc: "bg-blue-500/10 text-blue-500 border-blue-500/20",
            ma: "bg-indigo-500/10 text-indigo-500 border-indigo-500/20",
            tf: "bg-amber-500/10 text-amber-500 border-amber-500/20",
            fb: "bg-emerald-500/10 text-emerald-500 border-emerald-500/20",
            sa: "bg-violet-500/10 text-violet-500 border-violet-500/20",
            matching: "bg-rose-500/10 text-rose-500 border-rose-500/20",
            drag_sort: "bg-cyan-500/10 text-cyan-500 border-cyan-500/20",
            hotspot: "bg-orange-500/10 text-orange-500 border-orange-500/20",
            sequence: "bg-teal-500/10 text-teal-500 border-teal-500/20",
            scale: "bg-pink-500/10 text-pink-500 border-pink-500/20",
        };
        return colors[type] ?? "bg-muted text-muted-foreground border-border";
    }

    function itemPreview(item: any): string {
        const d = item.data;
        if (!d) return "";
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
                return `${(d.items ?? []).length} items to sort`;
            case "sequence":
                return `${(d.steps ?? []).length} steps`;
            case "scale":
                return d.question ?? "";
            default:
                return "";
        }
    }
</script>

<div class="flex w-full max-w-7xl mx-auto gap-6">
    <!-- Left: Generator -->
    <section class="w-[400px] shrink-0 flex flex-col gap-4">
        <!-- Create Form -->
        <div class="rounded-xl border border-border bg-card p-6">
            <h2
                class="text-base font-semibold tracking-tight text-foreground mb-4 flex items-center gap-2"
            >
                <Sparkles class="size-4 text-primary" />
                Generate Course
            </h2>

            <div class="flex flex-col gap-4">
                <div>
                    <label
                        for="gen-title"
                        class="block text-sm font-medium text-foreground mb-1.5"
                    >
                        Course Title
                    </label>
                    <input
                        id="gen-title"
                        type="text"
                        bind:value={title}
                        placeholder="Workplace Safety Fundamentals"
                        class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                    />
                </div>

                <div>
                    <label
                        for="gen-desc"
                        class="block text-sm font-medium text-foreground mb-1.5"
                    >
                        What should this course cover?
                    </label>
                    <textarea
                        id="gen-desc"
                        bind:value={description}
                        rows={4}
                        placeholder="Describe the learning objectives, topics, and any specific areas to focus on..."
                        class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring resize-none"
                    ></textarea>
                </div>

                <!-- Source documents -->
                {#if approvedDocs.length > 0}
                    <div>
                        <label
                            class="block text-sm font-medium text-foreground mb-1.5"
                        >
                            Source Documents
                            <span class="text-muted-foreground font-normal">
                                (optional — select specific docs to pull from)
                            </span>
                        </label>
                        <div
                            class="flex flex-col gap-1 max-h-32 overflow-y-auto rounded-lg border border-border bg-muted/30 p-2"
                        >
                            {#each approvedDocs as doc}
                                <button
                                    class="flex items-center gap-2 rounded-md px-2.5 py-1.5 text-xs text-left transition-colors {selectedDocIds.includes(
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
                                    <span class="truncate">{doc.title}</span>
                                </button>
                            {/each}
                        </div>
                    </div>
                {/if}

                {#if genError}
                    <div
                        class="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 flex items-center gap-2"
                    >
                        <XCircle class="size-4 text-red-500 shrink-0" />
                        <p class="text-xs text-red-600 dark:text-red-400">
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
                                <p class="text-[10px] text-muted-foreground">
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
        {#if generating && generationSteps.length > 0}
            <!-- Generation Progress -->
            <div class="rounded-xl border border-border bg-card p-8">
                <h3
                    class="text-sm font-semibold text-foreground mb-6 flex items-center gap-2"
                >
                    <Sparkles class="size-4 text-primary" />
                    Building your course
                </h3>
                <div class="flex flex-col gap-3">
                    {#each generationSteps as step, i}
                        <div class="flex items-center gap-3">
                            <div
                                class="size-5 rounded-full {i <
                                    generationSteps.length - 1 || generationDone
                                    ? 'bg-emerald-500/10 text-emerald-500'
                                    : 'bg-blue-500/10 text-blue-500'}
                                flex items-center justify-center shrink-0"
                            >
                                {#if i < generationSteps.length - 1 || generationDone}
                                    <CheckCircle class="size-3" />
                                {:else}
                                    <LoaderCircle class="size-3 animate-spin" />
                                {/if}
                            </div>
                            <p class="text-sm text-muted-foreground">
                                {step.detail}
                            </p>
                        </div>
                    {/each}
                    {#if !generationDone}
                        <div class="flex items-center gap-3">
                            <div
                                class="size-5 rounded-full bg-muted flex items-center justify-center shrink-0"
                            >
                                <LoaderCircle
                                    class="size-3 text-muted-foreground animate-spin"
                                />
                            </div>
                            <p
                                class="text-sm text-muted-foreground animate-pulse"
                            >
                                Working...
                            </p>
                        </div>
                    {/if}
                </div>
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
                        class="inline-flex items-center gap-1 rounded-full border border-border px-2.5 py-0.5 text-[11px] font-medium {viewingCourse.status ===
                        'published'
                            ? 'bg-emerald-500/10 text-emerald-500 border-emerald-500/20'
                            : viewingCourse.status === 'draft'
                              ? 'bg-amber-500/10 text-amber-500 border-amber-500/20'
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
                            class="inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-[11px] font-medium transition-colors {showSources
                                ? 'bg-primary/10 text-primary border-primary/20'
                                : 'border-border text-muted-foreground hover:text-foreground hover:border-foreground/20'}"
                            onclick={() => (showSources = !showSources)}
                        >
                            <Bookmark class="size-3" />
                            {viewingCourse.sources.length} source(s)
                        </button>
                    {/if}
                </div>

                {#if showSources && viewingCourse.sources?.length > 0}
                    <div
                        class="mt-4 rounded-xl border border-border bg-muted/20 overflow-hidden"
                    >
                        <div
                            class="px-4 py-3 border-b border-border bg-muted/30 flex items-center gap-2"
                        >
                            <Bookmark class="size-3.5 text-primary" />
                            <span class="text-xs font-semibold text-foreground"
                                >Sources used to generate this course</span
                            >
                        </div>
                        <div
                            class="divide-y divide-border max-h-64 overflow-y-auto"
                        >
                            {#each viewingCourse.sources as src, si}
                                <div
                                    class="px-4 py-3 hover:bg-muted/20 transition-colors"
                                >
                                    <div class="flex items-start gap-2.5">
                                        <span
                                            class="text-[10px] font-semibold text-muted-foreground mt-0.5 shrink-0"
                                            >#{si + 1}</span
                                        >
                                        <div class="min-w-0">
                                            <p
                                                class="text-xs font-medium text-foreground truncate"
                                            >
                                                {src.document_title}
                                            </p>
                                            <p
                                                class="text-[11px] text-muted-foreground mt-1 leading-relaxed line-clamp-3"
                                            >
                                                {src.excerpt}
                                            </p>
                                        </div>
                                    </div>
                                </div>
                            {/each}
                        </div>
                    </div>
                {/if}
            </div>

            <!-- AI Edit -->
            <div class="mt-4 flex gap-2">
                <input
                    type="text"
                    bind:value={editInstructions}
                    placeholder="Ask AI to edit this course — e.g. 'add drag-and-drop questions' or 'add a module on safety procedures'..."
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
                        Editing...
                    {:else}
                        <Sparkles class="size-3.5 mr-1.5" />
                        Edit with AI
                    {/if}
                </Button.Root>
            </div>
            {#if editError}
                <p class="text-xs text-red-500 mt-1.5">{editError}</p>
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
                            <div
                                class="size-7 rounded-full bg-primary/10 flex items-center justify-center shrink-0"
                            >
                                <span class="text-xs font-bold text-primary">
                                    {mi + 1}
                                </span>
                            </div>
                            <div class="min-w-0 flex-1">
                                <p
                                    class="text-sm font-semibold text-foreground"
                                >
                                    {mod.title}
                                </p>
                                <p class="text-xs text-muted-foreground mt-0.5">
                                    {mod.description}
                                </p>
                            </div>
                            <div class="flex items-center gap-2 shrink-0">
                                <span class="text-[11px] text-muted-foreground">
                                    {mod.items?.length ?? 0} items
                                </span>
                                {#if expandedModules[mod.id]}
                                    <ChevronDown
                                        class="size-4 text-muted-foreground"
                                    />
                                {:else}
                                    <ChevronRight
                                        class="size-4 text-muted-foreground"
                                    />
                                {/if}
                            </div>
                        </button>

                        {#if expandedModules[mod.id]}
                            <div class="border-t border-border px-5 pb-4 pt-3">
                                <div class="flex flex-col gap-2">
                                    {#each mod.items ?? [] as item, ii}
                                        {#if item.item_type === "content"}
                                            <div
                                                class="rounded-lg bg-muted/10 px-4 py-3"
                                            >
                                                <p
                                                    class="text-[10px] font-semibold uppercase text-muted-foreground mb-2"
                                                >
                                                    Learning Material
                                                </p>
                                                <div
                                                    class="text-sm text-foreground leading-relaxed whitespace-pre-line"
                                                >
                                                    {itemPreview(item)}
                                                </div>
                                            </div>
                                        {:else}
                                            <div
                                                class="flex items-start gap-3 rounded-lg border border-border bg-muted/20 px-3.5 py-2.5"
                                            >
                                                <span
                                                    class="inline-flex items-center rounded-md border px-1.5 py-0.5 text-[10px] font-semibold uppercase shrink-0 {itemTypeColor(
                                                        item.item_type,
                                                    )}"
                                                >
                                                    {item.item_type}
                                                </span>
                                                <div class="min-w-0 flex-1">
                                                    <p
                                                        class="text-xs text-muted-foreground mb-0.5"
                                                    >
                                                        {itemTypeLabel(
                                                            item.item_type,
                                                        )}
                                                    </p>
                                                    <p
                                                        class="text-sm text-foreground leading-relaxed"
                                                    >
                                                        {itemPreview(item) ||
                                                            "(no content)"}
                                                    </p>
                                                </div>
                                                <span
                                                    class="text-[10px] text-muted-foreground/50 shrink-0 mt-0.5"
                                                >
                                                    #{ii + 1}
                                                </span>
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
