<script lang="ts">
	import { ArrowRight, MessageSquare } from "@lucide/svelte";
	import { goto } from "$app/navigation";

	let {
		firstName,
		resumeHref = "/lesson-player",
	}: { firstName: string; resumeHref?: string } = $props();

	function greeting() {
		const hour = new Date().getHours();
		if (hour < 12) return "Good morning";
		if (hour < 17) return "Good afternoon";
		return "Good evening";
	}
</script>

<section class="hero-card relative min-h-[152px] overflow-hidden rounded-xl border border-primary/35 text-[#edf2f2] shadow-lg">
	<picture aria-hidden="true">
		<source srcset="/brand/hero/dashboard-ambient.webp" type="image/webp" />
		<img src="/brand/hero/dashboard-ambient.png" alt="" class="hero-skyline absolute inset-0 size-full object-cover object-center opacity-50" />
	</picture>
	<div class="absolute inset-0 bg-gradient-to-r from-[#071a2c] via-[#0a2138]/92 to-[#0c3150]/35"></div>
	<div class="hero-aurora absolute -right-20 -top-28 size-72 rounded-full bg-info/25 blur-3xl" aria-hidden="true"></div>
	<div class="hero-grid absolute inset-0 opacity-30 [background-image:linear-gradient(rgba(80,185,205,.12)_1px,transparent_1px),linear-gradient(90deg,rgba(80,185,205,.12)_1px,transparent_1px)] [background-size:32px_32px]" aria-hidden="true"></div>
	<div class="hero-sparks absolute inset-0" aria-hidden="true">
		<span style="--spark-x: 78%; --spark-y: 24%; --spark-delay: 0s"></span>
		<span style="--spark-x: 88%; --spark-y: 58%; --spark-delay: 1.2s"></span>
		<span style="--spark-x: 67%; --spark-y: 74%; --spark-delay: 2.1s"></span>
	</div>
	<div class="hero-content relative z-10 flex min-h-[152px] max-w-[72%] flex-col justify-center px-4 py-4 sm:px-5">
		<p class="academy-kicker">25 years of excellence</p>
		<h1 class="academy-heading mt-1.5 text-[clamp(1.45rem,2.5vw,2.25rem)] leading-tight text-[#f7f1df]">
			{greeting()}, {firstName}
			<span aria-hidden="true" class="hero-wave ml-1 inline-block text-xl">👋</span>
		</h1>
		<p class="mt-1 text-[11px] text-[#dce7ec]/78">Let's continue your learning journey.</p>
		<div class="mt-4 flex flex-wrap gap-2">
			<button class="focus-premium inline-flex h-8 items-center gap-2 rounded-md bg-accent px-3 text-[11px] font-semibold text-accent-foreground shadow-[var(--glow-gold)] transition-colors hover:bg-accent/90" onclick={() => goto("/gia-coach")}>
				<MessageSquare class="size-3.5" /> Chat with Gia
			</button>
			<button class="focus-premium inline-flex h-8 items-center gap-2 rounded-md border border-white/25 bg-[#061523]/55 px-3 text-[11px] font-medium text-[#edf2f2] hover:bg-[#102c46]" onclick={() => goto(resumeHref)}>
				Resume learning <ArrowRight class="size-3.5" />
			</button>
		</div>
	</div>
</section>

<style>
	.hero-card {
		background: linear-gradient(135deg, #0a2239, #0b3858);
	}
	.hero-skyline {
		animation: skyline-drift 18s ease-in-out infinite alternate;
		will-change: transform;
	}
	.hero-aurora {
		animation: aurora-breathe 7s ease-in-out infinite alternate;
	}
	.hero-grid {
		animation: grid-drift 16s linear infinite;
	}
	.hero-content {
		animation: hero-reveal 650ms cubic-bezier(0.22, 1, 0.36, 1) both;
	}
	.hero-wave {
		transform-origin: 70% 70%;
		animation: wave 2.8s ease-in-out 900ms infinite;
	}
	.hero-sparks span {
		position: absolute;
		left: var(--spark-x);
		top: var(--spark-y);
		width: 3px;
		height: 3px;
		border-radius: 999px;
		background: #d9bb70;
		box-shadow: 0 0 10px #d9bb70;
		animation: spark-float 4.5s ease-in-out var(--spark-delay) infinite;
	}
	@keyframes skyline-drift {
		from { transform: scale(1.02) translate3d(0, 0, 0); }
		to { transform: scale(1.065) translate3d(-1.2%, -0.5%, 0); }
	}
	@keyframes aurora-breathe {
		from { opacity: 0.28; transform: translate3d(0, 0, 0) scale(0.92); }
		to { opacity: 0.58; transform: translate3d(-16px, 10px, 0) scale(1.08); }
	}
	@keyframes grid-drift {
		to { background-position: 32px 32px; }
	}
	@keyframes hero-reveal {
		from { opacity: 0; transform: translateY(10px); }
		to { opacity: 1; transform: translateY(0); }
	}
	@keyframes wave {
		0%, 58%, 100% { transform: rotate(0deg); }
		64% { transform: rotate(16deg); }
		70% { transform: rotate(-8deg); }
		76% { transform: rotate(13deg); }
		82% { transform: rotate(0deg); }
	}
	@keyframes spark-float {
		0%, 100% { opacity: 0.2; transform: translateY(5px) scale(0.7); }
		50% { opacity: 0.9; transform: translateY(-7px) scale(1); }
	}
	@media (prefers-reduced-motion: reduce) {
		.hero-skyline,
		.hero-aurora,
		.hero-grid,
		.hero-content,
		.hero-wave,
		.hero-sparks span {
			animation: none;
		}
	}
	:global([data-reduced-motion="true"]) .hero-skyline,
	:global([data-reduced-motion="true"]) .hero-aurora,
	:global([data-reduced-motion="true"]) .hero-grid,
	:global([data-reduced-motion="true"]) .hero-content,
	:global([data-reduced-motion="true"]) .hero-wave,
	:global([data-reduced-motion="true"]) .hero-sparks span {
		animation: none;
	}
</style>
