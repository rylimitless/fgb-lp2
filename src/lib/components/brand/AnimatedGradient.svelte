<script lang="ts">
	import { cn } from "$lib/utils.js";

	type Props = {
		variant?: "primary" | "subtle" | "celebration";
		class?: string;
	};

	let { variant = "primary", class: className }: Props = $props();
</script>

<!--
	Animated brand gradient background.
	Inspired by Magic UI's `animated-gradient-background` pattern, native to our stack.
	Two slowly-drifting radial gradients on top of a base colour. Decorative only:
	aria-hidden, pointer-events: none, and disabled by prefers-reduced-motion.
-->
<div
	aria-hidden="true"
	class={cn(
		"pointer-events-none absolute inset-0 overflow-hidden",
		className,
	)}
	data-variant={variant}
>
	<div class="abg-base"></div>
	<div class="abg-blob abg-blob-a"></div>
	<div class="abg-blob abg-blob-b"></div>
	<div class="abg-grain"></div>
</div>

<style>
	[data-variant="primary"] .abg-base {
		background: radial-gradient(
				120% 90% at 0% 0%,
				oklch(0.32 0.12 245) 0%,
				oklch(0.18 0.04 245) 60%,
				oklch(0.15 0.03 245) 100%
			);
	}
	[data-variant="subtle"] .abg-base {
		background: linear-gradient(
			135deg,
			var(--primary-soft) 0%,
			var(--accent-soft) 100%
		);
	}
	[data-variant="celebration"] .abg-base {
		background: radial-gradient(
				120% 90% at 50% 0%,
				oklch(0.36 0.14 245) 0%,
				oklch(0.18 0.04 245) 70%
			);
	}

	.abg-base,
	.abg-blob,
	.abg-grain {
		position: absolute;
		inset: 0;
	}

	.abg-blob {
		border-radius: 9999px;
		filter: blur(60px);
		opacity: 0.55;
		will-change: transform, opacity;
	}
	.abg-blob-a {
		width: 60%;
		height: 60%;
		left: -10%;
		top: -10%;
		background: var(--accent);
		animation: abg-drift-a 18s ease-in-out infinite alternate;
	}
	.abg-blob-b {
		width: 70%;
		height: 70%;
		right: -15%;
		bottom: -20%;
		background: var(--primary);
		opacity: 0.45;
		animation: abg-drift-b 22s ease-in-out infinite alternate;
	}

	/* Optional subtle dot grain — purely aesthetic */
	.abg-grain {
		background-image: radial-gradient(
			oklch(1 0 0 / 0.06) 1px,
			transparent 1px
		);
		background-size: 22px 22px;
		opacity: 0.4;
		mix-blend-mode: overlay;
	}

	[data-variant="subtle"] .abg-blob {
		opacity: 0.35;
	}
	[data-variant="subtle"] .abg-grain {
		display: none;
	}

	@keyframes abg-drift-a {
		from { transform: translate3d(0, 0, 0) scale(1); }
		to   { transform: translate3d(8%, 6%, 0) scale(1.12); }
	}
	@keyframes abg-drift-b {
		from { transform: translate3d(0, 0, 0) scale(1); }
		to   { transform: translate3d(-6%, -8%, 0) scale(1.08); }
	}

	@media (prefers-reduced-motion: reduce) {
		.abg-blob {
			animation: none;
		}
	}
</style>
