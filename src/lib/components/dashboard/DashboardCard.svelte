<script lang="ts">
	import type { Snippet } from "svelte";
	import { cn } from "$lib/utils";

	let {
		title,
		kicker,
		action,
		children,
		class: className,
	}: {
		title?: string;
		kicker?: string;
		action?: Snippet;
		children: Snippet;
		class?: string;
	} = $props();
</script>

<section class={cn("academy-card min-w-0 overflow-hidden", className)}>
	{#if title || kicker || action}
		<header class="flex min-h-11 items-center justify-between gap-3 border-b border-border px-4 py-2.5">
			<div class="min-w-0">
				{#if kicker}<p class="academy-kicker">{kicker}</p>{/if}
				{#if title}<h2 class="truncate text-sm font-semibold uppercase tracking-[0.12em] text-foreground">{title}</h2>{/if}
			</div>
			{#if action}{@render action()}{/if}
		</header>
	{/if}
	{@render children()}
</section>
