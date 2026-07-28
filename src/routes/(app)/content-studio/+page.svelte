<script lang="ts">
    import {
        Upload,
        FileText,
        Trash2,
        LoaderCircle,
        CheckCircle,
        XCircle,
        Clock,
        CircleAlert,
    } from "@lucide/svelte";
    import * as Button from "$lib/components/ui/button";
    import { PageHeader } from "$lib/components/brand";

    let { data } = $props();

    // ---- State ----
    // Seeded from server-side load, kept mutable for polling/updates
    let documents = $state<any[]>([]);
    $effect(() => {
        documents = data.documents ?? [];
    });
    let uploading = $state(false);
    let uploadError = $state("");
    let uploadStatus = $state<"idle" | "success" | "error">("idle");
    let justUploadedId = $state<number | null>(null);

    // ---- Poll while active docs exist ----
    let pollInterval: ReturnType<typeof setInterval>;
    $effect(() => {
        const hasActive = documents.some(
            (d: any) => d.status === "uploaded" || d.status === "processing",
        );
        if (hasActive) {
            pollInterval = setInterval(refreshDocuments, 1500);
        } else {
            if (pollInterval) clearInterval(pollInterval);
        }
        return () => {
            if (pollInterval) clearInterval(pollInterval);
        };
    });

    // ---- Data fetching (client-side polling only) ----
    async function refreshDocuments() {
        try {
            const res = await fetch("/api/documents", {
                credentials: "include",
            });
            if (res.ok) {
                documents = await res.json();
                const stillActive = documents.some(
                    (d: any) =>
                        d.status === "uploaded" || d.status === "processing",
                );
                if (!stillActive) {
                    justUploadedId = null;
                }
            }
        } catch {
            // ignore
        }
    }

    // ---- Upload ----
    let uploadForm: HTMLFormElement;
    async function handleUpload() {
        const form = uploadForm;
        if (!form) return;
        const formData = new FormData(form);
        const fileInput = form.querySelector(
            'input[type="file"]',
        ) as HTMLInputElement;
        const file = fileInput?.files?.[0];

        uploadError = "";
        uploadStatus = "idle";

        if (!file) {
            uploadError = "Please select a PDF file.";
            uploadStatus = "error";
            return;
        }

        const ACCEPTED = [".pdf", ".txt", ".md"];
        const lower = file.name.toLowerCase();
        if (!ACCEPTED.some((ext) => lower.endsWith(ext))) {
            uploadError = "Accepts PDF, TXT, or Markdown files.";
            uploadStatus = "error";
            return;
        }

        uploading = true;

        try {
            const res = await fetch("/api/documents/upload", {
                method: "POST",
                credentials: "include",
                body: formData,
            });

            if (!res.ok) {
                const err = await res.json();
                uploadError = err.error || "Upload failed.";
                uploadStatus = "error";
                return;
            }

            const doc = await res.json();
            justUploadedId = doc.id;
            uploadStatus = "success";
            form.reset();
            await refreshDocuments();

            // Clear success after 5s
            setTimeout(() => {
                uploadStatus = "idle";
            }, 5000);
        } catch {
            uploadError = "Network error. Is the backend running?";
            uploadStatus = "error";
        } finally {
            uploading = false;
        }
    }

    // ---- Delete ----
    async function handleDelete(id: number) {
        try {
            const res = await fetch(`/api/documents/${id}`, {
                method: "DELETE",
                credentials: "include",
            });
            if (res.ok) {
                documents = documents.filter((d: any) => d.id !== id);
            }
        } catch {
            // ignore
        }
    }

    // ---- Helpers ----
    function statusIcon(status: string) {
        switch (status) {
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
    }

    function statusColor(status: string) {
        switch (status) {
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
    }

    function statusLabel(status: string) {
        switch (status) {
            case "uploaded":
                return "Queued";
            case "processing":
                return "Processing";
            case "ready":
                return "Ready";
            case "failed":
                return "Failed";
            default:
                return status;
        }
    }

    function progressPct(doc: any) {
        if (!doc.total_chunks || doc.total_chunks === 0) return 0;
        return Math.round((doc.chunks_done / doc.total_chunks) * 100);
    }

    function formatDate(dateStr: string) {
        if (!dateStr) return "";
        return new Date(dateStr).toLocaleDateString("en-US", {
            year: "numeric",
            month: "short",
            day: "numeric",
            hour: "2-digit",
            minute: "2-digit",
        });
    }

    let activeDocs = $derived(
        documents.filter(
            (d: any) => d.status === "uploaded" || d.status === "processing",
        ),
    );
    let doneDocs = $derived(
        documents.filter(
            (d: any) => d.status === "ready" || d.status === "failed",
        ),
    );
</script>

<div class="flex w-full max-w-6xl mx-auto flex-col gap-6">
    <PageHeader
        title="Content Studio"
        eyebrow="Documents"
        description="Upload PDFs and policy documents. Gia indexes them automatically and exposes them to learners after approval."
    >
        {#snippet icon()}
            <Upload class="size-6 text-primary" />
        {/snippet}
    </PageHeader>

    <div class="flex flex-col md:flex-row gap-6">
        <!-- Left column: Upload -->
        <section class="md:w-[380px] shrink-0 flex flex-col gap-4">
            <!-- Upload Status Feedback -->
            {#if uploadStatus === "success"}
                <div
                    role="status"
                    class="rounded-xl border border-success/30 bg-success/10 px-4 py-3 flex items-start gap-3 motion-rise-in"
                >
                    <CheckCircle class="size-4 text-success shrink-0 mt-0.5" />
                    <div>
                        <p class="text-sm font-medium text-success">
                            Upload successful
                        </p>
                        <p class="text-xs text-success/80">
                            Processing will begin automatically.
                        </p>
                    </div>
                </div>
            {:else if uploadStatus === "error"}
                <div
                    role="alert"
                    class="rounded-xl border border-destructive/30 bg-destructive/10 px-4 py-3 flex items-start gap-3 motion-rise-in"
                >
                    <CircleAlert
                        class="size-4 text-destructive shrink-0 mt-0.5"
                    />
                    <div>
                        <p class="text-sm font-medium text-destructive">
                            Upload failed
                        </p>
                        <p class="text-xs text-destructive/80">
                            {uploadError}
                        </p>
                    </div>
                </div>
            {/if}

            <div class="rounded-xl border border-border bg-card p-6">
                <h2
                    class="text-base font-semibold tracking-tight text-foreground mb-4"
                >
                    Upload PDF
                </h2>

                <form
                    method="POST"
                    enctype="multipart/form-data"
                    bind:this={uploadForm}
                    onsubmit={(e: SubmitEvent) => {
                        e.preventDefault();
                        handleUpload();
                    }}
                >
                    <div class="flex flex-col gap-4">
                        <div>
                            <label
                                for="title"
                                class="block text-sm font-medium text-foreground mb-1.5"
                            >
                                Title <span
                                    class="text-muted-foreground font-normal"
                                    >(optional)</span
                                >
                            </label>
                            <input
                                type="text"
                                id="title"
                                name="title"
                                placeholder="Employee Handbook 2025"
                                class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                            />
                        </div>

                        <div>
                            <label
                                for="file-upload"
                                class="block text-sm font-medium text-foreground mb-1.5"
                            >
                                PDF File
                            </label>
                            <div
                                class="flex flex-col items-center justify-center rounded-lg border-2 border-dashed border-border bg-muted/50 px-4 py-6 transition-colors hover:border-primary/50"
                            >
                                <Upload
                                    class="size-6 text-muted-foreground mb-1.5"
                                />
                                <p class="text-xs text-muted-foreground mb-2">
                                    Drag &amp; drop or click to browse
                                </p>
                                <p
                                    class="text-[10px] uppercase tracking-wider text-muted-foreground/70 mb-2"
                                >
                                    PDF · TXT · Markdown
                                </p>
                                <input
                                    id="file-upload"
                                    type="file"
                                    name="file"
                                    accept=".pdf,.txt,.md,application/pdf,text/plain,text/markdown"
                                    class="block w-full text-xs text-muted-foreground file:mr-3 file:rounded-md file:border-0 file:bg-primary file:px-3 file:py-1.5 file:text-xs file:font-medium file:text-primary-foreground hover:file:bg-primary/90 cursor-pointer"
                                />
                            </div>
                        </div>

                        <Button.Root
                            disabled={uploading}
                            class="w-full"
                            onclick={(e: MouseEvent) => {
                                e.preventDefault();
                                handleUpload();
                            }}
                        >
                            {#if uploading}
                                <LoaderCircle
                                    class="size-4 mr-2 animate-spin"
                                />
                                Uploading…
                            {:else}
                                <Upload class="size-4 mr-2" />
                                Upload PDF
                            {/if}
                        </Button.Root>
                    </div>
                </form>
            </div>
        </section>

        <!-- Right column: Document Queue -->
        <section class="flex-1 min-w-0 flex flex-col gap-6">
            <!-- Active / Processing -->
            <div>
                <h2
                    class="text-base font-semibold tracking-tight text-foreground mb-3 flex items-center gap-2"
                >
                    {#if activeDocs.length > 0}
                        <LoaderCircle class="size-4 text-info animate-spin" />
                        Processing
                    {:else}
                        <CheckCircle class="size-4 text-muted-foreground" />
                        Idle
                    {/if}
                    <span class="text-xs font-normal text-muted-foreground"
                        >({activeDocs.length})</span
                    >
                </h2>

                {#if activeDocs.length > 0}
                    <div class="flex flex-col gap-3">
                        {#each activeDocs as doc (doc.id)}
                            {@const pct = progressPct(doc)}
                            <div
                                class="rounded-xl border border-border bg-card p-4 transition-all {doc.id ===
                                justUploadedId
                                    ? 'ring-2 ring-primary/30'
                                    : ''}"
                            >
                                <div
                                    class="flex items-start justify-between gap-3 mb-3"
                                >
                                    <div
                                        class="flex items-center gap-2.5 min-w-0"
                                    >
                                        <FileText
                                            class="size-4 text-muted-foreground shrink-0"
                                        />
                                        <div class="min-w-0">
                                            <p
                                                class="text-sm font-medium text-foreground truncate"
                                            >
                                                {doc.title}
                                            </p>
                                            <p
                                                class="text-xs text-muted-foreground"
                                            >
                                                {formatDate(doc.created_at)}
                                            </p>
                                        </div>
                                    </div>
                                    <div
                                        class="flex items-center gap-1.5 shrink-0"
                                    >
                                        {#if doc.status === "processing"}
                                            <span
                                                class="text-xs font-medium tabular-nums text-info"
                                                >{pct}%</span
                                            >
                                        {:else if doc.status === "uploaded"}
                                            <span
                                                class="text-xs font-medium text-warning"
                                                >queued</span
                                            >
                                        {/if}
                                        <Button.Root
                                            variant="ghost"
                                            size="icon"
                                            class="size-7"
                                            onclick={() => handleDelete(doc.id)}
                                        >
                                            <Trash2
                                                class="size-3.5 text-muted-foreground hover:text-destructive"
                                            />
                                        </Button.Root>
                                    </div>
                                </div>

                                <div class="space-y-1.5">
                                    <div
                                        class="flex items-center justify-between text-xs text-muted-foreground"
                                    >
                                        <span>{statusLabel(doc.status)}</span>
                                        <span
                                            >{doc.chunks_done} / {doc.total_chunks ||
                                                "?"} chunks</span
                                        >
                                    </div>
                                    <div
                                        class="h-1.5 w-full rounded-full bg-muted overflow-hidden"
                                    >
                                        <div
                                            class="h-full rounded-full transition-all duration-500 ease-out"
                                            class:bg-warning={doc.status ===
                                                "uploaded"}
                                            class:bg-info={doc.status ===
                                                "processing"}
                                            style="width: {doc.status ===
                                            'uploaded'
                                                ? '8%'
                                                : Math.max(pct, 2) + '%'}"
                                        ></div>
                                    </div>
                                </div>
                            </div>
                        {/each}
                    </div>
                {:else}
                    <div
                        class="rounded-xl border border-border bg-card p-6 text-center"
                    >
                        <p class="text-xs text-muted-foreground">
                            {documents.length === 0
                                ? "Upload a PDF to start processing."
                                : "No documents currently processing."}
                        </p>
                    </div>
                {/if}
            </div>

            <!-- Completed Documents -->
            <div>
                <h2
                    class="text-base font-semibold tracking-tight text-foreground mb-3 flex items-center gap-2"
                >
                    <CheckCircle class="size-4 text-success" />
                    Completed
                    <span class="text-xs font-normal text-muted-foreground"
                        >({doneDocs.length})</span
                    >
                </h2>

                {#if doneDocs.length === 0 && documents.length === 0}
                    <div
                        class="rounded-xl border border-border bg-card p-10 text-center"
                    >
                        <FileText
                            class="size-8 text-muted-foreground/50 mx-auto mb-2"
                        />
                        <p class="text-sm text-muted-foreground">
                            No documents yet. Upload a PDF to get started.
                        </p>
                    </div>
                {:else if doneDocs.length === 0}
                    <div
                        class="rounded-xl border border-border bg-card p-6 text-center"
                    >
                        <p class="text-xs text-muted-foreground">
                            Completed documents appear here when processing
                            finishes.
                        </p>
                    </div>
                {:else}
                    <div
                        class="rounded-xl border border-border bg-card overflow-hidden"
                    >
                        <table class="w-full text-sm">
                            <thead>
                                <tr class="border-b border-border bg-muted/50">
                                    <th
                                        class="px-4 py-2.5 text-left font-medium text-muted-foreground"
                                        >Title</th
                                    >
                                    <th
                                        class="px-4 py-2.5 text-left font-medium text-muted-foreground hidden sm:table-cell"
                                        >Chunks</th
                                    >
                                    <th
                                        class="px-4 py-2.5 text-left font-medium text-muted-foreground hidden sm:table-cell"
                                        >Uploaded</th
                                    >
                                    <th
                                        class="px-4 py-2.5 text-right font-medium text-muted-foreground w-10"
                                    ></th>
                                </tr>
                            </thead>
                            <tbody>
                                {#each doneDocs as doc (doc.id)}
                                    {@const Icon = statusIcon(doc.status)}
                                    <tr
                                        class="border-b border-border last:border-0 hover:bg-muted/30 transition-colors"
                                    >
                                        <td class="px-4 py-2.5">
                                            <div
                                                class="flex items-center gap-2"
                                            >
                                                <Icon
                                                    class="size-3.5 shrink-0 {statusColor(
                                                        doc.status,
                                                    )}"
                                                />
                                                <div class="min-w-0">
                                                    <span
                                                        class="font-medium text-foreground truncate max-w-[200px] block"
                                                        >{doc.title}</span
                                                    >
                                                    {#if doc.status === "failed" && doc.error_message}
                                                        <p
                                                            class="text-[10px] text-destructive/80 mt-0.5 line-clamp-2"
                                                            title={doc.error_message}
                                                        >
                                                            {doc.error_message}
                                                        </p>
                                                    {/if}
                                                </div>
                                            </div>
                                        </td>
                                        <td
                                            class="px-4 py-2.5 text-muted-foreground hidden sm:table-cell"
                                        >
                                            {doc.total_chunks || 0}
                                        </td>
                                        <td
                                            class="px-4 py-2.5 text-muted-foreground hidden sm:table-cell text-xs"
                                        >
                                            {formatDate(doc.created_at)}
                                        </td>
                                        <td class="px-4 py-2.5 text-right">
                                            <Button.Root
                                                variant="ghost"
                                                size="icon"
                                                class="size-7"
                                                onclick={() =>
                                                    handleDelete(doc.id)}
                                            >
                                                <Trash2
                                                    class="size-3.5 text-muted-foreground hover:text-destructive"
                                                />
                                            </Button.Root>
                                        </td>
                                    </tr>
                                {/each}
                            </tbody>
                        </table>
                    </div>
                {/if}
            </div>
        </section>
    </div>
</div>
