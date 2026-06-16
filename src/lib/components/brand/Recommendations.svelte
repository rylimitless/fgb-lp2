<script lang="ts">
	import { cn } from "$lib/utils.js";
	import { Sparkles, ArrowRight, BookOpen, Layers } from "@lucide/svelte";
	import GiaTip from "./GiaTip.svelte";

	type Course = {
		id: number;
		title: string;
		description?: string;
		source_doc_ids?: number[];
		status?: string;
		progress?: { completed?: boolean; current_module?: number } | null;
	};

	type Props = {
		courses: Course[];
		limit?: number;
		class?: string;
	};

	let { courses, limit = 3, class: className }: Props = $props();

	// Recommended = published, not started yet. We derive this client-side
	// from the courses payload already returned by the dashboard load.
	let recommended = $derived(
		courses
			.filter(
				(c) =>
					c.status === "published" &&
					(!c.progress || c.progress.current_module === undefined),
			)
			.slice(0, limit),
	);

	// Programmatic gradient generator: deterministic per course id so a course
	// always reads the same colour in the absence of a Higgsfield-rendered
	// thumbnail. Returns a CSS gradient string.
	function gradientFor(id: number): string {
		const seed = (id * 7919) % 360;
		const start = `oklch(0.42 0.13 ${(seed + 245) % 360})`;
		const end = `oklch(0.62 0.13 ${(seed + 220) % 360})`;
		return `linear-gradient(135deg, ${start} 0%, ${end} 100%)`;
	}

	function initialFor(title: string): string {
		return title?.[0]?.toUpperCase() ?? "?";
	}

	// Mirrors CourseCard.svelte resolver — keep the two in sync.
	const PATHWAY_HINTS: Array<[RegExp, string]> = [
		[/cyber|security|infosec/i, "cybersecurity"],
		[/compliance|regulator|policy|aml/i, "compliance"],
		[/risk|credit|portfolio/i, "risk-credit"],
		[/leadership|executive|management/i, "leadership"],
		[/customer|service|relationship|retail/i, "customer-service"],
		[/banking|foundation|introduction|fundamental/i, "banking-foundations"],
	];
	function slugify(s: string): string {
		return (s ?? "").toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "");
	}
	function pathwayImageFor(title: string): string | undefined {
		for (const [pattern, slug] of PATHWAY_HINTS) {
			if (pattern.test(title)) return `/brand/pathways/${slug}.png`;
		}
		const slug = slugify(title);
		return slug ? `/brand/modules/${slug}.png` : undefined;
	}
</script>

<section
	class={cn(
		"rounded-2xl border border-border bg-card p-5 lift",
		className,
	)}
	aria-labelledby="recs-heading"
>
	<div class="flex items-center justify-between mb-4">
		<h3
			id="recs-heading"
			class="text-sm font-semibold text-foreground inline-flex items-center gap-2"
		>
			<Sparkles class="size-4 text-accent" />
			Recommended for you
		</h3>
		<a
			class="inline-flex items-center gap-1 text-xs font-medium text-primary hover:underline"
			href="/lesson-player"
		>
			Browse all
			<ArrowRight class="size-3" />
		</a>
	</div>

	{#if recommended.length === 0}
		<GiaTip
			size="sm"
			message="No fresh recommendations yet. Once a few more courses are published, I'll line them up here."
		/>
	{:else}
		<div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-3">
			{#each recommended as course, i (course.id)}
				<a
					href={`/lesson-player`}
					class="group relative flex flex-col overflow-hidden rounded-xl border border-border bg-surface-1 lift press motion-rise-in"
					style="animation-delay: {Math.min(i * 40, 200)}ms"
					title={course.description ?? course.title}
				>
					<!-- 16:9 thumbnail slot — programmatic gradient + initial sit
					     underneath; if a Higgsfield render exists it overlays them. -->
					<div
						class="relative aspect-video w-full overflow-hidden"
						style:background={gradientFor(course.id)}
					>
						<span
							class="absolute inset-0 flex items-center justify-center text-primary-foreground/85 text-3xl font-bold tracking-tight"
							aria-hidden="true"
						>
							{initialFor(course.title)}
						</span>
						{#if pathwayImageFor(course.title)}
							<img
								src={pathwayImageFor(course.title)}
								alt=""
								class="absolute inset-0 size-full object-cover"
								loading="lazy"
								decoding="async"
								onerror={(e) => ((e.currentTarget as HTMLImageElement).style.display = "none")}
							/>
							<div class="absolute inset-0 bg-gradient-to-t from-black/60 via-transparent to-transparent"></div>
						{/if}
						<span
							class="absolute bottom-2 right-2 inline-flex items-center gap-1 rounded-full bg-background/30 backdrop-blur px-2 py-0.5 text-[10px] font-medium text-primary-foreground tabular z-10"
						>
							<Layers class="size-3" />
							{course.source_doc_ids?.length ?? 0}
						</span>
					</div>
					<div class="flex flex-col gap-1 p-3.5">
						<div class="flex items-center gap-1.5">
							<BookOpen class="size-3 text-muted-foreground" />
							<span class="text-[10px] uppercase tracking-wider text-muted-foreground">
								Course
							</span>
						</div>
						<p class="text-sm font-semibold text-foreground line-clamp-2 leading-snug">
							{course.title}
						</p>
						{#if course.description}
							<p class="text-[11px] text-muted-foreground line-clamp-2 leading-relaxed">
								{course.description}
							</p>
						{/if}
					</div>
				</a>
			{/each}
		</div>
	{/if}
</section>
