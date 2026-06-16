<script lang="ts">
	import { prefersReducedMotion } from "svelte/motion";

	type Props = {
		count?: number;
		duration?: number; // ms
		colors?: string[];
		trigger?: number; // change to re-fire
	};

	let {
		count = 80,
		duration = 2200,
		colors = [
			"var(--accent)",
			"var(--primary)",
			"var(--success)",
			"var(--info)",
			"oklch(0.85 0.012 240)", // silver
		],
		trigger = 0,
	}: Props = $props();

	type Piece = {
		x: number;
		drift: number;
		spin: number;
		size: number;
		color: string;
		delay: number;
		shape: "rect" | "circle" | "wedge";
	};

	let pieces = $state<Piece[]>([]);
	let active = $state(false);
	let nonce = $state(0);

	function makeBatch(): Piece[] {
		const out: Piece[] = [];
		for (let i = 0; i < count; i++) {
			const x = (i / count) * 100 + (Math.random() - 0.5) * 4;
			const drift = (Math.random() - 0.5) * 220;
			const spin = 360 + Math.random() * 540 * (Math.random() < 0.5 ? -1 : 1);
			const size = 6 + Math.random() * 8;
			const color = colors[Math.floor(Math.random() * colors.length)] ?? "var(--accent)";
			const delay = Math.random() * 350;
			const shapes: Piece["shape"][] = ["rect", "rect", "circle", "wedge"];
			const shape = shapes[Math.floor(Math.random() * shapes.length)] ?? "rect";
			out.push({ x, drift, spin, size, color, delay, shape });
		}
		return out;
	}

	function fire() {
		if (prefersReducedMotion.current) return;
		nonce++;
		pieces = makeBatch();
		active = true;
		const id = setTimeout(() => {
			active = false;
			pieces = [];
		}, duration + 500);
		return () => clearTimeout(id);
	}

	// Fires once on mount AND whenever the `trigger` prop changes.
	$effect(() => {
		void trigger;
		const cleanup = fire();
		return cleanup;
	});
</script>

<!-- Pointer-events none, decorative only, full viewport overlay -->
<div
	aria-hidden="true"
	class="confetti-root pointer-events-none fixed inset-0 overflow-hidden z-50"
	data-active={active}
	data-nonce={nonce}
>
	{#if active}
		{#each pieces as p, i (i + "-" + nonce)}
			<span
				class="confetti-piece"
				style="
					left: {p.x}%;
					--drift: {p.drift}px;
					--spin: {p.spin}deg;
					width: {p.size}px;
					height: {p.size * (p.shape === 'rect' ? 0.5 : 1)}px;
					background: {p.color};
					animation-duration: {duration}ms;
					animation-delay: {p.delay}ms;
					border-radius: {p.shape === 'circle' ? '999px' : p.shape === 'wedge' ? '0 50% 0 50%' : '2px'};
				"
			></span>
		{/each}
	{/if}
</div>

<style>
	.confetti-piece {
		position: absolute;
		top: -10vh;
		transform: translate3d(0, -10vh, 0) rotate(0deg);
		animation-name: fgb-confetti-fall;
		animation-timing-function: cubic-bezier(0.25, 1, 0.5, 1);
		animation-fill-mode: forwards;
		opacity: 0.95;
	}

	@media (prefers-reduced-motion: reduce) {
		.confetti-root {
			display: none;
		}
	}
</style>
