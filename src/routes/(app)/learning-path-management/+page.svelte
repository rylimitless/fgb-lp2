<script lang="ts">
    import type { LearningPath } from "./+page.server";
    import { Layers, Plus, BookOpen, ChevronRight, Wand2 } from "@lucide/svelte";
    import * as Button from "$lib/components/ui/button";
    import { PageHeader, StatCard, EmptyState } from "$lib/components/brand";
    import { showToast } from "$lib/components/brand/toast.svelte";
    import { goto } from "$app/navigation";

    let { data } = $props();

    let paths: LearningPath[] = $state([]);

    $effect(() => {
        paths = data.paths ?? [];
    });

    let totalPaths = $derived(paths.length);
    let publishedPaths = $derived(
        paths.filter((p) => p.status === "published").length,
    );

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

    async function handleStatusChange(p: LearningPath, newStatus: string) {
        const res = await fetch(`/api/admin/learning-paths/${p.id}`, {
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
        paths = paths.map((x) => (x.id === updated.id ? updated : x));
        showToast(`Status changed to ${newStatus}.`, { variant: "success" });
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
                variant="outline"
                size="sm"
                onclick={() => goto("/learning-path-management/builder")}
            >
                <Wand2 class="size-4" /> AI Builder
            </Button.Root>
            <Button.Root
                variant="default"
                size="sm"
                onclick={() => goto("/learning-path-management/new")}
            >
                <Plus class="size-4" /> New Path
            </Button.Root>
        {/snippet}
    </PageHeader>

    <div class="grid grid-cols-1 gap-4 md:grid-cols-3 motion-rise-in motion-stagger-1">
        <StatCard label="Total Paths" value={totalPaths}>
            {#snippet icon()}<Layers class="size-5 text-primary" />{/snippet}
        </StatCard>
        <StatCard label="Published" value={publishedPaths}>
            {#snippet icon()}<BookOpen class="size-5 text-success" />{/snippet}
        </StatCard>
        <StatCard label="Drafts" value={totalPaths - publishedPaths}>
            {#snippet icon()}<Layers class="size-5 text-accent-foreground" />{/snippet}
        </StatCard>
    </div>

    {#if paths.length === 0}
        <EmptyState
            title="No learning paths yet"
            description="Create your first learning path to group courses into a guided curriculum."
        >
            {#snippet icon()}<Layers class="size-12" />{/snippet}
            {#snippet action()}
                <div class="flex flex-wrap justify-center gap-2">
                    <Button.Root
                        variant="default"
                        size="sm"
                        onclick={() => goto("/learning-path-management/new")}
                    >
                        <Plus class="size-4" /> Create Path
                    </Button.Root>
                    <Button.Root
                        variant="outline"
                        size="sm"
                        onclick={() =>
                            goto("/learning-path-management/builder")}
                    >
                        <Wand2 class="size-4" /> AI Builder
                    </Button.Root>
                </div>
            {/snippet}
        </EmptyState>
    {:else}
        <div class="space-y-3 motion-rise-in motion-stagger-2">
            {#each paths as p (p.id)}
                <div class="rounded-2xl border border-border bg-card p-4 shadow-sm">
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
                            {#if p.description}
                                <p
                                    class="mt-1 text-sm text-muted-foreground line-clamp-2"
                                >
                                    {p.description}
                                </p>
                            {/if}
                        </div>
                        <div class="flex shrink-0 items-center gap-1.5">
                            <Button.Root
                                variant="default"
                                size="sm"
                                onclick={() =>
                                    goto(`/learning-path-management/${p.id}`)}
                            >
                                Manage <ChevronRight class="size-4" />
                            </Button.Root>
                        </div>
                    </div>

                    <div
                        class="mt-3 flex flex-wrap items-center gap-2 border-t border-border pt-3"
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
                                    {statusLabel(s)}
                                </Button.Root>
                            {/if}
                        {/each}
                    </div>
                </div>
            {/each}
        </div>
    {/if}
</div>
