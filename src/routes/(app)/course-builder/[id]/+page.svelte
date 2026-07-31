<script lang="ts">
    import {
        Wand2,
        BookOpen,
        LoaderCircle,
        ChevronDown,
        ChevronRight,
        CheckCircle,
        XCircle,
        Clock,
        Plus,
        Trash2,
        RefreshCw,
        Sparkles,
        Pencil,
        X,
        Play,
        CircleAlert,
        Save,
        ArrowLeft,
        AlertTriangle,
        ArrowUp,
        ArrowDown,
        Square,
        Eye,
        EyeOff,
        Send,
        Rocket,
        CheckCircle2,
        ChevronsDownUp,
        ChevronsUpDown,
    } from "@lucide/svelte";
    import * as Button from "$lib/components/ui/button";
    import {
        DropdownMenu,
        DropdownMenuTrigger,
        DropdownMenuContent,
        DropdownMenuCheckboxGroup,
        DropdownMenuCheckboxItem,
        DropdownMenuLabel,
    } from "$lib/components/ui/dropdown-menu";
    import { PageHeader } from "$lib/components/brand";
    import Markdown from "$lib/components/brand/Markdown.svelte";
    import AiItemEditWrapper from "$lib/components/brand/AiItemEditWrapper.svelte";
    import ModulePreview from "$lib/components/brand/ModulePreview.svelte";
    import { uniqueConceptCount } from "$lib/concepts";
    import { goto } from "$app/navigation";

    let { data } = $props();

    // ---- Course state ----
    // Initialized from the server-loaded data, then updated locally as the
    // user edits/generates. The initial empty skeleton keeps the template
    // safe before the first $effect tick populates it from `data.course`.
    let course = $state<any>({ id: 0, title: "", description: "", modules: [] });
    $effect(() => {
        // Sync whenever the server-loaded course changes (initial load or
        // navigation between course ids). Local edits/generations write to
        // `course` directly via refresh(); we don't want those to be clobbered
        // by every `data` prop tick.
        if (data.course?.id !== course?.id) {
            course = data.course;
        }
    });

    // Refresh course state from server (after job completion, edits, etc).
    async function refresh() {
        try {
            const res = await fetch(`/api/courses/${course.id}`, {
                credentials: "include",
            });
            if (res.ok) {
                course = await res.json();
            }
        } catch {
            /* transient */
        }
    }

    // ---- Expanded modules ----
    let expandedModules = $state<Record<number, boolean>>({});
    function toggleModule(id: number) {
        expandedModules[id] = !expandedModules[id];
    }

    function collapseAll() {
        expandedModules = {};
    }

    function expandAll() {
        const all: Record<number, boolean> = {};
        for (const m of course.modules ?? []) {
            all[m.id] = true;
        }
        expandedModules = all;
    }

    let anyExpanded = $derived(
        Object.values(expandedModules).some((v) => v),
    );

    // ---- Multi-select for bulk delete ----
    // selectedModules is keyed by module id. Kept as a plain object so Svelte's
    // deep reactivity picks up per-key toggles without reassignment.
    let selectedModules = $state<Record<number, boolean>>({});
    let selectedIds = $derived(
        (course.modules ?? [])
            .filter((m: any) => selectedModules[m.id])
            .map((m: any) => m.id),
    );
    let selectedCount = $derived(selectedIds.length);
    let moduleCount = $derived(course.modules?.length ?? 0);
    // "some" = indeterminate checkbox state, "all" = fully checked.
    let allSelected = $derived(
        moduleCount > 0 && selectedCount === moduleCount,
    );
    let someSelected = $derived(selectedCount > 0 && !allSelected);
    let bulkDeleting = $state(false);
    let bulkDeleteError = $state<string | null>(null);

    function toggleSelect(id: number) {
        if (selectedModules[id]) {
            delete selectedModules[id];
        } else {
            selectedModules[id] = true;
        }
        bulkDeleteError = null;
    }

    function toggleSelectAll() {
        if (allSelected) {
            selectedModules = {};
        } else {
            const all: Record<number, boolean> = {};
            for (const m of course.modules ?? []) {
                all[m.id] = true;
            }
            selectedModules = all;
        }
        bulkDeleteError = null;
    }

    function clearSelection() {
        selectedModules = {};
    }

    // ---- Learner preview modal ----
    // Opens a full-screen preview of a module as a learner would see it —
    // clean content rendering, interactive questions with check/reveal.
    let previewingModule = $state<any>(null);

    // ---- Per-module generation status (by module id) ----
    // Tracks the active job id (for SSE stream) and human-readable progress
    // text for the row's status pill.
    type ModuleJobState = {
        jobId: string;
        steps: string[];
        abort: AbortController | null;
    };
    let moduleJobs = $state<Record<number, ModuleJobState>>({});

    // Module-level errors that don't rise to a full job failure
    let moduleErrors = $state<Record<number, string>>({});

    // ---- Inline module editing ----
    let editingModuleId = $state<number | null>(null);
    let editBuf = $state<{
        title: string;
        description: string;
        questionTypes: string[];
    }>({ title: "", description: "", questionTypes: [] });
    let savingModule = $state(false);

    // ---- Add-module form ----
    let addingModule = $state(false);
    let newModuleTitle = $state("");
    let newModuleDesc = $state("");

    // ---- Course header inline edit (title + description) ----
    let editingCourse = $state(false);
    let courseEditBuf = $state({ title: "", description: "" });
    let savingCourse = $state(false);
    let courseEditError = $state("");

    function startEditCourse() {
        courseEditBuf = {
            title: course.title ?? "",
            description: course.description ?? "",
        };
        editingCourse = true;
        courseEditError = "";
    }

    function cancelEditCourse() {
        editingCourse = false;
        courseEditError = "";
    }

    async function saveEditCourse() {
        if (!courseEditBuf.title.trim()) {
            courseEditError = "Title is required";
            return;
        }
        savingCourse = true;
        try {
            const res = await fetch(`/api/courses/${course.id}/meta`, {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                credentials: "include",
                body: JSON.stringify({
                    title: courseEditBuf.title.trim(),
                    description: courseEditBuf.description.trim(),
                }),
            });
            if (!res.ok) {
                const err = await res.json().catch(() => ({}));
                courseEditError = err.error || "Failed to save";
                return;
            }
            editingCourse = false;
            await refresh();
        } catch {
            courseEditError = "Network error";
        } finally {
            savingCourse = false;
        }
    }

    // ---- Module reordering ----
    // Backend takes an ordered list of module IDs; we send the full current
    // order with two adjacent entries swapped. Generating modules are pinned
    // (server rejects reorders that move them) to avoid races with the worker.
    let reordering = $state(false);
    let reorderError = $state("");

    async function moveModule(index: number, direction: -1 | 1) {
        const mods = course.modules ?? [];
        const target = index + direction;
        if (target < 0 || target >= mods.length) return;
        // Optimistic swap for snappy UI.
        const swapped = [...mods];
        [swapped[index], swapped[target]] = [swapped[target], swapped[index]];
        course = { ...course, modules: swapped };
        reorderError = "";
        reordering = true;
        try {
            const res = await fetch(
                `/api/courses/${course.id}/modules/reorder`,
                {
                    method: "PUT",
                    headers: { "Content-Type": "application/json" },
                    credentials: "include",
                    body: JSON.stringify({
                        module_ids: swapped.map((m: any) => m.id),
                    }),
                },
            );
            if (!res.ok) {
                const err = await res.json().catch(() => ({}));
                reorderError = err.error || "Failed to reorder";
                // Revert on failure.
                await refresh();
            }
        } catch {
            reorderError = "Network error";
            await refresh();
        } finally {
            reordering = false;
        }
    }

    // ---- Generate all pending modules ----
    // Runs sequentially through every pending/failed module, reusing the
    // per-module SSE flow. Sequential keeps LLM rate usage bounded; the user
    // can stop mid-way and the modules already done stay done.
    let generatingAll = $state(false);
    let generateAllTotal = $state(0);
    let generateAllDone = $state(0);
    let generateAllAbort = $state(false);

    async function generateAllPending() {
        if (generatingAll) return;
        // Snapshot the candidate module IDs up front — `course` will mutate
        // as each module flips status, and we don't want to pick up modules
        // that become ready mid-loop or skip ones that flip to failed.
        const candidates = (course.modules ?? [])
            .filter((m: any) => m.status === "pending" || m.status === "failed")
            .map((m: any) => m.id);
        if (candidates.length === 0) return;

        generatingAll = true;
        generateAllAbort = false;
        generateAllTotal = candidates.length;
        generateAllDone = 0;

        for (const moduleId of candidates) {
            if (generateAllAbort) break;
            await generateModule(moduleId);
            generateAllDone++;
        }

        generatingAll = false;
        generateAllAbort = false;
    }

    function stopGenerateAll() {
        generateAllAbort = true;
        // Also cancel the in-flight per-module job, if any.
        for (const id of Object.keys(moduleJobs)) {
            const numId = Number(id);
            if (!Number.isNaN(numId)) cancelModuleGeneration(numId);
        }
    }

    // ---- Outline revision ("ask AI to revise the whole outline") ----
    // Triggers a destructive operation: every module + item on the course is
    // wiped and replaced with the AI's revised structure. The prompt keeps
    // what already works, so this is safe to use even after some modules
    // are generated — but those generated modules are lost.
    let revisingOutline = $state(false);
    let reviseFeedback = $state("");
    let reviseError = $state("");
    let reviseSteps = $state<string[]>([]);
    let reviseAbort = $state<AbortController | null>(null);
    let showRevisePanel = $state(false);

    // ---- Per-item AI Edit (mirror of ai-content-generator pattern) ----
    let editingItemId = $state<number | null>(null);
    let itemEditInstructions = $state("");
    let itemEditLoading = $state(false);
    let itemEditError = $state("");
    let itemEditPreview = $state<any>(null);
    let itemEditAcceptedIds = $state<Set<number>>(new Set());

    // ---- Status helpers ----
    function statusPill(status: string) {
        switch (status) {
            case "generating":
                return { label: "Generating", class: "bg-info/15 text-info", Icon: LoaderCircle };
            case "ready":
                return { label: "Ready", class: "bg-success/15 text-success", Icon: CheckCircle };
            case "failed":
                return { label: "Failed", class: "bg-destructive/15 text-destructive", Icon: XCircle };
            case "pending":
            default:
                return { label: "Pending", class: "bg-muted text-muted-foreground", Icon: Clock };
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

    // Human-readable labels for question types, shown on item cards and badges.
    const typeLabels: Record<string, string> = {
        mc: "Multiple choice",
        ma: "Multiple answer",
        tf: "True / false",
        fb: "Fill-in-the-blank",
        sa: "Short answer",
        matching: "Matching",
        drag_sort: "Drag and sort",
        hotspot: "Hotspot",
    };

    // variantInfo reports where this item sits among its concept's type-variants.
    // Items generated by the adaptive pipeline share a question_group_id; this
    // lets the author see e.g. "variant 2 of 5" so the multiplicity is clear
    // rather than looking like duplicate questions. Ungrouped items return count 0.
    function variantInfo(mod: any, item: any): { index: number; count: number } {
        const gid = item.question_group_id;
        if (!gid) return { index: 0, count: 0 };
        const siblings = (mod.items ?? []).filter(
            (it: any) => it.question_group_id === gid,
        );
        return { index: siblings.findIndex((it: any) => it.id === item.id) + 1, count: siblings.length };
    }

    // Human-readable summary of a module's questions, deduped by concept so a
    // module with 40 variant items (but only 7 underlying concepts) reads as
    // "7 unique questions · 40 variants" rather than an inflated item count.
    function moduleQuestionSummary(mod: any): string {
        const qItems = (mod.items ?? []).filter((it: any) => it.item_type !== "content");
        const content = (mod.items ?? []).filter((it: any) => it.item_type === "content").length;
        const uniq = uniqueConceptCount(qItems);
        if (qItems.length === 0) return content > 0 ? `${content} content` : "0 items";
        let s = `${uniq} unique question${uniq === 1 ? "" : "s"}`;
        if (qItems.length > uniq) s += ` · ${qItems.length} variants`;
        if (content > 0) s += ` · ${content} content`;
        return s;
    }

    // ---- Per-module generation ----
    async function generateModule(moduleId: number) {
        const mod = (course.modules ?? []).find((m: any) => m.id === moduleId);
        if (!mod) return;
        // Optimistically flip status and clear items — the worker deletes
        // existing items at the start of generation, so we mirror that locally
        // to avoid showing stale content until new items stream in.
        course = {
            ...course,
            modules: course.modules.map((m: any) =>
                m.id === moduleId
                    ? { ...m, status: "generating", items: [] }
                    : m,
            ),
        };
        moduleErrors[moduleId] = "";

        const abort = new AbortController();
        moduleJobs[moduleId] = { jobId: "", steps: [], abort };

        try {
            const res = await fetch(
                `/api/courses/${course.id}/modules/${moduleId}/generate`,
                {
                    method: "POST",
                    headers: { "Content-Type": "application/json" },
                    credentials: "include",
                    body: JSON.stringify({}),
                },
            );
            if (!res.ok) {
                const err = await res.json().catch(() => ({}));
                throw new Error(err.error || "Failed to start generation");
            }
            const { job_id } = await res.json();
            moduleJobs[moduleId] = {
                jobId: job_id,
                steps: [],
                abort,
            };
            // Auto-expand so the user sees progress.
            expandedModules[moduleId] = true;
            await streamModuleJob(moduleId, job_id, abort.signal);
        } catch (e: any) {
            if (e?.name === "AbortError") {
                moduleErrors[moduleId] = "Generation cancelled";
            } else {
                moduleErrors[moduleId] = e?.message || "Generation failed";
            }
            // Revert optimistic status from server truth.
            await refresh();
            // Drop the in-progress job entry.
            const next = { ...moduleJobs };
            delete next[moduleId];
            moduleJobs = next;
        }
    }

    async function cancelModuleGeneration(moduleId: number) {
        const job = moduleJobs[moduleId];
        if (!job) return;
        if (job.abort) job.abort.abort();
        if (job.jobId) {
            try {
                await fetch(
                    `/api/courses/generate/${job.jobId}/cancel`,
                    {
                        method: "POST",
                        credentials: "include",
                    },
                );
            } catch {
                /* best-effort */
            }
        }
    }

    async function streamModuleJob(moduleId: number, jobId: string, signal: AbortSignal) {
        const res = await fetch(`/api/courses/generate/${jobId}/stream`, {
            credentials: "include",
            signal,
        });
        if (!res.ok) {
            moduleErrors[moduleId] = "Failed to connect to stream";
            return;
        }
        const reader = res.body?.getReader();
        if (!reader) {
            moduleErrors[moduleId] = "Streaming not supported";
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
                    let payload: any;
                    try {
                        payload = JSON.parse(line.slice(6));
                    } catch {
                        continue;
                    }
                    if (eventType === "step") {
                        const job = moduleJobs[moduleId];
                        if (job) {
                            moduleJobs[moduleId] = {
                                ...job,
                                steps: [
                                    ...job.steps,
                                    payload.detail || payload.step,
                                ],
                            };
                        }
                    } else if (eventType === "item") {
                        // Live preview: append the new item to the module's
                        // items array so it renders immediately. The item
                        // is already persisted in the DB; this just updates
                        // the local view without waiting for done/refresh.
                        const newItem = payload.item;
                        if (newItem) {
                            course = {
                                ...course,
                                modules: (course.modules ?? []).map(
                                    (m: any) =>
                                        m.id === moduleId
                                            ? {
                                                  ...m,
                                                  items: [
                                                      ...(m.items ?? []),
                                                      newItem,
                                                  ],
                                              }
                                            : m,
                                ),
                            };
                        }
                    } else if (eventType === "error") {
                        moduleErrors[moduleId] =
                            payload.message || "Generation failed";
                        await refresh();
                        const next = { ...moduleJobs };
                        delete next[moduleId];
                        moduleJobs = next;
                        return;
                    } else if (eventType === "done") {
                        // Server has flipped the module to 'ready'. Pull fresh
                        // course state so the new items show up.
                        const next = { ...moduleJobs };
                        delete next[moduleId];
                        moduleJobs = next;
                        await refresh();
                        return;
                    }
                    eventType = "";
                }
            }
        }
    }

    // ---- Module CRUD ----
    function startEditModule(mod: any) {
        editingModuleId = mod.id;
        editBuf = {
            title: mod.title ?? "",
            description: mod.description ?? "",
            // null question_types = "all types". We surface that in the UI
            // as an empty selection with an explicit "all types" hint.
            questionTypes: Array.isArray(mod.question_types)
                ? [...mod.question_types]
                : [],
        };
    }

    function cancelEditModule() {
        editingModuleId = null;
        editBuf = {
            title: "",
            description: "",
            questionTypes: [],
        };
    }

    async function saveEditModule() {
        if (editingModuleId === null) return;
        if (!editBuf.title.trim()) {
            moduleErrors[editingModuleId] = "Title is required";
            return;
        }
        savingModule = true;
        try {
            const res = await fetch(
                `/api/courses/${course.id}/modules/${editingModuleId}`,
                {
                    method: "PUT",
                    headers: { "Content-Type": "application/json" },
                    credentials: "include",
                    body: JSON.stringify({
                        title: editBuf.title.trim(),
                        description: editBuf.description.trim(),
                        // Always send the array so the server can tell
                        // "empty = all types" from "field omitted = leave
                        // unchanged". UpdateModuleHandler prunes unknowns.
                        question_types: editBuf.questionTypes,
                    }),
                },
            );
            if (!res.ok) {
                const err = await res.json().catch(() => ({}));
                moduleErrors[editingModuleId!] = err.error || "Failed to save";
                return;
            }
            editingModuleId = null;
            await refresh();
        } catch {
            moduleErrors[editingModuleId!] = "Network error";
        } finally {
            savingModule = false;
        }
    }

    async function deleteModule(modId: number) {
        if (
            !confirm(
                "Delete this module and all its content? This cannot be undone.",
            )
        )
            return;
        try {
            const res = await fetch(
                `/api/courses/${course.id}/modules/${modId}`,
                {
                    method: "DELETE",
                    credentials: "include",
                },
            );
            if (!res.ok) {
                const err = await res.json().catch(() => ({}));
                moduleErrors[modId] = err.error || "Failed to delete";
                return;
            }
            await refresh();
        } catch {
            moduleErrors[modId] = "Network error";
        }
    }

    // Bulk-delete every selected module in a single request. Mirrors the
    // single delete but hits the batch endpoint so the whole operation is
    // atomic on the backend and the user only confirms once.
    async function deleteSelectedModules() {
        const ids = [...selectedIds];
        if (ids.length === 0) return;
        if (
            !confirm(
                `Delete ${ids.length} module${ids.length === 1 ? "" : "s"} and all their content? This cannot be undone.`,
            )
        )
            return;
        bulkDeleting = true;
        bulkDeleteError = null;
        try {
            const res = await fetch(`/api/courses/${course.id}/modules`, {
                method: "DELETE",
                headers: { "Content-Type": "application/json" },
                credentials: "include",
                body: JSON.stringify({ module_ids: ids }),
            });
            if (!res.ok) {
                const err = await res.json().catch(() => ({}));
                bulkDeleteError = err.error || "Failed to delete modules";
                return;
            }
            clearSelection();
            await refresh();
        } catch {
            bulkDeleteError = "Network error";
        } finally {
            bulkDeleting = false;
        }
    }

    async function addModule() {
        if (!newModuleTitle.trim()) return;
        try {
            const res = await fetch(`/api/courses/${course.id}/modules`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                credentials: "include",
                body: JSON.stringify({
                    title: newModuleTitle.trim(),
                    description: newModuleDesc.trim(),
                }),
            });
            if (!res.ok) {
                const err = await res.json().catch(() => ({}));
                moduleErrors[-1] = err.error || "Failed to add module";
                return;
            }
            newModuleTitle = "";
            newModuleDesc = "";
            addingModule = false;
            await refresh();
        } catch {
            moduleErrors[-1] = "Network error";
        }
    }

    // ---- Outline revision handlers ----
    // Asks the AI to revise the whole outline based on free-text feedback.
    // Streams progress via SSE, then refreshes the course state. Destructive:
    // generated modules + items are wiped and replaced.
    async function startReviseOutline() {
        if (!reviseFeedback.trim()) return;
        // Warn if any module has already been generated — those will be lost.
        const generatedCount = (course.modules ?? []).filter(
            (m: any) => m.status === "ready" || m.status === "failed",
        ).length;
        if (
            generatedCount > 0 &&
            !confirm(
                `This will replace the outline and discard ${generatedCount} generated module${generatedCount === 1 ? "" : "s"}. Continue?`,
            )
        ) {
            return;
        }

        reviseError = "";
        reviseSteps = [];
        revisingOutline = true;
        reviseAbort = new AbortController();

        try {
            const enqueue = await fetch(
                `/api/courses/${course.id}/outline/regenerate`,
                {
                    method: "POST",
                    headers: { "Content-Type": "application/json" },
                    credentials: "include",
                    body: JSON.stringify({
                        instructions: reviseFeedback.trim(),
                    }),
                },
            );
            if (!enqueue.ok) {
                const err = await enqueue.json().catch(() => ({}));
                reviseError = err.error || "Failed to start revision";
                revisingOutline = false;
                return;
            }
            const { job_id } = await enqueue.json();
            await streamReviseJob(job_id, reviseAbort.signal);
        } catch (e: any) {
            if (e?.name === "AbortError") {
                reviseError = "Revision cancelled";
            } else if (!reviseError) {
                reviseError = "Network error";
            }
        } finally {
            revisingOutline = false;
            reviseAbort = null;
        }
    }

    async function cancelReviseOutline() {
        if (reviseAbort) reviseAbort.abort();
        // The job runs server-side; best-effort cancel so the worker stops
        // early instead of finishing a revision the user no longer wants.
        try {
            const activeRes = await fetch("/api/courses/generate/active", {
                credentials: "include",
            });
            if (activeRes.ok) {
                const jobs = await activeRes.json();
                for (const j of jobs) {
                    if (j.stage === "outline") {
                        await fetch(
                            `/api/courses/generate/${j.id}/cancel`,
                            { method: "POST", credentials: "include" },
                        );
                    }
                }
            }
        } catch {
            /* best-effort */
        }
        revisingOutline = false;
    }

    async function streamReviseJob(jobId: string, signal: AbortSignal) {
        const res = await fetch(`/api/courses/generate/${jobId}/stream`, {
            credentials: "include",
            signal,
        });
        if (!res.ok) {
            reviseError = "Failed to connect to stream";
            return;
        }
        const reader = res.body?.getReader();
        if (!reader) {
            reviseError = "Streaming not supported";
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
                    let payload: any;
                    try {
                        payload = JSON.parse(line.slice(6));
                    } catch {
                        continue;
                    }
                    if (eventType === "step") {
                        reviseSteps = [
                            ...reviseSteps,
                            payload.detail || payload.step,
                        ];
                    } else if (eventType === "error") {
                        reviseError =
                            payload.message || "Revision failed";
                        // Server may have partially applied changes; refresh.
                        await refresh();
                        return;
                    } else if (eventType === "done") {
                        reviseFeedback = "";
                        showRevisePanel = false;
                        await refresh();
                        return;
                    }
                    eventType = "";
                }
            }
        }
    }

    // ---- Per-item AI Edit (mirrors ai-content-generator) ----
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
        if (!itemEditInstructions.trim() || editingItemId === null) return;
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
        if (!itemEditPreview) return;
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
            itemEditAcceptedIds = new Set([
                ...itemEditAcceptedIds,
                itemEditPreview.item_id,
            ]);
            cancelItemEdit();
            await refresh();
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
            cancelItemEdit();
            await refresh();
        } catch {
            itemEditError = "Network error while deleting";
        }
    }

    // ---- Course-level summary ----
    let readyCount = $derived(
        (course.modules ?? []).filter((m: any) => m.status === "ready").length,
    );
    let pendingCount = $derived(
        (course.modules ?? []).filter(
            (m: any) => m.status !== "ready",
        ).length,
    );

    // ---- Publish gate / submit for review ----
    // The builder can't publish directly (that's approver-only via the review
    // queue). Instead, "Publish" = submit for review. The button is gated on
    // every module being ready, so incomplete courses can't be submitted.
    let allModulesReady = $derived(
        (course.modules ?? []).length > 0 &&
            (course.modules ?? []).every(
                (m: any) => m.status === "ready",
            ),
    );
    let submitting = $state(false);
    let submitError = $state("");
    let submittedForReview = $state(false);

    // Check if the user can also approve (admin/approver role). If so, we
    // offer a "Publish directly" option in addition to submit-for-review.
    let userRoles = $derived(data?.user?.roles ?? []);
    let canApprove = $derived(
        userRoles.includes("admin") || userRoles.includes("approver"),
    );

    let isAlreadyPublished = $derived(
        course.status === "published" ||
            course.review_status === "approved",
    );
    let isPendingReview = $derived(course.review_status === "pending");

    async function submitForReview() {
        if (!allModulesReady || submitting) return;
        const modCount = course.modules?.length ?? 0;
        if (
            !confirm(
                `Submit "${course.title}" for review? It has ${modCount} module${modCount === 1 ? "" : "s"} and will be sent to the review queue for approval.`,
            )
        )
            return;

        submitting = true;
        submitError = "";
        try {
            const res = await fetch(
                `/api/content-repository/courses/${course.id}/resubmit`,
                {
                    method: "PUT",
                    headers: { "Content-Type": "application/json" },
                    credentials: "include",
                },
            );
            if (!res.ok) {
                const err = await res.json().catch(() => ({}));
                submitError = err.error || "Failed to submit";
                return;
            }
            submittedForReview = true;
            await refresh();
        } catch {
            submitError = "Network error";
        } finally {
            submitting = false;
        }
    }

    async function publishDirectly() {
        if (!allModulesReady || submitting || !canApprove) return;
        if (
            !confirm(
                `Publish "${course.title}" directly? It will be immediately visible to learners.`,
            )
        )
            return;

        submitting = true;
        submitError = "";
        try {
            const res = await fetch(`/api/review/courses/${course.id}`, {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                credentials: "include",
                body: JSON.stringify({
                    review_status: "approved",
                    review_notes: "Published directly from Course Builder",
                }),
            });
            if (!res.ok) {
                const err = await res.json().catch(() => ({}));
                submitError = err.error || "Failed to publish";
                return;
            }
            submittedForReview = true;
            await refresh();
        } catch {
            submitError = "Network error";
        } finally {
            submitting = false;
        }
    }
