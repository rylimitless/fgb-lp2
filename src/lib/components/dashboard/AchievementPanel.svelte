<script lang="ts">
	import { CheckCircle2, CircleDashed, LockKeyhole, Trophy } from "@lucide/svelte";
	import BadgeMedal from "$lib/components/brand/BadgeMedal.svelte";
	import DashboardCard from "./DashboardCard.svelte";
	import type { AchievementItem } from "./types";

	let { achievements, xpReward = 40 }: { achievements: AchievementItem[]; xpReward?: number } = $props();
	let selected = $state(1);
	let displayItems = $derived(
		achievements.length
			? achievements.slice(0, 4)
			: [{ label: "Start your first course", tier: "bronze" as const }],
	);
	let current = $derived(displayItems[Math.min(selected, displayItems.length - 1)]);

	function progressFor(label: string) {
		const normalized = label.toLowerCase();
		if (normalized.includes("eager beaver")) return { value: 0, label: "Not started" };
		if (normalized.includes("rising rooster")) return { value: 50, label: "Halfway" };
		return { value: 100, label: "Completed" };
	}

	let currentProgress = $derived(progressFor(current?.label ?? ""));
</script>

<DashboardCard title="Progress / Achievements" class="h-full">
	<div class="grid min-h-[140px] grid-cols-[minmax(0,1.05fr)_minmax(120px,.95fr)] gap-3 p-3">
		<div class="flex flex-col gap-1">
			{#each displayItems as achievement, index}
				{@const progress = progressFor(achievement.label)}
				<button
					type="button"
					class="focus-premium flex min-h-10 items-center gap-2 rounded-md border px-3 py-2 text-left text-xs transition-colors {selected === index ? 'border-accent/55 bg-accent/10 text-foreground' : 'border-border bg-surface-1 text-muted-foreground hover:border-border-strong'}"
					onclick={() => (selected = index)}
					aria-pressed={selected === index}
				>
					<BadgeMedal tier={achievement.tier} size={28} />
					<span class="min-w-0 flex-1">
						<span class="flex items-start justify-between gap-1">
							<span class="break-words font-medium leading-tight">{achievement.label}</span>
							<span class="shrink-0 text-[10px] uppercase tracking-wide {progress.value === 100 ? 'text-success' : progress.value > 0 ? 'text-accent' : 'text-muted-foreground'}">{progress.label}</span>
						</span>
							<span class="mt-1 block h-1.5 overflow-hidden rounded-full bg-muted" aria-hidden="true">
							<span class="block h-full rounded-full {progress.value === 100 ? 'bg-success' : 'bg-accent'}" style:width={`${progress.value}%`}></span>
						</span>
					</span>
					{#if progress.value === 100}
						<CheckCircle2 class="size-4 shrink-0 text-success" />
					{:else if progress.value > 0}
						<CircleDashed class="size-4 shrink-0 text-accent" />
					{:else}
						<LockKeyhole class="size-4 shrink-0" />
					{/if}
				</button>
			{/each}
		</div>
		<div class="relative overflow-hidden rounded-md border border-border bg-surface-1 p-3">
			<Trophy class="absolute -bottom-2 -right-2 size-16 text-accent/8" aria-hidden="true" />
			<h3 class="academy-heading text-sm text-foreground">How to excel?</h3>
			<p class="mt-1.5 text-xs leading-relaxed text-muted-foreground">
				{#if currentProgress.value === 0}
					Start the next activity in your learning path to begin earning {current?.label}.
				{:else if currentProgress.value < 100}
					You're halfway to {current?.label}. Keep your learning streak active to finish it.
				{:else}
					{current?.label} is complete. Select another achievement to see your next milestone.
				{/if}
			</p>
			<div class="mt-3 border-t border-border pt-3">
				<p class="text-xs uppercase tracking-wider text-muted-foreground">XP reward</p>
				<p class="text-lg font-bold tabular text-accent">{xpReward} XP</p>
			</div>
			<span class="sr-only">Selected achievement: {current?.label}</span>
		</div>
	</div>
</DashboardCard>
