<script lang="ts">
	import { cn } from "$lib/utils.js";
	import { Trophy } from "@lucide/svelte";
	import BadgeMedal from "./BadgeMedal.svelte";
	import GiaTip from "./GiaTip.svelte";

	type CompletedCourse = {
		id: number;
		title: string;
		progress?: {
			score_pct?: string | number;
			completed?: boolean;
		};
	};

	type Props = {
		courses: CompletedCourse[];
		class?: string;
	};

	let { courses, class: className }: Props = $props();

	let achievements = $derived(
		courses
			.filter((c) => c.progress?.completed)
			.map((c) => {
				const raw = c.progress?.score_pct;
				const score =
					typeof raw === "string" ? parseFloat(raw) : (raw ?? 0);
				const tier: "bronze" | "silver" | "gold" =
					score >= 90 ? "gold" : score >= 75 ? "silver" : "bronze";
				return { id: c.id, title: c.title, score, tier };
			}),
	);

	let goldCount = $derived(
		achievements.filter((a) => a.tier === "gold").length,
	);
</script>

<section
	class={cn(
		"rounded-2xl border border-border bg-card p-5 lift",
		className,
	)}
	aria-labelledby="achievements-heading"
>
	<div class="flex items-center justify-between mb-4">
		<h3
			id="achievements-heading"
			class="text-sm font-semibold text-foreground inline-flex items-center gap-2"
		>
			<Trophy class="size-4 text-accent" />
			Achievements
		</h3>
		{#if achievements.length > 0}
			<span class="text-[10px] uppercase tracking-wider text-muted-foreground tabular">
				{achievements.length} earned
				{#if goldCount > 0}
					· {goldCount} gold
				{/if}
			</span>
		{/if}
	</div>

	{#if achievements.length === 0}
		<GiaTip
			size="sm"
			message="Finish a course to earn your first badge. Bronze at 50%, silver at 75%, gold at 90%."
		/>
	{:else}
		<ul
			class="-mx-1 flex items-end gap-4 overflow-x-auto snap-x snap-mandatory pb-1 px-1 list-none"
		>
			{#each achievements as a, i (a.id)}
				<li class="snap-start shrink-0 motion-rise-in" style="animation-delay: {Math.min(i * 40, 240)}ms">
					<a
						href={`/lesson-player?retake=${a.id}`}
						class="block press"
						title={`${a.title} · ${Math.round(a.score)}% · ${a.tier}`}
						aria-label={`${a.title}, scored ${Math.round(a.score)} percent, ${a.tier} tier`}
					>
						<BadgeMedal tier={a.tier} size={68} label={a.tier} />
					</a>
				</li>
			{/each}
		</ul>
	{/if}
</section>
