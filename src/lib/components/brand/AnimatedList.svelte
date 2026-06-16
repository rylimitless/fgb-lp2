<script lang="ts" generics="T">
	import { cn } from "$lib/utils.js";
	import type { Snippet } from "svelte";

	type Props<T> = {
		items: T[];
		getKey?: (item: T, index: number) => string | number;
		class?: string;
		children: Snippet<[T, number]>;
	};

	let {
		items,
		getKey = (_item: T, index: number) => index,
		class: className,
		children,
	}: Props<T> = $props();
</script>

<ul class={cn("flex flex-col", className)}>
	{#each items as item, i (getKey(item, i))}
		<li class="al-item border-b border-border/40 last:border-0" style:--al-delay="{i * 55}ms">
			{@render children(item, i)}
		</li>
	{/each}
</ul>

<style>
	.al-item {
		opacity: 0;
		transform: translateX(-8px) scale(0.985);
		animation: al-in 0.45s cubic-bezier(0.22, 1, 0.36, 1) forwards;
		animation-delay: var(--al-delay, 0ms);
	}

	@keyframes al-in {
		to {
			opacity: 1;
			transform: translateX(0) scale(1);
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.al-item {
			opacity: 1;
			transform: none;
			animation: none;
		}
	}
</style>
