<script lang="ts">
	import { cn } from "$lib/utils.js";
	import { prefersReducedMotion } from "svelte/motion";
	import type { Snippet } from "svelte";

	type Props = {
		delay?: number; // ms
		y?: number; // px translate distance
		once?: boolean;
		class?: string;
		children: Snippet;
	};

	let {
		delay = 0,
		y = 24,
		once = true,
		class: className,
		children,
	}: Props = $props();

	let shown = $state(false);

	// IntersectionObserver-driven reveal, via a Svelte 5 action (see Context7:
	// use: action + $effect teardown). Reduced motion shows instantly.
	function reveal(node: HTMLElement) {
		if (prefersReducedMotion.current) {
			shown = true;
			return;
		}
		const io = new IntersectionObserver(
			(entries) => {
				for (const e of entries) {
					if (e.isIntersecting) {
						shown = true;
						if (once) io.unobserve(node);
					} else if (!once) {
						shown = false;
					}
				}
			},
			{ threshold: 0.15, rootMargin: "0px 0px -8% 0px" },
		);
		io.observe(node);
		return { destroy: () => io.disconnect() };
	}
</script>

<div
	use:reveal
	class={cn("sr", className)}
	class:sr-in={shown}
	style:transition-delay="{delay}ms"
	style:--sr-y="{y}px"
>
	{@render children()}
</div>

<style>
	.sr {
		opacity: 0;
		transform: translateY(var(--sr-y, 24px));
		transition:
			opacity 0.6s cubic-bezier(0.22, 1, 0.36, 1),
			transform 0.6s cubic-bezier(0.22, 1, 0.36, 1);
		will-change: opacity, transform;
	}
	.sr-in {
		opacity: 1;
		transform: translateY(0);
	}
	@media (prefers-reduced-motion: reduce) {
		.sr {
			opacity: 1;
			transform: none;
			transition: none;
		}
	}
</style>
