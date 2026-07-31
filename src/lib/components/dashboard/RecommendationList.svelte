<script lang="ts">
	import { BookOpen, Clock, Plus } from "@lucide/svelte";
	import { goto } from "$app/navigation";
	import DashboardCard from "./DashboardCard.svelte";
	import type { RecommendationItem } from "./types";

	let { items }: { items: RecommendationItem[] } = $props();
</script>

<DashboardCard title="Recommendations" class="h-full">
	{#snippet action()}
		<button class="focus-premium rounded px-1.5 py-1 text-[8px] text-muted-foreground hover:bg-muted hover:text-foreground" onclick={() => goto("/lesson-player")}>View all ›</button>
	{/snippet}
	{#if items.length}
		<ul class="divide-y divide-border px-2" aria-label="Recommended courses">
			{#each items.slice(0, 4) as item}
				<li>
					<button class="focus-premium group flex min-h-6 w-full items-center gap-2 rounded-md px-1 py-0.5 text-left hover:bg-muted/60" onclick={() => goto(item.id ? `/lesson-player?id=${item.id}` : "/lesson-player")}>
						<span class="flex h-5 w-10 shrink-0 items-center justify-center overflow-hidden rounded bg-gradient-to-br from-primary/65 to-[#071a2c] text-primary-foreground">
							{#if item.image}
								<img src={item.image} alt="" class="size-full object-cover" />
							{:else}
								<BookOpen class="size-3" aria-hidden="true" />
							{/if}
						</span>
						<span class="min-w-0 flex-1">
							<span class="block break-words text-[9px] font-medium leading-snug text-foreground">{item.title}</span>
							<span class="mt-0.5 flex items-center gap-1 text-[7.5px] text-muted-foreground">
								{item.level ?? "Course"} {#if item.duration}<span>•</span><Clock class="size-2" /> {item.duration}{/if}
							</span>
						</span>
						<span class="flex size-5 shrink-0 items-center justify-center rounded border border-border-strong text-muted-foreground transition-colors group-hover:border-accent group-hover:text-accent" aria-hidden="true"><Plus class="size-3" /></span>
					</button>
				</li>
			{/each}
		</ul>
	{:else}
		<p class="p-4 text-center text-[10px] text-muted-foreground">Recommendations will appear as your learning profile grows.</p>
	{/if}
</DashboardCard>
