<script lang="ts" generics="T">
	import { cn } from "$lib/utils.js";
	import type { Snippet } from "svelte";
	import EmptyState from "./EmptyState.svelte";

	type Props<T> = {
		items: T[];
		columns: string[];
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
		title,
		description,
		emptyTitle = "No records found",
		emptyDescription = "Try adjusting your filters or search query.",
		class: className,
		row,
	}: Props<T> = $props();
</script>

<section class={cn("overflow-hidden rounded-3xl border border-border bg-card shadow-sm", className)}>
	{#if title || description}
		<div class="flex flex-wrap items-end justify-between gap-3 border-b border-border bg-gradient-to-r from-surface-1 to-card px-5 py-4">
			<div>
				{#if title}
					<h3 class="text-sm font-semibold text-foreground">{title}</h3>
				{/if}
				{#if description}
					<p class="mt-0.5 text-xs text-muted-foreground">{description}</p>
				{/if}
			</div>
			<span class="rounded-full border border-border bg-background px-2.5 py-1 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground tabular">
				{items.length} rows
			</span>
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
					<tr class="border-b border-border bg-muted/30 text-left text-xs text-muted-foreground">
						{#each columns as col}
							<th class="whitespace-nowrap px-5 py-3 font-semibold uppercase tracking-[0.12em]">
								{col}
							</th>
						{/each}
					</tr>
				</thead>
				<tbody>
					{#each items as item, i}
						<tr class="border-b border-border/45 transition-colors hover:bg-muted/20 last:border-0">
							{@render row(item, i)}
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</section>
