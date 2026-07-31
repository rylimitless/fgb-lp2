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
		state: avatarState = "idle",
		size = 40,
		halo = true,
		pulse = false,
		class: className,
		alt = "Gia, your FGB Academy coach",
	}: Props = $props();

	const sources: Record<State, string> = {
		idle: "/brand/gia/gia-idle.webp",
		thinking: "/brand/gia/gia-idle.webp",
		celebrating: "/brand/gia/gia-idle.webp",
	};

	let src = $derived(sources[avatarState]);
	let imageFailed = $state(false);
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
	<div class="absolute inset-0 flex items-center justify-center bg-gradient-to-br from-[#f2d5af] via-[#dca77d] to-[#0b3151]" aria-hidden="true">
		<span class="academy-heading rounded-full border border-white/35 bg-[#09243d]/85 px-2 py-1 text-[clamp(9px,18%,14px)] text-[#f7e7c0]">Gia</span>
	</div>
	{#if !imageFailed}
		<img
			{src}
			{alt}
			width={size}
			height={size}
			class="relative h-full w-full object-cover"
			loading="lazy"
			decoding="async"
			onerror={() => (imageFailed = true)}
		/>
	{:else}
		<span class="sr-only">{alt}</span>
	{/if}
	{#if halo}
		<span
			class="pointer-events-none absolute -top-px left-1/2 -translate-x-1/2 size-1.5 rounded-full bg-accent shadow-sm"
			aria-hidden="true"
		></span>
	{/if}
	{#if avatarState === "thinking"}
		<span
			class="absolute bottom-1 right-1 inline-flex h-2 w-2 rounded-full bg-info motion-glow"
			aria-hidden="true"
		></span>
	{/if}
</div>
