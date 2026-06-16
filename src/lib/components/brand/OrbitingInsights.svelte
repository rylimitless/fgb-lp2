<script lang="ts">
	import { cn } from "$lib/utils.js";
	import { BookOpen, Brain, Sparkles, Target } from "@lucide/svelte";
	import GiaAvatar from "./GiaAvatar.svelte";

	type Props = {
		class?: string;
		size?: number;
	};

	let { class: className, size = 220 }: Props = $props();

	const nodes = [
		{ label: "Sources", icon: BookOpen, delay: "0s" },
		{ label: "Adaptive", icon: Brain, delay: "-3s" },
		{ label: "Insight", icon: Sparkles, delay: "-6s" },
		{ label: "Focus", icon: Target, delay: "-9s" },
	];
</script>

<div
	class={cn("relative flex items-center justify-center", className)}
	style:width="{size}px"
	style:height="{size}px"
	aria-label="Gia coach insight system"
>
	<div class="absolute inset-0 rounded-full border border-border/70"></div>
	<div class="absolute inset-8 rounded-full border border-accent/20"></div>
	<div class="relative z-10 rounded-full bg-background p-2 shadow-md">
		<GiaAvatar size={72} pulse />
	</div>

	{#each nodes as node}
		<div
			class="orbit"
			style:--orbit-size="{size}px"
			style:animation-delay={node.delay}
		>
			<div class="node">
				<node.icon class="size-4" />
				<span class="sr-only">{node.label}</span>
			</div>
		</div>
	{/each}
</div>

<style>
	.orbit {
		position: absolute;
		inset: 0;
		animation: orbit 12s linear infinite;
		transform-origin: center;
	}
	.node {
		position: absolute;
		left: 50%;
		top: 0;
		display: flex;
		width: 2.4rem;
		height: 2.4rem;
		transform: translateX(-50%);
		align-items: center;
		justify-content: center;
		border-radius: 9999px;
		border: 1px solid color-mix(in oklch, var(--accent) 35%, transparent);
		background: color-mix(in oklch, var(--background) 92%, transparent);
		color: var(--primary);
		box-shadow: var(--shadow-md);
		backdrop-filter: blur(12px);
	}
	@keyframes orbit {
		to {
			transform: rotate(360deg);
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.orbit {
			animation: none;
		}
		.orbit:nth-of-type(3) { transform: rotate(90deg); }
		.orbit:nth-of-type(4) { transform: rotate(180deg); }
		.orbit:nth-of-type(5) { transform: rotate(270deg); }
	}
</style>
