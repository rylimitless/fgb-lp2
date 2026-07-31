<script lang="ts">
	import { ArrowUpRight, Newspaper } from "@lucide/svelte";
	import { goto } from "$app/navigation";
	import DashboardCard from "./DashboardCard.svelte";
	import type { NewsItem } from "./types";

	let { items }: { items: NewsItem[] } = $props();

	function openNews(item: NewsItem) {
		const href = item.href ?? `/gia-coach?q=${encodeURIComponent(`Tell me more about this Academy update: ${item.title}`)}`;
		if (/^https?:\/\//i.test(href)) window.location.assign(href);
		else goto(href);
	}
</script>

<DashboardCard title="Latest news" class="h-full">
	{#snippet action()}
		<button class="focus-premium rounded px-1.5 py-1 text-[8px] text-muted-foreground hover:bg-muted hover:text-foreground" onclick={() => goto("/gia-coach?q=What%20are%20the%20latest%20FGB%20Academy%20updates%3F")}>View all ›</button>
	{/snippet}
	{#if items.length}
		<ul class="divide-y divide-border px-2">
			{#each items.slice(0, 5) as item}
				<li>
					<button type="button" class="focus-premium group flex min-h-7 w-full items-start gap-2 rounded px-1 py-1 text-left hover:bg-muted/60" onclick={() => openNews(item)}>
						<span class="flex h-5 w-8 shrink-0 items-center justify-center rounded-sm bg-gradient-to-br from-primary/30 to-accent/25 text-primary">
							{#if item.icon}<item.icon class="size-3" aria-hidden="true" />{:else}<Newspaper class="size-3" />{/if}
						</span>
						<span class="w-16 shrink-0 pt-0.5 text-[7px] uppercase text-muted-foreground">{item.meta}</span>
						<span class="min-w-0 flex-1 break-words text-[8.5px] leading-snug text-foreground">{item.title}</span>
						<ArrowUpRight class="mt-0.5 size-3 shrink-0 text-muted-foreground transition-colors group-hover:text-accent" aria-hidden="true" />
					</button>
				</li>
			{/each}
		</ul>
	{:else}
		<p class="p-4 text-center text-[10px] text-muted-foreground">No Academy news is available right now.</p>
	{/if}
</DashboardCard>
