<script lang="ts">
	import { AlarmClock, ChevronRight } from "@lucide/svelte";
	import { goto } from "$app/navigation";
	import type { DeadlineItem } from "./types";

	let { deadline }: { deadline?: DeadlineItem } = $props();

	const countdown = $derived(
		deadline?.isOverdue
			? "OVERDUE"
			: deadline?.daysLeft === 0
				? "DUE TODAY"
				: deadline?.daysLeft != null
					? `${deadline.daysLeft} DAYS LEFT`
					: "NO DEADLINE",
	);
</script>

{#if deadline}
	<button
		type="button"
		class="focus-premium flex min-h-[48px] w-full items-center gap-3 rounded-xl border border-destructive/45 bg-destructive/14 px-4 text-left text-foreground shadow-sm transition-colors hover:bg-destructive/20"
		onclick={() => goto(`/lesson-player?id=${deadline.courseId}`)}
		aria-label={`${deadline.title}, ${countdown}, ${deadline.due}`}
	>
		<AlarmClock class="size-5 shrink-0 text-destructive" aria-hidden="true" />
		<span class="shrink-0 border-r border-destructive/25 pr-3">
			<span class="block text-xs uppercase tracking-wider text-muted-foreground">Course deadline</span>
			<strong class="block text-sm font-bold text-destructive">{countdown}</strong>
		</span>
		<span class="min-w-0 flex-1 truncate text-xs font-semibold uppercase tracking-wide">{deadline.title}</span>
		<ChevronRight class="size-4 shrink-0 text-destructive" />
	</button>
{:else}
	<div class="flex min-h-[48px] items-center gap-2 rounded-xl border border-success/25 bg-success/8 px-4 text-sm text-muted-foreground">
		<AlarmClock class="size-4 text-success" /> No upcoming deadlines — you're on track.
	</div>
{/if}
