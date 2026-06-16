<script lang="ts">
	import { cn } from "$lib/utils.js";
	import NumberTicker from "./NumberTicker.svelte";
	import type { Snippet } from "svelte";

	type Tone = "neutral" | "primary" | "success" | "warning" | "info" | "destructive" | "accent";

	type Props = {
		label: string;
		value: number;
		tone?: Tone;
		hint?: string;
		decimals?: number;
		suffix?: string;
		prefix?: string;
		icon?: Snippet;
		trend?: number; // delta as percentage; positive = up, negative = down
		class?: string;
	};

	let {
		label,
		value,
		tone = "neutral",
		hint,
		decimals = 0,
		suffix = "",
		prefix = "",
		icon,
		trend,
		class: className,
	}: Props = $props();

	const toneStyles: Record<Tone, { value: string; chip: string }> = {
		neutral: {
			value: "text-foreground",
			chip: "bg-muted text-muted-foreground",
		},
		primary: {
			value: "text-primary",
			chip: "bg-primary/10 text-primary",
		},
		success: {
			value: "text-success",
			chip: "bg-success/10 text-success",
		},
		warning: {
			value: "text-warning",
			chip: "bg-warning/10 text-warning",
		},
		info: {
			value: "text-info",
			chip: "bg-info/10 text-info",
		},
		destructive: {
			value: "text-destructive",
			chip: "bg-destructive/10 text-destructive",
		},
		accent: {
			value: "text-foreground",
			chip: "bg-accent-soft text-accent-foreground",
		},
	};

	let t = $derived(toneStyles[tone]);
	let trendUp = $derived(typeof trend === "number" && trend > 0);
	let trendDown = $derived(typeof trend === "number" && trend < 0);
</script>

<div
	class={cn(
		"group relative flex flex-col gap-2 rounded-2xl border border-border bg-card p-4 lift press overflow-hidden",
		className,
	)}
>
	<div class="flex items-center justify-between gap-2">
		<div class="flex items-center gap-2 min-w-0">
			{#if icon}
				<span class={cn("flex size-7 items-center justify-center rounded-full", t.chip)}>
					{@render icon()}
				</span>
			{/if}
			<span class="truncate text-xs text-muted-foreground">{label}</span>
		</div>
		{#if typeof trend === "number"}
			<span
				class={cn(
					"inline-flex items-center gap-0.5 rounded-full px-1.5 py-0.5 text-[10px] font-semibold tabular",
					trendUp && "bg-success/10 text-success",
					trendDown && "bg-destructive/10 text-destructive",
					!trendUp && !trendDown && "bg-muted text-muted-foreground",
				)}
				title="vs. previous period"
			>
				{trendUp ? "▲" : trendDown ? "▼" : "▬"}
				{Math.abs(trend).toFixed(1)}%
			</span>
		{/if}
	</div>
	<p class={cn("text-display-md font-bold leading-none tracking-tight", t.value)}>
		<NumberTicker {value} {decimals} {suffix} {prefix} />
	</p>
	{#if hint}
		<p class="text-[11px] text-muted-foreground/90">{hint}</p>
	{/if}
	<!-- Hover gold underline -->
	<span
		class="pointer-events-none absolute inset-x-4 bottom-0 h-px scale-x-0 origin-left bg-accent transition-transform duration-300 group-hover:scale-x-100"
		aria-hidden="true"
	></span>
</div>
