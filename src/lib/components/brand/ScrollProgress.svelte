<script lang="ts">
	/**
	 * FGB Academy — ScrollProgress.
	 *
	 * A slim gold progress bar fixed at the top of the viewport that reflects
	 * document scroll depth. Meant for long lesson-player modules so learners
	 * always know how much reading remains. Zero motion cost when reduced-
	 * motion is set (bar still updates, no animation).
	 */
	import { onMount } from "svelte";
	import { cn } from "$lib/utils.js";

	type Props = {
		target?: HTMLElement | null;
		class?: string;
	};

	let { target = null, class: className }: Props = $props();

	let progress = $state(0);

	function computeProgress() {
		const el = target ?? document.documentElement;
		const scrollTop = el === document.documentElement
			? window.scrollY
			: el.scrollTop;
		const height =
			(el === document.documentElement
				? document.documentElement.scrollHeight
				: el.scrollHeight) -
			(el === document.documentElement
				? window.innerHeight
				: el.clientHeight);
		if (height <= 0) {
			progress = 0;
			return;
		}
		progress = Math.min(100, Math.max(0, (scrollTop / height) * 100));
	}

	onMount(() => {
		computeProgress();
		const scrollTarget = target ?? window;
		const onScroll = () => computeProgress();
		const onResize = () => computeProgress();
		scrollTarget.addEventListener("scroll", onScroll, { passive: true });
		window.addEventListener("resize", onResize);
		return () => {
			scrollTarget.removeEventListener("scroll", onScroll);
			window.removeEventListener("resize", onResize);
		};
	});
</script>

<div
	class={cn(
		"pointer-events-none fixed inset-x-0 top-0 z-40 h-0.5 bg-transparent",
		className,
	)}
	role="progressbar"
	aria-label="Page scroll progress"
	aria-valuemin="0"
	aria-valuemax="100"
	aria-valuenow={Math.round(progress)}
>
	<div
		class="h-full bg-gradient-to-r from-primary via-accent to-primary transition-[width] duration-100 ease-out"
		style:width="{progress}%"
	></div>
</div>