</script>

<div class="flex w-full max-w-6xl mx-auto flex-col gap-6">
    <PageHeader
        eyebrow="Staged builder"
        title={course.title}
        description={course.description || "No description set."}
    >
        {#snippet icon()}
            <Wand2 class="size-6 text-primary" />
        {/snippet}
        {#snippet actions()}
            {#if isAlreadyPublished}
                <span class="inline-flex items-center gap-1.5 text-xs font-medium text-success px-2.5 py-1.5 rounded-lg border border-success/30 bg-success/5">
                    <CheckCircle2 class="size-3.5" />
                    Published
                </span>
            {:else if isPendingReview}
                <span class="inline-flex items-center gap-1.5 text-xs font-medium text-info px-2.5 py-1.5 rounded-lg border border-info/30 bg-info/5">
                    <CheckCircle2 class="size-3.5" />
                    Pending review
                </span>
            {:else if submittedForReview}
                <span class="inline-flex items-center gap-1.5 text-xs font-medium text-info px-2.5 py-1.5 rounded-lg border border-info/30 bg-info/5">
                    <CheckCircle2 class="size-3.5" />
                    Submitted!
                </span>
            {:else}
                {#if canApprove && allModulesReady}
                    <Button.Root
                        size="sm"
                        onclick={publishDirectly}
                        disabled={submitting}
                        title="Publish immediately — visible to learners right away"
                    >
                        {#if submitting}
                            <LoaderCircle class="size-3.5 mr-1.5 animate-spin" />
                        {:else}
                            <Rocket class="size-3.5 mr-1.5" />
                        {/if}
                        Publish directly
                    </Button.Root>
                {/if}
                <Button.Root
                    size="sm"
                    variant={canApprove ? "outline" : "default"}
                    onclick={submitForReview}
                    disabled={!allModulesReady || submitting}
                    title={
                        !allModulesReady
                            ? `${pendingCount} module${pendingCount === 1 ? "" : "s"} still need generation before you can submit`
                            : "Send to the review queue for approval"
                    }
                >
                    {#if submitting}
                        <LoaderCircle class="size-3.5 mr-1.5 animate-spin" />
                    {:else}
                        <Send class="size-3.5 mr-1.5" />
                    {/if}
                    Submit for review
                </Button.Root>
            {/if}
            <Button.Root
                variant="outline"
                size="sm"
                onclick={() => (editingCourse ? cancelEditCourse() : startEditCourse())}
            >
                {#if editingCourse}
                    <X class="size-4 mr-1.5" />
                    Close
                {:else}
                    <Pencil class="size-4 mr-1.5" />
                    Edit course
                {/if}
            </Button.Root>
            <Button.Root variant="outline" size="sm" onclick={() => goto("/course-builder")}>
                <ArrowLeft class="size-4 mr-1.5" />
                All drafts
            </Button.Root>
        {/snippet}
    </PageHeader>

    {#if editingCourse}
        <div class="rounded-xl border border-border bg-card p-4 flex flex-col gap-3">
            <div class="flex flex-col gap-1.5">
                <label for="course-title-edit" class="text-xs font-medium text-muted-foreground">
                    Course title
                </label>
                <input
                    id="course-title-edit"
                    type="text"
                    bind:value={courseEditBuf.title}
                    disabled={savingCourse}
                    class="rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                    placeholder="Course title"
                />
            </div>
            <div class="flex flex-col gap-1.5">
                <label for="course-desc-edit" class="text-xs font-medium text-muted-foreground">
                    Course description
                </label>
                <textarea
                    id="course-desc-edit"
                    bind:value={courseEditBuf.description}
                    rows={3}
                    disabled={savingCourse}
                    class="rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring resize-none"
                    placeholder="What this course covers"
                ></textarea>
            </div>
            {#if courseEditError}
                <p class="text-xs text-destructive">{courseEditError}</p>
            {/if}
            <div class="flex gap-2">
                <Button.Root size="sm" onclick={saveEditCourse} disabled={savingCourse}>
                    {#if savingCourse}
                        <LoaderCircle class="size-3.5 mr-1 animate-spin" />
                    {:else}
                        <Save class="size-3.5 mr-1" />
                    {/if}
                    Save
                </Button.Root>
                <Button.Root size="sm" variant="outline" onclick={cancelEditCourse} disabled={savingCourse}>
                    Cancel
                </Button.Root>
            </div>
        </div>
    {/if}

    <!-- Sticky toolbar: stays pinned so key actions are always reachable
         even when scrolling through 20+ modules. -->
    <div class="sticky top-0 z-30 -mx-4 px-4 py-2.5 border-b border-border bg-background/95 backdrop-blur-sm flex items-center gap-2 flex-wrap">
        {#if moduleCount > 0}
            <label class="flex items-center gap-1.5 cursor-pointer select-none shrink-0" title={allSelected ? "Clear selection" : "Select all modules"}>
                <input
                    type="checkbox"
                    class="size-4 cursor-pointer rounded border-input text-primary accent-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                    checked={allSelected}
                    indeterminate={someSelected}
                    onchange={toggleSelectAll}
                />
            </label>
        {/if}
        <div class="flex items-center gap-1.5 text-sm">
            <BookOpen class="size-4 text-muted-foreground" />
            <span class="font-medium text-foreground">{course.modules?.length ?? 0}</span>
            <span class="text-muted-foreground">modules</span>
        </div>
        <span class="text-muted-foreground/30">·</span>
        <div class="flex items-center gap-1.5 text-xs">
            <CheckCircle class="size-3.5 text-success" />
            <span class="text-muted-foreground">{readyCount} ready</span>
        </div>
        {#if pendingCount > 0}
            <span class="text-muted-foreground/30">·</span>
            <div class="flex items-center gap-1.5 text-xs">
                <Clock class="size-3.5 text-muted-foreground" />
                <span class="text-muted-foreground">{pendingCount} pending</span>
            </div>
        {/if}

        <div class="ml-auto flex items-center gap-1.5 flex-wrap">
            {#if selectedCount > 0}
                <span class="text-xs text-muted-foreground">{selectedCount} selected</span>
                <Button.Root
                    size="sm"
                    variant="destructive"
                    onclick={deleteSelectedModules}
                    disabled={bulkDeleting}
                    title="Delete all selected modules"
                >
                    {#if bulkDeleting}
                        <LoaderCircle class="size-3.5 mr-1 animate-spin" />
                    {:else}
                        <Trash2 class="size-3.5 mr-1" />
                    {/if}
                    Delete{selectedCount > 1 ? ` ${selectedCount}` : ""}
                </Button.Root>
                <Button.Root
                    size="sm"
                    variant="ghost"
                    onclick={clearSelection}
                    disabled={bulkDeleting}
                    title="Clear selection"
                >
                    <X class="size-3.5 mr-1" />
                    Clear
                </Button.Root>
            {/if}
            {#if anyExpanded}
                <Button.Root
                    size="sm"
                    variant="ghost"
                    onclick={collapseAll}
                    title="Collapse all modules"
                >
                    <ChevronsDownUp class="size-3.5 mr-1" />
                    Collapse all
                </Button.Root>
            {:else if (course.modules?.length ?? 0) > 0}
                <Button.Root
                    size="sm"
                    variant="ghost"
                    onclick={expandAll}
                    title="Expand all modules"
                >
                    <ChevronsUpDown class="size-3.5 mr-1" />
                    Expand all
                </Button.Root>
            {/if}

            {#if generatingAll}
                <span class="text-xs text-muted-foreground">
                    Generating {generateAllDone}/{generateAllTotal}…
                </span>
                <Button.Root
                    size="sm"
                    variant="destructive"
                    onclick={stopGenerateAll}
                >
                    <Square class="size-3.5 mr-1" />
                    Stop
                </Button.Root>
            {:else if pendingCount > 0}
                <Button.Root
                    size="sm"
                    disabled={revisingOutline}
                    onclick={generateAllPending}
                >
                    <Play class="size-3.5 mr-1" />
                    Generate all ({pendingCount})
                </Button.Root>
            {/if}

            {#if isAlreadyPublished}
                <span class="inline-flex items-center gap-1 text-xs font-medium text-success px-2 py-1 rounded-md border border-success/30 bg-success/5">
                    <CheckCircle2 class="size-3" />
                    Published
                </span>
            {:else if isPendingReview}
                <span class="inline-flex items-center gap-1 text-xs font-medium text-info px-2 py-1 rounded-md border border-info/30 bg-info/5">
                    <CheckCircle2 class="size-3" />
                    Pending review
                </span>
            {:else}
                {#if canApprove && allModulesReady}
                    <Button.Root
                        size="sm"
                        onclick={publishDirectly}
                        disabled={submitting}
                    >
                        {#if submitting}
                            <LoaderCircle class="size-3.5 mr-1 animate-spin" />
                        {:else}
                            <Rocket class="size-3.5 mr-1" />
                        {/if}
                        Publish
                    </Button.Root>
                {/if}
                <Button.Root
                    size="sm"
                    variant={canApprove ? "outline" : "default"}
                    onclick={submitForReview}
                    disabled={!allModulesReady || submitting}
                    title={!allModulesReady
                        ? `${pendingCount} module${pendingCount === 1 ? "" : "s"} still need generation`
                        : "Submit to the review queue"}
                >
                    {#if submitting}
                        <LoaderCircle class="size-3.5 mr-1 animate-spin" />
                    {:else}
                        <Send class="size-3.5 mr-1" />
                    {/if}
                    Submit
                </Button.Root>
            {/if}

            <Button.Root
                size="sm"
                variant="outline"
                disabled={revisingOutline || generatingAll}
                onclick={() => (showRevisePanel = !showRevisePanel)}
            >
                <Sparkles class="size-3.5 mr-1" />
                Revise
            </Button.Root>

            <Button.Root
                size="sm"
                variant="outline"
                onclick={() => (addingModule = true)}
                disabled={addingModule}
            >
                <Plus class="size-3.5 mr-1" />
                Add module
            </Button.Root>
        </div>
    </div>

    {#if showRevisePanel}
        <div class="rounded-xl border border-primary/30 bg-primary/5 p-4 flex flex-col gap-3">
            <div class="flex items-start gap-2">
                <Sparkles class="size-4 text-primary shrink-0 mt-0.5" />
                <div class="flex-1 min-w-0">
                    <p class="text-sm font-semibold text-foreground">
                        Revise outline with AI
                    </p>
                    <p class="text-xs text-muted-foreground mt-0.5">
                        Describe how you want the outline changed. The AI keeps what works and only applies your feedback. This replaces the current modules and any generated content.
                    </p>
                </div>
            </div>
            <textarea
                bind:value={reviseFeedback}
                rows={3}
                disabled={revisingOutline}
                placeholder="e.g. ‘cut it down to 4 modules’, ‘add a module on practical applications’, ‘make module 3 more hands-on’, ‘drop the intro module’"
                class="rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring resize-none"
            ></textarea>
            {#if reviseError}
                <div class="rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 flex items-center gap-2">
                    <XCircle class="size-4 text-destructive shrink-0" />
                    <p class="text-xs text-destructive">{reviseError}</p>
                </div>
            {/if}
            {#if revisingOutline && reviseSteps.length > 0}
                <div class="rounded-lg border border-info/30 bg-info/5 px-3 py-2 max-h-32 overflow-y-auto">
                    <div class="flex flex-col gap-0.5">
                        {#each reviseSteps.slice(-4) as s}
                            <p class="text-xs text-info/80">{s}</p>
                        {/each}
                    </div>
                </div>
            {/if}
            <div class="flex items-center gap-2">
                {#if revisingOutline}
                    <Button.Root size="sm" variant="destructive" onclick={cancelReviseOutline}>
                        <X class="size-3.5 mr-1" />
                        Stop
                    </Button.Root>
                    <Button.Root size="sm" disabled>
                        <LoaderCircle class="size-3.5 mr-1 animate-spin" />
                        Revising…
                    </Button.Root>
                {:else}
                    <Button.Root
                        size="sm"
                        disabled={!reviseFeedback.trim()}
                        onclick={startReviseOutline}
                    >
                        <Sparkles class="size-3.5 mr-1" />
                        Revise outline
                    </Button.Root>
                    <Button.Root
                        size="sm"
                        variant="outline"
                        onclick={() => (showRevisePanel = false)}
                    >
                        Cancel
                    </Button.Root>
                {/if}
            </div>
        </div>
    {/if}

    {#if bulkDeleteError}
        <div class="rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 flex items-center gap-2">
            <CircleAlert class="size-4 text-destructive shrink-0" />
            <p class="text-xs text-destructive flex-1">{bulkDeleteError}</p>
            <button
                class="text-xs text-destructive/70 hover:text-destructive"
                onclick={() => (bulkDeleteError = null)}
            >
                Dismiss
            </button>
        </div>
    {/if}

    {#if reorderError}
        <div class="rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 flex items-center gap-2">
            <CircleAlert class="size-4 text-destructive shrink-0" />
            <p class="text-xs text-destructive flex-1">{reorderError}</p>
            <button
                class="text-xs text-destructive/70 hover:text-destructive"
                onclick={() => (reorderError = "")}
            >
                Dismiss
            </button>
        </div>
    {/if}

    {#if submitError}
        <div class="rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 flex items-center gap-2">
            <CircleAlert class="size-4 text-destructive shrink-0" />
            <p class="text-xs text-destructive flex-1">{submitError}</p>
            <button
                class="text-xs text-destructive/70 hover:text-destructive"
                onclick={() => (submitError = "")}
            >
                Dismiss
            </button>
        </div>
    {/if}

    {#if !isAlreadyPublished && !isPendingReview && !allModulesReady && (course.modules?.length ?? 0) > 0}
        <div class="rounded-lg border border-info/30 bg-info/5 px-3 py-2 flex items-center gap-2">
            <Clock class="size-4 text-info shrink-0" />
            <p class="text-xs text-info/90 flex-1">
                {pendingCount} module{pendingCount === 1 ? "" : "s"} still need generation before you can submit for review.
                Use <strong>Generate all</strong> above, or generate modules individually.
            </p>
        </div>
    {/if}

    {#if addingModule}
        <div class="rounded-xl border border-primary/30 bg-primary/5 p-4 flex flex-col gap-3">
            <div class="flex items-center gap-2">
                <Plus class="size-4 text-primary" />
                <p class="text-sm font-semibold text-foreground">Add a new module</p>
            </div>
            <input
                type="text"
                bind:value={newModuleTitle}
                placeholder="Module title"
                class="rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-ring"
            />
            <textarea
                bind:value={newModuleDesc}
                rows={2}
                placeholder="What this module covers (optional)"
                class="rounded-lg border border-input bg-background px-3 py-2 text-xs text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring resize-none"
            ></textarea>
            {#if moduleErrors[-1]}
                <p class="text-xs text-destructive">{moduleErrors[-1]}</p>
            {/if}
            <div class="flex gap-2">
                <Button.Root size="sm" onclick={addModule} disabled={!newModuleTitle.trim()}>
                    <Plus class="size-3.5 mr-1.5" />
                    Add module
                </Button.Root>
                <Button.Root size="sm" variant="outline" onclick={() => (addingModule = false)}>
                    Cancel
                </Button.Root>
            </div>
        </div>
    {/if}

    <!-- Modules -->
    <div class="flex flex-col gap-3">
        {#each course.modules ?? [] as mod, mi (mod.id)}
            {@const Pill = statusPill(mod.status)}
            {@const job = moduleJobs[mod.id]}
            <div class="rounded-xl border border-border bg-card overflow-hidden">
                <!-- Header row -->
                <div class="flex items-center gap-3 px-5 py-4 hover:bg-muted/30 transition-colors">
                    <input
                        type="checkbox"
                        class="size-4 cursor-pointer rounded border-input text-primary accent-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring shrink-0"
                        checked={!!selectedModules[mod.id]}
                        onchange={() => toggleSelect(mod.id)}
                        title={selectedModules[mod.id] ? "Remove from selection" : "Select for bulk delete"}
                        aria-label="Select module {mod.title}"
                    />
                    <button
                        class="shrink-0"
                        onclick={() => toggleModule(mod.id)}
                        title={expandedModules[mod.id] ? "Collapse" : "Expand"}
                    >
                        {#if expandedModules[mod.id]}
                            <ChevronDown class="size-4 text-muted-foreground" />
                        {:else}
                            <ChevronRight class="size-4 text-muted-foreground" />
                        {/if}
                    </button>

                    <div class="flex-1 min-w-0">
                        {#if editingModuleId === mod.id}
                            <!-- Inline edit -->
                            <div class="flex flex-col gap-2">
                                <input
                                    type="text"
                                    bind:value={editBuf.title}
                                    class="rounded-lg border border-input bg-background px-3 py-1.5 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                                    placeholder="Module title"
                                />
                                <textarea
                                    bind:value={editBuf.description}
                                    rows={2}
                                    class="rounded-lg border border-input bg-background px-3 py-1.5 text-xs text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring resize-none"
                                    placeholder="What this module covers..."
                                ></textarea>
                                <!-- Per-module question type allowlist.
                                     Empty selection = allow all types. The
                                     backend normalises & prunes unknowns. -->
                                <div class="flex flex-col gap-1">
                                    <span class="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
                                        Question types
                                        {#if editBuf.questionTypes.length === 0}
                                            <span class="text-muted-foreground/60 font-normal normal-case tracking-normal">(all types)</span>
                                        {:else}
                                            <span class="text-primary font-normal normal-case tracking-normal">({editBuf.questionTypes.length} selected)</span>
                                        {/if}
                                    </span>
                                    <DropdownMenu>
                                        <DropdownMenuTrigger class="flex items-center justify-between rounded-lg border border-input bg-background px-3 py-1.5 text-xs text-foreground hover:bg-muted/50 transition-colors">
                                            {#if editBuf.questionTypes.length === 0}
                                                <span class="text-muted-foreground">All question types</span>
                                            {:else}
                                                <span class="truncate">{editBuf.questionTypes.map((t) => t.replace("_", " ")).join(", ")}</span>
                                            {/if}
                                            <ChevronDown class="size-3.5 text-muted-foreground shrink-0 ml-2" />
                                        </DropdownMenuTrigger>
                                        <DropdownMenuContent class="w-56">
                                            <DropdownMenuLabel>Select question types</DropdownMenuLabel>
                                            <DropdownMenuCheckboxGroup bind:value={editBuf.questionTypes}>
                                                <DropdownMenuCheckboxItem value="mc">Multiple Choice</DropdownMenuCheckboxItem>
                                                <DropdownMenuCheckboxItem value="ma">Multiple Answer</DropdownMenuCheckboxItem>
                                                <DropdownMenuCheckboxItem value="tf">True / False</DropdownMenuCheckboxItem>
                                                <DropdownMenuCheckboxItem value="fb">Fill in the Blank</DropdownMenuCheckboxItem>
                                                <DropdownMenuCheckboxItem value="sa">Short Answer</DropdownMenuCheckboxItem>
                                                <DropdownMenuCheckboxItem value="matching">Matching</DropdownMenuCheckboxItem>
                                                <DropdownMenuCheckboxItem value="drag_sort">Drag &amp; Sort</DropdownMenuCheckboxItem>
                                                <DropdownMenuCheckboxItem value="hotspot">Hotspot</DropdownMenuCheckboxItem>
                                            </DropdownMenuCheckboxGroup>
                                        </DropdownMenuContent>
                                    </DropdownMenu>
                                    {#if editBuf.questionTypes.length > 0}
                                        <p class="text-[10px] text-muted-foreground/70">
                                            The AI will skip any type it can't produce faithfully for this module's content and tell you why.
                                        </p>
                                    {/if}
                                </div>
                                <div class="flex gap-2">
                                    <Button.Root size="sm" onclick={saveEditModule} disabled={savingModule}>
                                        {#if savingModule}
                                            <LoaderCircle class="size-3.5 mr-1 animate-spin" />
                                        {:else}
                                            <Save class="size-3.5 mr-1" />
                                        {/if}
                                        Save
                                    </Button.Root>
                                    <Button.Root size="sm" variant="outline" onclick={cancelEditModule}>
                                        <X class="size-3.5 mr-1" />
                                        Cancel
                                    </Button.Root>
                                </div>
                            </div>
                        {:else}
                            <div class="flex items-center gap-2 flex-wrap">
                                <span class="text-xs font-semibold text-muted-foreground">
                                    Module {mi + 1}
                                </span>
                                <span class="text-xs text-muted-foreground">
                                    {moduleQuestionSummary(mod)}
                                </span>
                                <span class="inline-flex items-center gap-1 text-[10px] font-semibold uppercase tracking-wider px-2 py-0.5 rounded-full {Pill.class}">
                                    {#if mod.status === "generating"}
                                        <LoaderCircle class="size-2.5 animate-spin" />
                                    {:else}
                                        <Pill.Icon class="size-2.5" />
                                    {/if}
                                    {Pill.label}
                                </span>
                            </div>
                            <button
                                class="text-sm font-semibold text-foreground mt-0.5 text-left hover:text-primary transition-colors"
                                onclick={() => toggleModule(mod.id)}
                                title={expandedModules[mod.id] ? "Hide content" : "Preview content"}
                            >
                                {mod.title}
                            </button>
                            {#if mod.description && !expandedModules[mod.id]}
                                <p class="text-xs text-muted-foreground mt-0.5 line-clamp-2">
                                    {mod.description}
                                </p>
                            {/if}
                            <!-- Question-type allowlist + AI limitations summary -->
                            <div class="flex items-center gap-2 flex-wrap mt-1">
                                {#if Array.isArray(mod.question_types) && mod.question_types.length > 0}
                                    <span class="text-[10px] text-muted-foreground">Types:</span>
                                    {#each mod.question_types as qt}
                                        <span class="text-[10px] px-1.5 py-0.5 rounded border border-border bg-muted/40 text-muted-foreground">
                                            {qt.replace("_", " ")}
                                        </span>
                                    {/each}
                                {/if}
                                {#if Array.isArray(mod.limitations) && mod.limitations.length > 0}
                                    <span
                                        class="inline-flex items-center gap-1 text-[10px] px-1.5 py-0.5 rounded border border-warning/40 bg-warning/10 text-warning"
                                        title={mod.limitations.map((l: any) => `${l.type}: ${l.reason}`).join("\n")}
                                    >
                                        <AlertTriangle class="size-2.5" />
                                        {mod.limitations.length} skipped type{mod.limitations.length === 1 ? "" : "s"}
                                    </span>
                                {/if}
                            </div>
                        {/if}
                    </div>

                    <!-- Action buttons (hidden while editing) -->
                    {#if editingModuleId !== mod.id}
                        <div class="flex items-center gap-1 shrink-0">
                            <!-- Reorder controls. Disabled at the ends and
                                 while another reorder is in flight. -->
                            <button
                                class="size-7 rounded-md hover:bg-muted flex items-center justify-center text-muted-foreground hover:text-foreground disabled:opacity-30 disabled:hover:bg-transparent"
                                disabled={reordering || mi === 0}
                                onclick={() => moveModule(mi, -1)}
                                title="Move up"
                            >
                                <ArrowUp class="size-3.5" />
                            </button>
                            <button
                                class="size-7 rounded-md hover:bg-muted flex items-center justify-center text-muted-foreground hover:text-foreground disabled:opacity-30 disabled:hover:bg-transparent"
                                disabled={reordering || mi === (course.modules?.length ?? 0) - 1}
                                onclick={() => moveModule(mi, 1)}
                                title="Move down"
                            >
                                <ArrowDown class="size-3.5" />
                            </button>
                            <!-- Preview toggle: expands the module to show its
                                 content/questions. During generation, items
                                 stream in live. Always visible so the user
                                 has an obvious way to view module content. -->
                            <button
                                class="size-7 rounded-md hover:bg-muted flex items-center justify-center text-muted-foreground hover:text-foreground {expandedModules[mod.id] ? 'bg-muted text-foreground' : ''}"
                                onclick={() => toggleModule(mod.id)}
                                title={expandedModules[mod.id] ? "Hide preview" : "Preview content"}
                            >
                                {#if expandedModules[mod.id]}
                                    <EyeOff class="size-3.5" />
                                {:else}
                                    <Eye class="size-3.5" />
                                {/if}
                            </button>
                            <button
                                class="size-7 rounded-md hover:bg-muted flex items-center justify-center text-muted-foreground hover:text-foreground"
                                onclick={() => startEditModule(mod)}
                                title="Edit module"
                            >
                                <Pencil class="size-3.5" />
                            </button>
                            <button
                                class="size-7 rounded-md hover:bg-destructive/10 hover:text-destructive flex items-center justify-center text-muted-foreground"
                                onclick={() => deleteModule(mod.id)}
                                title="Delete module"
                            >
                                <Trash2 class="size-3.5" />
                            </button>
                            <!-- Learner preview: opens a full-screen modal showing
                                 the module as a learner would see it — clean
                                 content, interactive questions with check/reveal. -->
                            {#if (mod.items?.length ?? 0) > 0}
                                <Button.Root
                                    size="sm"
                                    variant="secondary"
                                    onclick={() => (previewingModule = mod)}
                                >
                                    <Eye class="size-3.5 mr-1" />
                                    Preview
                                </Button.Root>
                            {/if}
                            {#if mod.status === "generating" && job}
                                <Button.Root size="sm" variant="destructive" onclick={() => cancelModuleGeneration(mod.id)}>
                                    <X class="size-3.5 mr-1" />
                                    Stop
                                </Button.Root>
                            {:else}
                                <Button.Root
                                    size="sm"
                                    onclick={() => generateModule(mod.id)}
                                >
                                    {#if mod.status === "ready" || mod.status === "failed"}
                                        <RefreshCw class="size-3.5 mr-1" />
                                        Regenerate
                                    {:else}
                                        <Play class="size-3.5 mr-1" />
                                        Generate
                                    {/if}
                                </Button.Root>
                            {/if}
                        </div>
                    {/if}
                </div>

                <!-- Module-level error -->
                {#if moduleErrors[mod.id]}
                    <div class="px-5 pb-3">
                        <div class="rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 flex items-center gap-2">
                            <CircleAlert class="size-4 text-destructive shrink-0" />
                            <p class="text-xs text-destructive">{moduleErrors[mod.id]}</p>
                        </div>
                    </div>
                {/if}

                <!-- Streaming progress while generating -->
                {#if mod.status === "generating" && job?.steps.length}
                    <div class="px-5 pb-3">
                        <div class="rounded-lg border border-info/30 bg-info/5 px-3 py-2 max-h-32 overflow-y-auto">
                            <div class="flex flex-col gap-0.5">
                                {#each job.steps.slice(-4) as s}
                                    <p class="text-xs text-info/80">{s}</p>
                                {/each}
                            </div>
                        </div>
                    </div>
                {/if}

                <!-- Expanded: items -->
                {#if expandedModules[mod.id]}
                    <div class="border-t border-border px-5 pb-4 pt-3">
                        {#if mod.status === "generating"}
                            <div class="flex items-center gap-1.5 mb-3">
                                <span class="relative flex size-2">
                                    <span class="absolute inline-flex h-full w-full rounded-full bg-info opacity-75 animate-ping"></span>
                                    <span class="relative inline-flex size-2 rounded-full bg-info"></span>
                                </span>
                                <span class="text-[10px] font-semibold uppercase tracking-wider text-info">
                                    Live preview — content streams as it’s generated
                                </span>
                            </div>
                        {/if}
                        {#if mod.description}
                            <p class="text-xs text-muted-foreground mb-3">{mod.description}</p>
                        {/if}

                        {#if Array.isArray(mod.limitations) && mod.limitations.length > 0}
                            <div class="rounded-lg border border-warning/40 bg-warning/10 px-3 py-2 mb-3">
                                <div class="flex items-center gap-1.5 mb-1.5">
                                    <AlertTriangle class="size-3.5 text-warning shrink-0" />
                                    <p class="text-xs font-semibold text-warning">
                                        The AI skipped {mod.limitations.length} requested question type{mod.limitations.length === 1 ? "" : "s"}
                                    </p>
                                </div>
                                <ul class="flex flex-col gap-1 ml-5">
                                    {#each mod.limitations as l}
                                        <li class="text-xs text-warning/90">
                                            <span class="font-mono font-semibold">{l.type?.replace("_", " ") ?? "unknown"}</span>
                                            {#if l.section}<span class="text-warning/60"> · {l.section}:</span>{/if}
                                            <span class="text-warning/80">{l.reason}</span>
                                        </li>
                                    {/each}
                                </ul>
                            </div>
                        {/if}

                        {#if mod.status === "pending"}
                            <div class="rounded-lg border border-dashed border-border bg-muted/20 px-4 py-6 text-center">
                                <Sparkles class="size-5 text-muted-foreground/60 mx-auto mb-2" />
                                <p class="text-sm text-muted-foreground mb-1">
                                    This module hasn’t been generated yet.
                                </p>
                                <p class="text-[11px] text-muted-foreground/70 mb-3">
                                    Tip: use the pencil icon to set which question types the AI should use here first.
                                </p>
                                <Button.Root size="sm" onclick={() => generateModule(mod.id)}>
                                    <Play class="size-3.5 mr-1.5" />
                                    Generate now
                                </Button.Root>
                            </div>
                        {:else if (mod.items ?? []).length === 0}
                            {#if mod.status === "generating"}
                                <div class="flex items-center justify-center gap-2 py-6">
                                    <LoaderCircle class="size-4 text-info animate-spin" />
                                    <p class="text-xs text-info">
                                        Generating — content will appear here as it’s written…
                                    </p>
                                </div>
                            {:else}
                                <p class="text-xs text-muted-foreground text-center py-4">
                                    No items in this module yet.
                                </p>
                            {/if}
                        {:else}
                            <div class="flex flex-col gap-4">
                                {#each mod.items ?? [] as item (item.id)}
                                    {@const d = itemData(item)}
                                    {@const vInfo = variantInfo(mod, item)}
                                    {#if item.item_type === "content"}
                                        <article class="rounded-2xl border border-border bg-card p-6 md:p-8 relative group/item">
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
                                            <span class="text-[10px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">
                                                Learning material
                                            </span>
                                            {#if d.body}
                                                <Markdown source={d.body} />
                                            {/if}
                                        </article>
                                    {:else}
                                        <div class="rounded-2xl border border-border bg-card p-5 relative group/item">
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
                                            <div class="flex items-center justify-between gap-2 mb-2">
                                                <span class="text-[10px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">
                                                    {typeLabels[item.item_type] ?? item.item_type}
                                                </span>
                                                {#if vInfo.count > 1}
                                                    <span class="text-[10px] px-1.5 py-0.5 rounded border border-primary/30 bg-primary/5 text-primary/80" title="Adaptive variants: this concept is rendered in multiple question types so each learner can be served their preferred format.">
                                                        variant {vInfo.index}/{vInfo.count}
                                                    </span>
                                                {/if}
                                            </div>
                                            {#if d.question}
                                                <p class="text-sm font-medium text-foreground">{d.question}</p>
                                            {:else if d.statement}
                                                <p class="text-sm font-medium text-foreground">{d.statement}</p>
                                            {:else if d.text}
                                                <p class="text-sm font-medium text-foreground">{d.text}</p>
                                            {/if}
                                            {#if d.options}
                                                <ul class="mt-2 flex flex-col gap-1">
                                                    {#each d.options as opt, oi}
                                                        <li class="text-xs {d.correct === oi || (Array.isArray(d.correct) && d.correct.includes(oi)) ? 'text-success font-medium' : 'text-muted-foreground'}">
                                                            {opt}
                                                        </li>
                                                    {/each}
                                                </ul>
                                            {/if}
                                            {#if d.explanation}
                                                <p class="text-xs text-muted-foreground mt-2 italic">
                                                    {d.explanation}
                                                </p>
                                            {/if}
                                        </div>
                                    {/if}
                                {/each}
                            </div>
                        {/if}
                    </div>
                {/if}
            </div>
        {/each}

        <!-- Add module -->
        <!-- Bottom add-module shortcut: just scrolls back to the toolbar
             where the real form lives. Keeps the bottom of the list clean. -->
        <div class="rounded-xl border border-dashed border-border bg-card/50 p-4">
            <button
                class="w-full flex items-center justify-center gap-2 text-sm text-muted-foreground hover:text-foreground py-2 transition-colors"
                onclick={() => {
                    addingModule = true;
                    window.scrollTo({ top: 0, behavior: "smooth" });
                }}
            >
                <Plus class="size-4" />
                Add module
            </button>
        </div>
    </div>
</div>

<!-- Learner preview modal -->
{#if previewingModule}
    <ModulePreview
        bind:module={previewingModule}
        courseTitle={course.title}
        allModules={course.modules ?? []}
        courseSources={course.sources ?? []}
        onClose={() => (previewingModule = null)}
    />
{/if}
