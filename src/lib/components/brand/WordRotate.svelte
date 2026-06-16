<script lang="ts">
	import { cn } from "$lib/utils.js";
	import { prefersReducedMotion } from "svelte/motion";

	type Props = {
		words: string[];
		interval?: number; // ms per word
		class?: string;
	};

	let { words, interval = 2200, class: className }: Props = $props();

	let index = $state(0);

	$effect(() => {
		if (prefersReducedMotion.current || words.length <= 1) return;
		const id = setInterval(() => {
			index = (index + 1) % words.length;
		}, interval);
		return () => clearInterval(id);
	});

	let current = $derived(words[index] ?? words[0] ?? "");
</script>

<span class={cn("relative inline-grid", className)}>
	{#key current}
		<span class="wr-word col-start-1 row-start-1">{current}</span>
	{/key}
	<!-- invisible widest word reserves width so layout doesn't jump -->
	<span class="invisible col-start-1 row-start-1" aria-hidden="true">
		{words.reduce((a, b) => (b.length > a.length ? b : a), "")}
	</span>
</span>

<style>
	.wr-word {
		display: inline-block;
		animation: wr-in 0.5s cubic-bezier(0.22, 1, 0.36, 1) both;
	}
	@keyframes wr-in {
		from {
			opacity: 0;
			transform: translateY(0.4em) rotateX(-40deg);
		}
		to {
			opacity: 1;
			transform: translateY(0) rotateX(0);
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.wr-word {
			animation: none;
		}
	}
</style>
