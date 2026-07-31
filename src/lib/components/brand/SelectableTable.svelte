<script lang="ts" generics="T">
    import { cn } from "$lib/utils.js";
    import type { Snippet } from "svelte";
    import EmptyState from "./EmptyState.svelte";

    type Props<T> = {
        items: T[];
        columns: string[];
        idKey?: string;
        bindSelectedIds?: number[];
        onSelectionChange?: (ids: number[]) => void;
        title?: string;
        description?: string;
        emptyTitle?: string;
        emptyDescription?: string;
        class?: string;
        row: Snippet<[T, number]>;
    };

    let {
        items,
        columns,
        idKey = "id",
        bindSelectedIds = $bindable([]),
        onSelectionChange,
        title,
        description,
        emptyTitle = "No records found",
        emptyDescription = "Try adjusting your filters or search query.",
        class: className,
        row,
    }: Props<T> = $props();

    let selectedIds = $state<number[]>([]);
    $effect(() => {
        selectedIds = bindSelectedIds;
    });
    $effect(() => {
        bindSelectedIds = selectedIds;
    });

    let allSelected = $derived(
        items.length > 0 && selectedIds.length === items.length,
    );
    let someSelected = $derived(
        selectedIds.length > 0 && selectedIds.length < items.length,
    );

    function toggleAll() {
        if (allSelected) {
            selectedIds = [];
        } else {
            selectedIds = items.map((item: any) => item[idKey]);
        }
        onSelectionChange?.(selectedIds);
    }

    function toggleOne(id: number) {
        if (selectedIds.includes(id)) {
            selectedIds = selectedIds.filter((sid) => sid !== id);
        } else {
            selectedIds = [...selectedIds, id];
        }
        onSelectionChange?.(selectedIds);
    }

    // Extract the row's id by idKey. `T` is unconstrained (so the component
    // works for any row shape), but the selection logic needs an id — cast here.
    function rowId(item: T): number {
        return (item as any)[idKey];
    }
</script>

<section
    class={cn(
        "overflow-hidden rounded-3xl border border-border bg-card shadow-sm",
        className,
    )}
>
    {#if title || description}
        <div
            class="flex flex-wrap items-end justify-between gap-3 border-b border-border bg-gradient-to-r from-surface-1 to-card px-5 py-4"
        >
            <div>
                {#if title}
                    <h3 class="text-sm font-semibold text-foreground">
                        {title}
                    </h3>
                {/if}
                {#if description}
                    <p class="mt-0.5 text-xs text-muted-foreground">
                        {description}
                    </p>
                {/if}
            </div>
            <div class="flex items-center gap-2">
                {#if selectedIds.length > 0}
                    <span
                        class="rounded-full border border-primary/30 bg-primary/10 px-2.5 py-1 text-[10px] font-semibold uppercase tracking-wider text-primary tabular"
                    >
                        {selectedIds.length} selected
                    </span>
                {/if}
                <span
                    class="rounded-full border border-border bg-background px-2.5 py-1 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground tabular"
                >
                    {items.length} rows
                </span>
            </div>
        </div>
    {/if}

    {#if items.length === 0}
        <div class="p-5">
            <EmptyState title={emptyTitle} description={emptyDescription} />
        </div>
    {:else}
        <div class="overflow-x-auto">
            <table class="w-full text-sm">
                <thead>
                    <tr
                        class="border-b border-border bg-muted/30 text-left text-xs text-muted-foreground"
                    >
                        <th
                            class="w-10 whitespace-nowrap px-5 py-3 font-semibold uppercase tracking-[0.12em]"
                        >
                            <label class="flex items-center justify-center">
                                <input
                                    type="checkbox"
                                    class="size-4 rounded border-muted-foreground/30 cursor-pointer accent-primary"
                                    checked={allSelected}
                                    oninput={toggleAll}
                                />
                            </label>
                        </th>
                        {#each columns as col}
                            <th
                                class="whitespace-nowrap px-5 py-3 font-semibold uppercase tracking-[0.12em]"
                            >
                                {col}
                            </th>
                        {/each}
                    </tr>
                </thead>
                <tbody>
                    {#each items as item, i (rowId(item))}
                        <tr
                            class="border-b border-border/45 transition-colors hover:bg-muted/20 last:border-0"
                        >
                            <td class="w-10 px-5 py-3">
                                <label class="flex items-center justify-center">
                                    <input
                                        type="checkbox"
                                        class="size-4 rounded border-muted-foreground/30 cursor-pointer accent-primary"
                                        checked={selectedIds.includes(
                                            rowId(item),
                                        )}
                                        oninput={() => toggleOne(rowId(item))}
                                    />
                                </label>
                            </td>
                            {@render row(item, i)}
                        </tr>
                    {/each}
                </tbody>
            </table>
        </div>
    {/if}
</section>
