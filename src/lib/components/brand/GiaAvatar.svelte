<script lang="ts">
	import { cn } from "$lib/utils.js";

	type State = "idle" | "thinking" | "celebrating";

	type Props = {
		state?: State;
		size?: number;
		halo?: boolean;
		pulse?: boolean;
		class?: string;
		alt?: string;
	};

	let {
		state = "idle",
		size = 40,
		halo = true,
		pulse = false,
		class: className,
		alt = "Gia, your FGB Academy coach",
	}: Props = $props();

	const sources: Record<State, string> = {
		idle: "/brand/gia/gia-idle.png",
		thinking: "/brand/gia/gia-thinking.png",
		celebrating: "/brand/gia/gia-celebrating.png",
	};

	let src = $derived(sources[state]);
</script>

<div
	class={cn(
		"relative inline-flex items-center justify-center overflow-hidden rounded-full bg-surface-2",
		halo && "ring-1 ring-primary/30",
		pulse && "motion-glow",
		className,
	)}
	style:width="{size}px"
	style:height="{size}px"
>
	<img
		{src}
		{alt}
		width={size}
		height={size}
		class="h-full w-full object-cover"
		loading="lazy"
		decoding="async"
	/>
	{#if halo}
		<span
			class="pointer-events-none absolute -top-px left-1/2 -translate-x-1/2 size-1.5 rounded-full bg-accent shadow-sm"
			aria-hidden="true"
		></span>
	{/if}
	{#if state === "thinking"}
		<span
			class="absolute bottom-1 right-1 inline-flex h-2 w-2 rounded-full bg-info motion-glow"
			aria-hidden="true"
		></span>
	{/if}
</div>
