<script lang="ts">
	import { cn } from "$lib/utils.js";

	type Props = {
		size?: number;
		duration?: number; // seconds for full sweep
		color?: string;
		class?: string;
	};

	let {
		size = 80,
		duration = 6,
		color = "var(--accent)",
		class: className,
	}: Props = $props();
</script>

<!--
	Border beam — a soft moving highlight that traces the edge of the parent
	rounded container. Inspired by Magic UI's `border-beam`, native to our stack.
	The parent must have `position: relative` and an `overflow: hidden` rounded box.
-->
<div
	aria-hidden="true"
	class={cn("bb-root pointer-events-none absolute inset-0", className)}
	style:--bb-size="{size}px"
	style:--bb-duration="{duration}s"
	style:--bb-color={color}
></div>

<style>
	.bb-root {
		border-radius: inherit;
		mask:
			linear-gradient(#000, #000) content-box,
			linear-gradient(#000, #000);
		-webkit-mask:
			linear-gradient(#000, #000) content-box,
			linear-gradient(#000, #000);
		mask-composite: exclude;
		-webkit-mask-composite: xor;
		padding: 1.5px;
	}
	.bb-root::after {
		content: "";
		position: absolute;
		inset: 0;
		border-radius: inherit;
		background: conic-gradient(
			from 0deg,
			transparent 0deg,
			transparent calc(360deg - var(--bb-size, 80px) * 0.06deg / 1px),
			var(--bb-color, var(--accent)) 360deg
		);
		animation: bb-spin var(--bb-duration, 6s) linear infinite;
	}

	@keyframes bb-spin {
		to {
			transform: rotate(360deg);
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.bb-root::after {
			animation: none;
			opacity: 0.45;
		}
	}
</style>
