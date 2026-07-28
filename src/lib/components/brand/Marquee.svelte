<script lang="ts">
	import { cn } from "$lib/utils.js";
	import type { Snippet } from "svelte";

	type Props = {
		speed?: number; // pixels per second
		pauseOnHover?: boolean;
		fade?: boolean;
		class?: string;
		children: Snippet;
	};

	let {
		speed = 28,
		pauseOnHover = true,
		fade = true,
		class: className,
		children,
	}: Props = $props();
</script>

<!--
	Lightweight marquee inspired by Magic UI's `marquee`, native to our stack.
	Renders the children twice in a row, then loops the whole strip horizontally.
-->
<!-- aria-live="off" so screen readers don't announce the endlessly scrolling
	 content — the marquee is decorative discovery ribbon, not live status.
	 See ACCESSIBILITY.md §4.2. -->
<div
	class={cn(
		"mq-root relative w-full overflow-hidden",
		fade && "mq-fade",
		className,
	)}
	style:--mq-speed="{speed}s"
	data-pause-hover={pauseOnHover}
	aria-live="off"
	role="marquee"
>
	<div class="mq-track">
		<div class="mq-group">{@render children()}</div>
		<div class="mq-group" aria-hidden="true">{@render children()}</div>
	</div>
</div>

<style>
	.mq-track {
		display: flex;
		width: max-content;
		animation: mq-slide var(--mq-speed, 28s) linear infinite;
		gap: 1.5rem;
	}
	.mq-group {
		display: flex;
		gap: 1.5rem;
		flex-shrink: 0;
	}
	[data-pause-hover="true"]:hover .mq-track {
		animation-play-state: paused;
	}
	.mq-fade::before,
	.mq-fade::after {
		content: "";
		position: absolute;
		top: 0;
		bottom: 0;
		width: 4rem;
		z-index: 1;
		pointer-events: none;
	}
	.mq-fade::before {
		left: 0;
		background: linear-gradient(to right, var(--background), transparent);
	}
	.mq-fade::after {
		right: 0;
		background: linear-gradient(to left, var(--background), transparent);
	}

	@keyframes mq-slide {
		from { transform: translateX(0); }
		to   { transform: translateX(-50%); }
	}

	@media (prefers-reduced-motion: reduce) {
		.mq-track {
			animation: none;
		}
	}
</style>
