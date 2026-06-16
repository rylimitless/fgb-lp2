<script lang="ts">
	import { cn } from "$lib/utils.js";

	type Props = {
		class?: string;
		// gap between grid lines in px
		size?: number;
		// fade the grid toward edges with a radial mask
		fade?: boolean;
	};

	let { class: className, size = 56, fade = true }: Props = $props();
</script>

<!--
	Animated grid pattern — pure CSS (cheap). Two layered line grids slowly pan;
	a radial mask fades the edges. Inspired by Magic UI animated-grid-pattern /
	retro-grid, native + reduced-motion safe.
-->
<div
	aria-hidden="true"
	class={cn("ag-root pointer-events-none absolute inset-0 overflow-hidden", className)}
	class:ag-fade={fade}
	style:--ag-size="{size}px"
>
	<div class="ag-grid"></div>
	<div class="ag-glow"></div>
</div>

<style>
	.ag-root {
		mask-image: none;
	}
	.ag-fade {
		-webkit-mask-image: radial-gradient(
			ellipse 80% 70% at 50% 40%,
			#000 40%,
			transparent 100%
		);
		mask-image: radial-gradient(
			ellipse 80% 70% at 50% 40%,
			#000 40%,
			transparent 100%
		);
	}
	.ag-grid {
		position: absolute;
		inset: -2px;
		background-image:
			linear-gradient(
				to right,
				color-mix(in oklch, var(--accent) 22%, transparent) 1px,
				transparent 1px
			),
			linear-gradient(
				to bottom,
				color-mix(in oklch, var(--accent) 22%, transparent) 1px,
				transparent 1px
			);
		background-size: var(--ag-size, 56px) var(--ag-size, 56px);
		animation: ag-pan 26s linear infinite;
		opacity: 0.5;
	}
	.ag-glow {
		position: absolute;
		left: 10%;
		top: -20%;
		width: 60%;
		height: 80%;
		background: radial-gradient(
			circle at center,
			color-mix(in oklch, var(--accent) 30%, transparent) 0%,
			transparent 60%
		);
		filter: blur(60px);
		animation: ag-drift 18s ease-in-out infinite alternate;
	}

	@keyframes ag-pan {
		from {
			background-position:
				0 0,
				0 0;
		}
		to {
			background-position:
				var(--ag-size, 56px) var(--ag-size, 56px),
				var(--ag-size, 56px) var(--ag-size, 56px);
		}
	}
	@keyframes ag-drift {
		from {
			transform: translate3d(0, 0, 0);
		}
		to {
			transform: translate3d(30%, 12%, 0);
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.ag-grid,
		.ag-glow {
			animation: none;
		}
	}
</style>
