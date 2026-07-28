<script lang="ts">
	/**
	 * FGB Academy — ShimmerButton.
	 *
	 * Native Svelte 5 reimplementation of Magic UI's `shimmer-button` pattern.
	 * A gold sheen sweeps across the button on hover and every 5s at rest —
	 * a premium moment reserved for hero CTAs (login "Sign in", dashboard
	 * "Chat with Gia"). Respects `data-reduced-motion` and OS prefers-reduced.
	 */
	import { cn } from "$lib/utils.js";
	import type { Snippet } from "svelte";

	type Props = {
		type?: "button" | "submit";
		disabled?: boolean;
		onclick?: (e: MouseEvent) => void;
		class?: string;
		children: Snippet;
	};

	let {
		type = "button",
		disabled = false,
		onclick,
		class: className,
		children,
	}: Props = $props();
</script>

<button
	{type}
	{disabled}
	{onclick}
	class={cn(
		"shimmer-btn relative inline-flex items-center justify-center gap-2 overflow-hidden rounded-full bg-primary px-5 py-2 text-sm font-semibold text-primary-foreground shadow-glow transition-all press disabled:pointer-events-none disabled:opacity-60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50",
		className,
	)}
>
	<span class="relative z-10 inline-flex items-center gap-2">
		{@render children()}
	</span>
	<span class="shimmer-sheen" aria-hidden="true"></span>
</button>

<style>
	.shimmer-btn {
		background-image: linear-gradient(
			110deg,
			var(--primary) 0%,
			var(--primary) 40%,
			oklch(0.5 0.14 245) 50%,
			var(--primary) 60%,
			var(--primary) 100%
		);
		background-size: 220% 100%;
		animation: sheen 5s linear infinite;
	}
	.shimmer-sheen {
		position: absolute;
		inset: 0;
		background: linear-gradient(
			110deg,
			transparent 40%,
			oklch(0.9 0.06 92 / 0.35) 50%,
			transparent 60%
		);
		background-size: 220% 100%;
		background-position: -60% 0;
		animation: sheen 3.5s var(--ease-emphasized) infinite;
		pointer-events: none;
	}
	@keyframes sheen {
		from { background-position: -60% 0; }
		to   { background-position: 160% 0; }
	}
	:global(:root[data-reduced-motion="true"]) .shimmer-btn,
	:global(:root[data-reduced-motion="true"]) .shimmer-sheen {
		animation: none;
	}
	@media (prefers-reduced-motion: reduce) {
		.shimmer-btn, .shimmer-sheen { animation: none; }
	}
</style>
