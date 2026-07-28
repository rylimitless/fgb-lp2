<script lang="ts">
	/**
	 * FGB Academy — CourseCover.
	 *
	 * A drop-in `<img>` replacement for course/module cover art. When the
	 * upstream image is missing or fails to load, renders a deterministic
	 * title-tinted gradient with a `BookOpen` glyph so the tile is never
	 * blank. Used by `+page.svelte` (Continue Learning, Recommended), the
	 * lesson-player catalogue tiles, and the certificate page.
	 *
	 * The tint is derived by hashing `title` → an index in a curated set of
	 * six brand-aligned OKLCH pairs. Same title always renders the same tint,
	 * so the visual is stable across sessions and devices.
	 */
	import { BookOpen } from "@lucide/svelte";
	import { cn } from "$lib/utils.js";

	type Props = {
		title: string;
		image?: string | null;
		alt?: string;
		aspect?: "16/9" | "16/10" | "video" | "square";
		class?: string;
	};

	let {
		title,
		image,
		alt = "",
		aspect = "video",
		class: className,
	}: Props = $props();

	let broken = $state(false);
	let showImage = $derived(!!image && !broken);

	// Six brand-aligned gradient pairs. Every pair is derived from the FGB
	// palette (navy / gold / info / success / streak / accent-soft) and pairs
	// with white-on-dark or navy-on-light foreground text.
	const TINTS: Array<{ from: string; to: string; fg: string }> = [
		{ from: "oklch(0.32 0.12 245)", to: "oklch(0.18 0.04 245)", fg: "oklch(0.98 0.005 240)" },
		{ from: "oklch(0.62 0.13 230)", to: "oklch(0.32 0.12 245)", fg: "oklch(0.98 0.005 230)" },
		{ from: "oklch(0.81 0.07 90)", to: "oklch(0.52 0.09 88)", fg: "oklch(0.22 0.04 90)" },
		{ from: "oklch(0.62 0.16 150)", to: "oklch(0.28 0.09 150)", fg: "oklch(0.98 0.005 150)" },
		{ from: "oklch(0.7 0.18 40)", to: "oklch(0.34 0.12 40)", fg: "oklch(0.98 0.005 40)" },
		{ from: "oklch(0.42 0.13 245)", to: "oklch(0.7 0.14 240)", fg: "oklch(0.98 0.005 240)" },
	];

	function hash(str: string): number {
		let h = 0;
		for (let i = 0; i < str.length; i++) {
			h = (h << 5) - h + str.charCodeAt(i);
			h |= 0;
		}
		return Math.abs(h);
	}

	let tint = $derived(TINTS[hash(title) % TINTS.length]);
	let aspectClass = $derived(
		aspect === "16/10"
			? "aspect-[16/10]"
			: aspect === "square"
				? "aspect-square"
				: "aspect-video",
	);
</script>

<div
	class={cn(
		"relative overflow-hidden bg-surface-2",
		aspectClass,
		className,
	)}
>
	{#if showImage}
		<img
			src={image}
			{alt}
			class="size-full object-cover"
			loading="lazy"
			onerror={() => (broken = true)}
		/>
	{:else}
		<div
			class="flex size-full items-center justify-center p-4"
			style:background="linear-gradient(135deg, {tint.from} 0%, {tint.to} 100%)"
			style:color={tint.fg}
			aria-hidden="true"
		>
			<div class="flex flex-col items-center gap-2 text-center">
				<span
					class="flex size-9 items-center justify-center rounded-2xl bg-white/15 backdrop-blur-sm"
				>
					<BookOpen class="size-4" />
				</span>
				<span
					class="text-xs font-semibold leading-tight line-clamp-2 opacity-90"
					>{title}</span
				>
			</div>
		</div>
		{#if alt}
			<span class="sr-only">{alt}</span>
		{/if}
	{/if}
</div>
