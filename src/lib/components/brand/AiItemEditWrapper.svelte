<script lang="ts">
    import {
        Pencil,
        LoaderCircle,
        CheckCircle,
        XCircle,
        Sparkles,
        Trash2,
    } from "@lucide/svelte";
    import * as Button from "$lib/components/ui/button";
    import Markdown from "$lib/components/brand/Markdown.svelte";

    let {
        item,
        editingItemId,
        itemEditInstructions,
        itemEditLoading,
        itemEditError,
        itemEditPreview,
        onStartEdit,
        onCancelEdit,
        onAiEdit,
        onAccept,
        onDiscard,
        onInstructionsInput,
        onDelete,
    }: {
        item: any;
        editingItemId: number | null;
        itemEditInstructions: string;
        itemEditLoading: boolean;
        itemEditError: string;
        itemEditPreview: any;
        onStartEdit: (itemId: number) => void;
        onCancelEdit: () => void;
        onAiEdit: () => void;
        onAccept: () => void;
        onDiscard: () => void;
        onInstructionsInput: (e: Event) => void;
        onDelete: (itemId: number) => void;
    } = $props();

    let isEditing = $derived(editingItemId === item.id);
    let hasPreview = $derived(
        itemEditPreview && itemEditPreview.item_id === item.id,
    );
    // Show the delete action only for assessment items, not learning content.
    let isQuestion = $derived(item.item_type !== "content");
</script>

<!-- Pencil button + popover anchored to top-right of the card -->
<div class="absolute top-2 right-2 z-30">
    <button
        class="size-7 rounded-full bg-primary/90 text-primary-foreground flex items-center justify-center opacity-0 group-hover/item:opacity-100 transition-opacity hover:bg-primary press shadow-md {isEditing
            ? '!opacity-100'
            : ''}"
        onclick={() => onStartEdit(item.id)}
        title="AI Edit this section"
    >
        <Pencil class="size-3.5" />
    </button>

    {#if isEditing}
        <!-- Click-away backdrop -->
        <button
            class="fixed inset-0 z-40 cursor-default"
            aria-label="Close edit panel"
            onclick={onCancelEdit}
        ></button>

        <!-- Popover panel anchored below the pencil icon -->
        <div
            class="absolute top-full right-0 mt-1.5 z-50 w-[420px] max-w-[calc(100vw-2rem)] rounded-xl border border-border bg-card shadow-2xl"
        >
            <div
                class="flex items-center gap-2 px-4 py-3 border-b border-border"
            >
                <Sparkles class="size-4 text-info shrink-0" />
                <span class="text-xs font-semibold text-info"
                    >AI Edit this section</span
                >
                {#if isQuestion}
                    <button
                        class="ml-auto size-7 rounded-md flex items-center justify-center text-destructive hover:bg-destructive/10 transition-colors"
                        onclick={() => onDelete(item.id)}
                        title="Delete this question"
                    >
                        <Trash2 class="size-4" />
                    </button>
                    <button
                        class="size-5 rounded flex items-center justify-center text-muted-foreground hover:text-foreground transition-colors"
                        onclick={onCancelEdit}
                    >
                        <XCircle class="size-4" />
                    </button>
                {:else}
                    <button
                        class="ml-auto size-5 rounded flex items-center justify-center text-muted-foreground hover:text-foreground transition-colors"
                        onclick={onCancelEdit}
                    >
                        <XCircle class="size-4" />
                    </button>
                {/if}
            </div>

            <div class="p-4 max-h-[60vh] overflow-y-auto">
                {#if !hasPreview}
                    <div class="flex flex-col gap-2">
                        <input
                            type="text"
                            value={itemEditInstructions}
                            oninput={onInstructionsInput}
                            placeholder="e.g., Make this more concise, add an example..."
                            class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                            onkeydown={(e: KeyboardEvent) => {
                                if (e.key === "Enter" && !itemEditLoading) {
                                    onAiEdit();
                                }
                            }}
                        />
                        <Button.Root
                            size="sm"
                            class="w-full"
                            disabled={itemEditLoading ||
                                !itemEditInstructions.trim()}
                            onclick={onAiEdit}
                        >
                            {#if itemEditLoading}
                                <LoaderCircle
                                    class="size-3.5 mr-1.5 animate-spin"
                                />
                                Generating…
                            {:else}
                                <Sparkles class="size-3.5 mr-1.5" />
                                Edit with AI
                            {/if}
                        </Button.Root>
                    </div>
                {/if}

                {#if itemEditError}
                    <div
                        class="mt-2 rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 flex items-center gap-2"
                    >
                        <XCircle class="size-3.5 text-destructive shrink-0" />
                        <p class="text-xs text-destructive">
                            {itemEditError}
                        </p>
                    </div>
                {/if}

                {#if hasPreview}
                    <div class="flex flex-col gap-3">
                        <div
                            class="rounded-lg border border-success/30 bg-success/5 p-3"
                        >
                            <div class="flex items-center gap-2 mb-2">
                                <CheckCircle
                                    class="size-4 text-success shrink-0"
                                />
                                <span class="text-xs font-semibold text-success"
                                    >AI Suggestion</span
                                >
                            </div>

                            {#if itemEditPreview.item_type === "content"}
                                <div
                                    class="rounded-lg border border-border bg-card p-3 max-h-48 overflow-y-auto"
                                >
                                    <Markdown
                                        source={typeof itemEditPreview.suggested_data ===
                                        "string"
                                            ? (JSON.parse(
                                                  itemEditPreview.suggested_data,
                                              ).body ?? "")
                                            : (itemEditPreview.suggested_data
                                                  .body ?? "")}
                                        class="max-w-prose text-[13px]"
                                    />
                                </div>
                            {:else}
                                <pre
                                    class="text-xs text-foreground bg-muted/30 rounded-lg p-2.5 overflow-x-auto whitespace-pre-wrap max-h-48">{JSON.stringify(
                                        typeof itemEditPreview.suggested_data ===
                                            "string"
                                            ? JSON.parse(
                                                  itemEditPreview.suggested_data,
                                              )
                                            : itemEditPreview.suggested_data,
                                        null,
                                        2,
                                    )}</pre>
                            {/if}
                        </div>

                        <div class="flex gap-2">
                            <Button.Root
                                size="sm"
                                variant="outline"
                                class="flex-1"
                                onclick={onDiscard}
                            >
                                <XCircle class="size-3.5 mr-1.5" />
                                Discard
                            </Button.Root>
                            <Button.Root
                                size="sm"
                                class="flex-1 bg-success hover:bg-success/90"
                                onclick={onAccept}
                            >
                                <CheckCircle class="size-3.5 mr-1.5" />
                                Accept &amp; Save
                            </Button.Root>
                        </div>
                    </div>
                {/if}
            </div>
        </div>
    {/if}
</div>
