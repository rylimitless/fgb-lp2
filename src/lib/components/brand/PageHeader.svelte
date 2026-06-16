<script lang="ts">
	import { cn } from "$lib/utils.js";
	import type { Snippet } from "svelte";
	import BrandLogo from "./BrandLogo.svelte";
	import Spotlight from "./Spotlight.svelte";

	type Props = {
		title: string;
		eyebrow?: string;
		description?: string;
		icon?: Snippet;
		actions?: Snippet;
		variant?: "default" | "hero";
		class?: string;
	};

	let {
		title,
		eyebrow,
		description,
		icon,
		actions,
		variant = "default",
		class: className,
	}: Props = $props();
</script>

<header
	class={cn(
		"relative overflow-hidden rounded-3xl border border-border",
		variant === "hero"
			? "brand-gradient text-primary-foreground px-6 py-8 md:px-10 md:py-10"
			: "bg-gradient-to-br from-card via-card to-surface-2 px-5 py-6 md:px-7 md:py-7 shadow-sm",
		"motion-rise-in",
		className,
	)}
>
	{#if variant === "hero"}
		<Spotlight class="h-full w-full" opacity={0.42} />
		<div
			class="pointer-events-none absolute top-3 right-4 opacity-25"
			aria-hidden="true"
		>
			<BrandLogo variant="mark" size={92} class="text-primary-foreground" />
		</div>
	{:else}
		<!-- gold accent bar + soft spotlight + mark watermark for premium depth -->
		<span class="pointer-events-none absolute left-0 top-0 h-full w-1 bg-gradient-to-b from-accent to-primary/40" aria-hidden="true"></span>
		<Spotlight class="h-1/2 w-1/3" opacity={0.08} />
		<div class="pointer-events-none absolute -top-2 right-3 opacity-[0.06]" aria-hidden="true">
			<BrandLogo variant="mark" size={84} class="text-primary" />
		</div>
	{/if}

	<div class="relative z-10 flex flex-wrap items-end justify-between gap-4">
		<div class="min-w-0 flex flex-col gap-1.5">
			{#if eyebrow}
				<span
					class={cn(
						"text-[11px] font-semibold uppercase tracking-[0.18em]",
						variant === "hero"
							? "text-primary-foreground/70"
							: "text-muted-foreground",
					)}>{eyebrow}</span
				>
			{/if}
			<div class="flex items-center gap-3">
				{#if icon}
					{@render icon()}
				{/if}
				<h1
					class={cn(
						"font-semibold tracking-tight",
						variant === "hero"
							? "text-display-md text-primary-foreground"
							: "text-2xl md:text-[1.7rem] text-foreground",
					)}
				>
					{title}
				</h1>
			</div>
			{#if description}
				<p
					class={cn(
						"max-w-2xl text-sm leading-relaxed",
						variant === "hero"
							? "text-primary-foreground/80"
							: "text-muted-foreground",
					)}
				>
					{description}
				</p>
			{/if}
		</div>
		{#if actions}
			<div class="flex items-center gap-2">
				{@render actions()}
			</div>
		{/if}
	</div>
</header>
