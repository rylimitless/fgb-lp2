<script lang="ts">
	import { Check, ChevronRight, Lock, Map, Play } from "@lucide/svelte";
	import { goto } from "$app/navigation";
	import DashboardCard from "./DashboardCard.svelte";
	import type { LearningStep } from "./types";

	let { steps }: { steps: LearningStep[] } = $props();

	function state(step: LearningStep, index: number) {
		if (step.status === "Completed") return { label: "Completed", kind: "done", Icon: Check };
		if (step.status === "In Progress") return { label: "In progress", kind: "current", Icon: Play };
		const final = index === steps.length - 1;
		return { label: final ? "Locked" : "Up next", kind: final ? "locked" : "next", Icon: final ? Lock : Play };
	}
</script>

<DashboardCard title="Learning path">
	{#snippet action()}
		<button class="focus-premium inline-flex items-center gap-1 rounded px-2 py-1.5 text-xs font-medium text-foreground hover:bg-muted" onclick={() => goto("/adaptive-room")}>
			<Map class="size-4" /> View full path <ChevronRight class="size-4" />
		</button>
	{/snippet}
	<div class="overflow-x-auto px-4 py-3 [scrollbar-width:thin]">
		<ol class="flex min-w-[620px] items-start" aria-label="Your learning path">
			{#each steps as step, index}
				{@const item = state(step, index)}
				<li class="relative flex flex-1 flex-col items-center text-center">
					{#if index < steps.length - 1}
						<span class="absolute left-1/2 top-4 h-px w-full bg-border" aria-hidden="true">
							{#if item.kind === "done"}<span class="block h-full w-full bg-success"></span>{/if}
						</span>
					{/if}
					<span class:item-current={item.kind === "current"} class:item-done={item.kind === "done"} class="relative z-10 flex size-7 items-center justify-center rounded-full border border-border-strong bg-card text-muted-foreground">
						<item.Icon class="size-4" aria-hidden="true" />
					</span>
					<span class="mt-1.5 max-w-28 break-words px-1.5 text-xs leading-tight text-foreground">{step.label}</span>
					<span class="mt-1 text-[10px] uppercase leading-none tracking-wide {item.kind === 'done' ? 'text-success' : item.kind === 'current' ? 'text-accent' : 'text-muted-foreground'}">{item.label}</span>
				</li>
			{/each}
		</ol>
	</div>
</DashboardCard>

<style>
	.item-done {
		border-color: var(--success);
		background: color-mix(in oklab, var(--success) 18%, var(--card));
		color: var(--success);
	}
	.item-current {
		border-color: var(--accent);
		background: var(--accent);
		color: var(--accent-foreground);
		box-shadow: var(--glow-gold);
	}
</style>
