<script lang="ts">
	/**
	 * FGB Academy — DotPattern background.
	 *
	 * Native Svelte 5 reimplementation of Magic UI's `dot-pattern`. A subtle
	 * dotted SVG grid for hero surfaces where a full `AnimatedGrid` would be
	 * visual overkill — audit-log heroes, review-queue banners, etc.
	 */
	import { cn } from "$lib/utils.js";

	type Props = {
		gap?: number;
		radius?: number;
		colorClass?: string;
		class?: string;
	};

	let {
		gap = 20,
		radius = 1,
		colorClass = "text-muted-foreground/25",
		class: className,
	}: Props = $props();
</script>

<svg
	class={cn(
		"pointer-events-none absolute inset-0 h-full w-full",
		colorClass,
		className,
	)}
	aria-hidden="true"
>
	<defs>
		<pattern
			id="fgb-dot-pattern"
			x="0"
			y="0"
			width={gap}
			height={gap}
			patternUnits="userSpaceOnUse"
		>
			<circle cx={gap / 2} cy={gap / 2} r={radius} fill="currentColor" />
		</pattern>
	</defs>
	<rect width="100%" height="100%" fill="url(#fgb-dot-pattern)" />
</svg>
