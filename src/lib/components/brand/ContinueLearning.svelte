<script lang="ts">
	import { cn } from "$lib/utils.js";
	import { Play, ArrowRight, Layers, Clock } from "@lucide/svelte";
	import * as Button from "$lib/components/ui/button";
	import Spotlight from "./Spotlight.svelte";
	import BorderBeam from "./BorderBeam.svelte";
	import { Tween, prefersReducedMotion } from "svelte/motion";
	import { cubicOut } from "svelte/easing";

	type Course = {
		id: number;
		title: string;
		description?: string;
		source_doc_ids?: number[];
		progress?: {
			current_module?: number;
			completed?: boolean;
			score_pct?: string | number;
		};
	};

	type Props = {
		course: Course;
		href?: string;
		class?: string;
	};

	let { course, href, class: className }: Props = $props();

	// Module count is unknown without the full course payload. The dashboard
	// load only returns the course shell; the lesson-player owns the modules.
	// We surface "Module N" honestly and link to the player to resume.
	let currentModule = $derived((course.progress?.current_module ?? 0) + 1);
	let estProgress = $derived(
		Math.min(((course.progress?.current_module ?? 0) / 5) * 100, 92),
	);

	const tween = new Tween(0, { duration: 900, easing: cubicOut });
	$effect(() => {
		if (prefersReducedMotion.current) {
			tween.set(estProgress, { duration: 0 });
		} else {
			tween.target = estProgress;
		}
	});
</script>

<a
	href={href ?? "/lesson-player"}
	class={cn(
		"group relative block overflow-hidden rounded-3xl border border-border brand-gradient text-primary-foreground px-6 py-7 md:px-8 md:py-8 lift press",
		className,
	)}
	aria-label={`Continue learning ${course.title}`}
>
	<Spotlight class="h-full w-full" opacity={0.4} />
	<BorderBeam size={120} duration={9} color="var(--accent)" />

	<div class="relative z-10 flex items-start gap-4">
		<div
			class="flex size-11 items-center justify-center rounded-full bg-accent/15 text-accent shrink-0"
			aria-hidden="true"
		>
			<Play class="size-4 fill-accent" />
		</div>
		<div class="min-w-0 flex-1 flex flex-col gap-3">
			<div>
				<span
					class="text-[10px] font-semibold uppercase tracking-[0.18em] text-accent"
				>
					Continue learning
				</span>
				<h2
					class="mt-1 text-display-md font-semibold tracking-tight text-primary-foreground leading-tight"
				>
					{course.title}
				</h2>
				{#if course.description}
					<p
						class="mt-1.5 text-sm text-primary-foreground/75 line-clamp-2 max-w-xl"
					>
						{course.description}
					</p>
				{/if}
			</div>

			<div class="flex items-center gap-4 text-xs text-primary-foreground/70">
				<span class="inline-flex items-center gap-1 tabular">
					<Layers class="size-3.5" />
					Module {currentModule}
				</span>
				{#if course.source_doc_ids?.length}
					<span class="inline-flex items-center gap-1 tabular">
						<Clock class="size-3.5" />
						{course.source_doc_ids.length} source{course.source_doc_ids.length === 1 ? "" : "s"}
					</span>
				{/if}
			</div>

			<div class="relative h-2 w-full rounded-full bg-primary-foreground/15 overflow-hidden">
				<div
					class="h-full rounded-full bg-gradient-to-r from-accent to-accent/70"
					style:width="{tween.current}%"
				></div>
			</div>

			<div class="flex items-center justify-between gap-3 mt-1">
				<span
					class="text-[11px] text-primary-foreground/60 tabular"
					>{Math.round(estProgress)}% through</span
				>
				<Button.Root
					size="lg"
					class="bg-accent text-accent-foreground hover:bg-accent/85 shadow-glow group-hover:translate-x-0.5 transition-transform"
				>
					Resume
					<ArrowRight class="size-4" />
				</Button.Root>
			</div>
		</div>
	</div>
</a>
