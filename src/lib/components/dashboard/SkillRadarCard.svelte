<script lang="ts">
	import { onMount } from "svelte";
	import { Chart, registerables } from "chart.js";
	import { BarChart3 } from "@lucide/svelte";
	import DashboardCard from "./DashboardCard.svelte";

	Chart.register(...registerables);

	let {
		labels,
		values,
	}: { labels: string[]; values: number[] } = $props();
	let canvas = $state<HTMLCanvasElement>();

	onMount(() => {
		if (!canvas) return;
		let chart: Chart | undefined;

		const render = () => {
			chart?.destroy();
			const styles = getComputedStyle(document.documentElement);
			const foreground = styles.getPropertyValue("--foreground").trim();
			const muted = styles.getPropertyValue("--muted-foreground").trim();
			const border = styles.getPropertyValue("--border-strong").trim();
			const teal = styles.getPropertyValue("--success").trim();
			chart = new Chart(canvas!, {
				type: "radar",
				data: {
					labels,
					datasets: [{
						label: "Skill level",
						data: values,
						backgroundColor: `color-mix(in oklab, ${teal} 22%, transparent)`,
						borderColor: teal,
						borderWidth: 2,
						pointBackgroundColor: styles.getPropertyValue("--accent").trim(),
						pointBorderColor: teal,
						pointRadius: 3,
						pointHoverRadius: 4,
					}],
				},
				options: {
					responsive: true,
					maintainAspectRatio: false,
					devicePixelRatio: Math.min(Math.max(window.devicePixelRatio, 2), 3),
					animation: matchMedia("(prefers-reduced-motion: reduce)").matches ? false : { duration: 450 },
					plugins: { legend: { display: false }, tooltip: { enabled: true } },
					scales: {
						r: {
							min: 0,
							max: 100,
							ticks: { display: false, stepSize: 20 },
							grid: { color: border },
							angleLines: { color: border },
							pointLabels: { color: muted || foreground, font: { size: 8.5, family: "Inter Variable" } },
						},
					},
				},
			});
		};

		render();
		window.addEventListener("fgb:themechange", render);
		return () => {
			window.removeEventListener("fgb:themechange", render);
			chart?.destroy();
		};
	});

	let summary = $derived(labels.map((label, index) => `${label}: ${values[index] ?? 0}%`).join(", "));
</script>

<DashboardCard title="Skill profile" class="h-full">
	{#snippet action()}
		<span class="inline-flex items-center gap-1 text-[8px] text-muted-foreground"><BarChart3 class="size-3" /> Live profile</span>
	{/snippet}
	<div class="relative h-[118px] px-2 pb-1">
		{#if labels.length}
			<canvas bind:this={canvas} aria-label={`Skill profile. ${summary}`}></canvas>
			<p class="sr-only">{summary}</p>
		{:else}
			<div class="flex h-full items-center justify-center text-[10px] text-muted-foreground">Complete learning activities to build your skill profile.</div>
		{/if}
	</div>
</DashboardCard>
