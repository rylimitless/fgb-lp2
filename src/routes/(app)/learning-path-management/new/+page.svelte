<script lang="ts">
    import { Layers, Plus, ArrowLeft, Wand2 } from "@lucide/svelte";
    import * as Button from "$lib/components/ui/button";
    import { PageHeader } from "$lib/components/brand";
    import { showToast } from "$lib/components/brand/toast.svelte";
    import { goto } from "$app/navigation";

    let title = $state("");
    let description = $state("");
    let error = $state("");
    let creating = $state(false);

    async function handleSubmit(e: Event) {
        e.preventDefault();
        error = "";
        if (!title.trim()) {
            error = "Title is required.";
            return;
        }
        creating = true;
        try {
            const res = await fetch("/api/admin/learning-paths", {
                method: "POST",
                credentials: "include",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    title: title.trim(),
                    description: description.trim(),
                }),
            });
            if (!res.ok) {
                const err = await res.json().catch(() => ({}));
                error = err.error ?? "Failed to create learning path";
                return;
            }
            const created = await res.json();
            showToast("Learning path created.", { variant: "success" });
            await goto(`/learning-path-management/${created.id}`);
        } finally {
            creating = false;
        }
    }
</script>

<svelte:head><title>New Learning Path - FGB Academy</title></svelte:head>

<div class="mx-auto flex w-full max-w-3xl flex-col gap-6">
    <PageHeader
        title="New Learning Path"
        eyebrow="Content"
        description="Create a learning path manually. You'll add courses and learners next."
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

    <form
        onsubmit={handleSubmit}
        class="flex flex-col gap-4 rounded-xl border border-border bg-card p-6"
    >
        <div class="flex flex-col gap-1.5">
            <label for="lp-title" class="text-sm font-medium text-foreground"
                >Title</label
            >
            <input
                id="lp-title"
                type="text"
                bind:value={title}
                placeholder="e.g. Leadership Fundamentals"
                class="w-full rounded-lg border border-border bg-muted px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
            />
        </div>
        <div class="flex flex-col gap-1.5">
            <label for="lp-desc" class="text-sm font-medium text-foreground"
                >Description</label
            >
            <textarea
                id="lp-desc"
                rows="4"
                bind:value={description}
                placeholder="Optional description of the learning journey..."
                class="w-full rounded-lg border border-border bg-muted px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
            ></textarea>
        </div>

        {#if error}
            <p class="text-sm text-destructive">{error}</p>
        {/if}

        <div class="flex justify-end gap-2">
            <Button.Root
                type="button"
                variant="outline"
                size="sm"
                onclick={() => goto("/learning-path-management")}
                >Cancel</Button.Root
            >
            <Button.Root type="submit" variant="default" size="sm" disabled={creating}>
                <Plus class="size-4" />
                {creating ? "Creating..." : "Create Path"}
            </Button.Root>
        </div>
    </form>

    <!-- Link to AI builder -->
    <div
        class="flex items-center justify-between gap-3 rounded-xl border border-primary/30 bg-primary/5 p-4"
    >
        <div>
            <p class="text-sm font-semibold text-foreground">
                Or let AI assemble it for you
            </p>
            <p class="text-xs text-muted-foreground">
                Describe a goal and the AI picks &amp; orders courses from your
                approved catalog.
            </p>
        </div>
        <Button.Root
            variant="default"
            size="sm"
            onclick={() => goto("/learning-path-management/builder")}
        >
            <Wand2 class="size-4" /> AI Builder
        </Button.Root>
    </div>
</div>
