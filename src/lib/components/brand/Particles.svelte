<script lang="ts">
	import { cn } from "$lib/utils.js";
	import { prefersReducedMotion } from "svelte/motion";

	type Props = {
		quantity?: number;
		color?: string; // any CSS color the canvas can parse
		minSize?: number;
		maxSize?: number;
		speed?: number;
		class?: string;
	};

	let {
		quantity = 90,
		color = "#d6c47e",
		minSize = 0.6,
		maxSize = 1.8,
		speed = 0.25,
		class: className,
	}: Props = $props();

	let canvas = $state<HTMLCanvasElement>();

	type P = {
		x: number;
		y: number;
		vx: number;
		vy: number;
		size: number;
		alpha: number;
		dAlpha: number;
	};

	// Inspired by Magic UI `particles` + `flickering-grid` (canvas + rAF gated by
	// IntersectionObserver). Native Svelte 5 port. Decorative: aria-hidden.
	$effect(() => {
		const el = canvas;
		if (!el) return;
		const ctx = el.getContext("2d");
		if (!ctx) return;

		const dpr = window.devicePixelRatio || 1;
		let particles: P[] = [];
		let raf = 0;
		let inView = true;
		let w = 0;
		let h = 0;

		function resize() {
			if (!el) return;
			const rect = el.getBoundingClientRect();
			w = rect.width;
			h = rect.height;
			el.width = w * dpr;
			el.height = h * dpr;
			ctx!.setTransform(dpr, 0, 0, dpr, 0, 0);
			seed();
		}

		function seed() {
			particles = Array.from({ length: quantity }, () => ({
				x: Math.random() * w,
				y: Math.random() * h,
				vx: (Math.random() - 0.5) * speed,
				vy: (Math.random() - 0.5) * speed - speed * 0.3,
				size: minSize + Math.random() * (maxSize - minSize),
				alpha: Math.random() * 0.6 + 0.1,
				dAlpha: (Math.random() - 0.5) * 0.01,
			}));
		}

		function draw() {
			ctx!.clearRect(0, 0, w, h);
			for (const p of particles) {
				p.x += p.vx;
				p.y += p.vy;
				p.alpha += p.dAlpha;
				if (p.alpha <= 0.05 || p.alpha >= 0.7) p.dAlpha *= -1;
				// wrap
				if (p.x < -5) p.x = w + 5;
				if (p.x > w + 5) p.x = -5;
				if (p.y < -5) p.y = h + 5;
				if (p.y > h + 5) p.y = -5;
				ctx!.beginPath();
				ctx!.arc(p.x, p.y, p.size, 0, Math.PI * 2);
				ctx!.fillStyle = color;
				ctx!.globalAlpha = p.alpha;
				ctx!.fill();
			}
			ctx!.globalAlpha = 1;
		}

		function loop() {
			if (inView) draw();
			raf = requestAnimationFrame(loop);
		}

		// Reduced motion: draw a single static frame and stop.
		if (prefersReducedMotion.current) {
			resize();
			draw();
			const ro = new ResizeObserver(() => {
				resize();
				draw();
			});
			ro.observe(el);
			return () => ro.disconnect();
		}

		resize();
		const ro = new ResizeObserver(resize);
		ro.observe(el);
		const io = new IntersectionObserver(
			([entry]) => (inView = entry?.isIntersecting ?? true),
			{ threshold: 0 },
		);
		io.observe(el);
		raf = requestAnimationFrame(loop);

		return () => {
			cancelAnimationFrame(raf);
			ro.disconnect();
			io.disconnect();
		};
	});
</script>

<canvas
	bind:this={canvas}
	aria-hidden="true"
	class={cn("pointer-events-none absolute inset-0 size-full", className)}
></canvas>
