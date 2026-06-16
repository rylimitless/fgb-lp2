<script lang="ts">
	import { cn } from "$lib/utils.js";
	import { Tween } from "svelte/motion";
	import { prefersReducedMotion } from "svelte/motion";
	import { cubicOut } from "svelte/easing";

	type Props = {
		value: number;
		duration?: number;
		decimals?: number;
		suffix?: string;
		prefix?: string;
		class?: string;
	};

	let {
		value,
		duration = 900,
		decimals = 0,
		suffix = "",
		prefix = "",
		class: className,
	}: Props = $props();

	const tween = new Tween(0, { duration: 900, easing: cubicOut });

	$effect(() => {
		const target = Number.isFinite(value) ? value : 0;
		if (prefersReducedMotion.current) {
			tween.set(target, { duration: 0 });
		} else {
			tween.set(target, { duration, easing: cubicOut });
		}
	});

	let formatted = $derived(
		(prefix ?? "") +
			tween.current.toLocaleString(undefined, {
				minimumFractionDigits: decimals,
				maximumFractionDigits: decimals,
			}) +
			(suffix ?? ""),
	);
</script>

<span class={cn("tabular", className)} aria-label={`${value}${suffix ?? ""}`}>
	{formatted}
</span>
