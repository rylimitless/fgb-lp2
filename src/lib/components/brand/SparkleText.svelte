<script lang="ts">
	/**
	 * FGB Academy — SparkleText.
	 *
	 * Native Svelte 5 reimplementation of Magic UI's `sparkles-text`. Wraps a
	 * label with two tiny SVG sparkles that fade + scale on a stagger. Used
	 * only on achievement reveals and completion ceremony sub-headers — high
	 * motion cost is reserved for moments of pride.
	 */
	import { cn } from "$lib/utils.js";

	type Props = {
		text: string;
		class?: string;
	};

	let { text, class: className }: Props = $props();
</script>

<span class={cn("relative inline-flex items-center gap-1 tracking-tight", className)}>
	<span
		class="sparkle sparkle-a"
		aria-hidden="true"
	>
		<svg viewBox="0 0 12 12" class="size-2.5" xmlns="http://www.w3.org/2000/svg">
			<path
				d="M6 0 L7 5 L12 6 L7 7 L6 12 L5 7 L0 6 L5 5 Z"
				fill="var(--accent)"
			/>
		</svg>
	</span>
	<span class="relative">{text}</span>
	<span
		class="sparkle sparkle-b"
		aria-hidden="true"
	>
		<svg viewBox="0 0 12 12" class="size-2" xmlns="http://www.w3.org/2000/svg">
			<path
				d="M6 0 L7 5 L12 6 L7 7 L6 12 L5 7 L0 6 L5 5 Z"
				fill="var(--accent)"
			/>
		</svg>
	</span>
</span>

<style>
	.sparkle {
		display: inline-flex;
		opacity: 0;
		transform: scale(0.5) rotate(0deg);
		animation: twinkle 2.6s var(--ease-emphasized) infinite;
	}
	.sparkle-a { animation-delay: 0s; }
	.sparkle-b { animation-delay: 1.3s; }
	@keyframes twinkle {
		0%, 100% { opacity: 0; transform: scale(0.4) rotate(0deg); }
		30% { opacity: 1; transform: scale(1) rotate(70deg); }
		60% { opacity: 0.7; transform: scale(0.9) rotate(140deg); }
	}
	:global(:root[data-reduced-motion="true"]) .sparkle,
	:global([media~="(prefers-reduced-motion: reduce)"]) .sparkle {
		animation: none;
		opacity: 0.9;
		transform: none;
	}
	@media (prefers-reduced-motion: reduce) {
		.sparkle { animation: none; opacity: 0.9; transform: none; }
	}
</style>
