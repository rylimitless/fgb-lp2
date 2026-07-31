<script lang="ts">
	import { Medal } from "@lucide/svelte";
	import DashboardCard from "./DashboardCard.svelte";
	import type { LeaderboardItem } from "./types";

	let { rows, loading = false }: { rows: LeaderboardItem[]; loading?: boolean } = $props();
	const scopes = [
		{ id: "global", label: "Global" },
		{ id: "department", label: "Department" },
		{ id: "friends", label: "Friends" },
	] as const;
	type Scope = (typeof scopes)[number]["id"];
	let activeScope = $state<Scope>("global");

	function handleTabKey(event: KeyboardEvent) {
		if (!["ArrowLeft", "ArrowRight", "Home", "End"].includes(event.key)) return;
		event.preventDefault();
		const currentIndex = scopes.findIndex((scope) => scope.id === activeScope);
		const nextIndex =
			event.key === "Home"
				? 0
				: event.key === "End"
					? scopes.length - 1
					: (currentIndex + (event.key === "ArrowRight" ? 1 : -1) + scopes.length) % scopes.length;
		activeScope = scopes[nextIndex].id;
		requestAnimationFrame(() => {
			document.querySelector<HTMLButtonElement>(`[data-leaderboard-tab="${activeScope}"]`)?.focus();
		});
	}
</script>

<DashboardCard title="Leaderboards" class="h-full">
	{#snippet action()}
		<span class="text-[8px] text-muted-foreground">All time</span>
	{/snippet}
	<div class="flex gap-3 border-b border-border px-3 text-[8px]" role="tablist" aria-label="Leaderboard scope">
		{#each scopes as scope}
			<button
				type="button"
				role="tab"
				id={`leaderboard-tab-${scope.id}`}
				data-leaderboard-tab={scope.id}
				aria-selected={activeScope === scope.id}
				aria-controls="leaderboard-panel"
				tabindex={activeScope === scope.id ? 0 : -1}
				class="focus-premium border-b py-1.5 font-semibold transition-colors {activeScope === scope.id ? 'border-accent text-foreground' : 'border-transparent text-muted-foreground hover:text-foreground'}"
				onclick={() => (activeScope = scope.id)}
				onkeydown={handleTabKey}
			>
				{scope.label}
			</button>
		{/each}
	</div>
	<div id="leaderboard-panel" role="tabpanel" aria-labelledby={`leaderboard-tab-${activeScope}`} tabindex="0">
	{#if activeScope !== "global"}
		<p class="flex min-h-[72px] items-center justify-center px-4 text-center text-[9px] leading-relaxed text-muted-foreground">
			{activeScope === "department"
				? "Department rankings are not available in the current Academy data."
				: "Friends rankings are not available in the current Academy data."}
		</p>
	{:else if loading}
		<div class="space-y-1 p-2" aria-label="Loading leaderboard">
			{#each [1, 2, 3, 4] as _}<div class="h-7 animate-pulse rounded bg-muted"></div>{/each}
		</div>
	{:else if rows.length}
		<ol class="grid grid-cols-1 gap-x-4 px-2 py-1 sm:grid-cols-2">
			{#each rows.slice(0, 6) as row, index}
				<li class="flex min-h-[22px] items-center gap-2 border-b border-border/60 px-1 text-[8px] {row.me ? 'bg-accent/8' : ''}">
					<span class="flex size-4 shrink-0 items-center justify-center rounded-full text-[8px] font-bold {index < 3 ? 'bg-accent text-accent-foreground' : 'bg-muted text-muted-foreground'}">
						{#if index === 0}<Medal class="size-2.5" />{:else}{index + 1}{/if}
					</span>
					<span class="flex size-5 shrink-0 items-center justify-center rounded-full bg-primary/15 text-[7px] font-bold text-primary">{row.name.slice(0, 2).toUpperCase()}</span>
					<span class="min-w-0 flex-1 break-words leading-tight text-foreground">{row.name}{row.me ? " (You)" : ""}</span>
					<strong class="tabular text-accent">{row.xp.toLocaleString()} XP</strong>
				</li>
			{/each}
		</ol>
	{:else}
		<p class="p-4 text-center text-[10px] text-muted-foreground">No leaderboard results yet.</p>
	{/if}
	</div>
</DashboardCard>
