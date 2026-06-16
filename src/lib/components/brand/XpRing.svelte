<script lang="ts">
	import { cn } from "$lib/utils.js";
	import type { Snippet } from "svelte";

	type Props = {
		value?: number; // 0..100
		size?: number;
		stroke?: number;
		ring?: "primary" | "accent" | "success" | "streak";
		label?: string;
		class?: string;
		center?: Snippet;
	};

	let {
		value = 0,
		size = 96,
		stroke = 8,
		ring = "primary",
		label,
		class: className,
		center,
	}: Props = $props();

	const ringColor: Record<NonNullable<Props["ring"]>, string> = {
		primary: "var(--primary)",
		accent: "var(--accent)",
		success: "var(--success)",
		streak: "var(--streak)",
	};

	let pct = $derived(Math.max(0, Math.min(100, value)));
	let radius = $derived((size - stroke) / 2);
	let circumference = $derived(2 * Math.PI * radius);
	let dash = $derived((pct / 100) * circumference);
	let color = $derived(ringColor[ring]);
</script>

<div class={cn("inline-flex flex-col items-center gap-1.5", className)}>
	<div class="relative" style:width="{size}px" style:height="{size}px">
		<svg
			width={size}
			height={size}
			viewBox="0 0 {size} {size}"
			aria-hidden="true"
		>
			<circle
				cx={size / 2}
				cy={size / 2}
				r={radius}
				stroke="var(--border)"
				stroke-width={stroke}
				fill="none"
			/>
			<circle
				cx={size / 2}
				cy={size / 2}
				r={radius}
				stroke={color}
				stroke-width={stroke}
				stroke-linecap="round"
				fill="none"
				stroke-dasharray="{dash} {circumference}"
				transform="rotate(-90 {size / 2} {size / 2})"
				style="transition: stroke-dasharray var(--dur-slow) var(--ease-out-quart);"
			/>
		</svg>
		<div class="absolute inset-0 flex items-center justify-center">
			{#if center}
				{@render center()}
			{:else}
				<span class="text-sm font-semibold text-foreground tabular">
					{Math.round(pct)}%
				</span>
			{/if}
		</div>
	</div>
	{#if label}
		<span class="text-[11px] font-medium uppercase tracking-wider text-muted-foreground">
			{label}
		</span>
	{/if}
</div>
