<script lang="ts">
	import { cn } from "$lib/utils.js";
	import GiaAvatar from "./GiaAvatar.svelte";
	import type { Snippet } from "svelte";

	type Props = {
		title: string;
		description?: string;
		gia?: boolean;
		giaState?: "idle" | "thinking" | "celebrating";
		class?: string;
		action?: Snippet;
		icon?: Snippet;
	};

	let {
		title,
		description,
		gia = true,
		giaState = "idle",
		class: className,
		action,
		icon,
	}: Props = $props();
</script>

<div
	class={cn(
		"flex flex-col items-center justify-center gap-4 rounded-2xl border border-border bg-surface-1 px-6 py-10 text-center motion-rise-in",
		className,
	)}
>
	{#if icon}
		<div class="text-muted-foreground/50">
			{@render icon()}
		</div>
	{:else if gia}
		<GiaAvatar state={giaState} size={72} />
	{/if}
	<div class="flex flex-col gap-1.5 max-w-sm">
		<p class="text-base font-semibold text-foreground">{title}</p>
		{#if description}
			<p class="text-sm text-muted-foreground leading-relaxed">
				{description}
			</p>
		{/if}
	</div>
	{#if action}
		<div class="mt-2">
			{@render action()}
		</div>
	{/if}
</div>
