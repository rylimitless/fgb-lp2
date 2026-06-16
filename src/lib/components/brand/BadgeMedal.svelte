<script lang="ts">
	import { cn } from "$lib/utils.js";
	import type { Snippet } from "svelte";

	type Tier = "bronze" | "silver" | "gold";

	type Props = {
		tier?: Tier;
		label?: string;
		locked?: boolean;
		size?: number;
		class?: string;
		icon?: Snippet;
	};

	let {
		tier = "gold",
		label,
		locked = false,
		size = 88,
		class: className,
		icon,
	}: Props = $props();

	// Hex tile dimensions
	const points = "50,2 95,26 95,74 50,98 5,74 5,26";

	const tierStyles: Record<
		Tier,
		{ fill: string; rim: string; ink: string }
	> = {
		bronze: {
			fill: "oklch(0.66 0.09 60)",
			rim: "oklch(0.46 0.09 50)",
			ink: "oklch(0.99 0 0)",
		},
		silver: {
			fill: "oklch(0.85 0.012 240)",
			rim: "oklch(0.6 0.012 240)",
			ink: "oklch(0.22 0.04 245)",
		},
		gold: {
			fill: "var(--accent)",
			rim: "oklch(0.62 0.1 86)",
			ink: "var(--accent-foreground)",
		},
	};

	let style = $derived(tierStyles[tier]);
</script>

<div
	class={cn("inline-flex flex-col items-center gap-1.5", className)}
	style:width="{size}px"
>
	<div
		class={cn(
			"relative flex items-center justify-center transition-transform duration-300",
			!locked && "drop-shadow-[var(--shadow-glow)]",
			locked && "opacity-40 grayscale",
		)}
		style:width="{size}px"
		style:height="{size}px"
	>
		<svg viewBox="0 0 100 100" width={size} height={size} aria-hidden="true">
			<defs>
				<radialGradient id="medal-shine-{tier}" cx="50%" cy="35%" r="60%">
					<stop offset="0%" stop-color="oklch(1 0 0 / 0.42)" />
					<stop offset="60%" stop-color="oklch(1 0 0 / 0)" />
				</radialGradient>
			</defs>
			<polygon {points} fill={style.fill} stroke={style.rim} stroke-width="2.5" />
			<polygon {points} fill="url(#medal-shine-{tier})" />
		</svg>
		<div
			class="absolute inset-0 flex items-center justify-center"
			style:color={style.ink}
		>
			{#if icon}
				{@render icon()}
			{:else}
				<span class="text-lg font-bold tabular">★</span>
			{/if}
		</div>
	</div>
	{#if label}
		<span class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground text-center">
			{label}
		</span>
	{/if}
</div>
