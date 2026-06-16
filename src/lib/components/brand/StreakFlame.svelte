<script lang="ts">
	import { cn } from "$lib/utils.js";
	import { Flame } from "@lucide/svelte";

	type Props = {
		count?: number;
		class?: string;
		size?: "sm" | "md" | "lg";
	};

	let { count = 0, class: className, size = "md" }: Props = $props();

	let prevCount = $state(0);
	let bumping = $state(false);

	$effect(() => {
		if (count !== prevCount) {
			prevCount = count;
			bumping = true;
			const id = setTimeout(() => (bumping = false), 360);
			return () => clearTimeout(id);
		}
	});

	const sizes = {
		sm: { box: "h-7 px-2 text-xs", icon: "size-3.5" },
		md: { box: "h-9 px-3 text-sm", icon: "size-4" },
		lg: { box: "h-11 px-4 text-base", icon: "size-5" },
	} as const;
	let s = $derived(sizes[size]);

	let isActive = $derived(count > 0);
</script>

<div
	class={cn(
		"inline-flex items-center gap-1.5 rounded-pill border transition-colors",
		isActive
			? "border-streak/30 bg-streak/10 text-streak"
			: "border-border bg-surface-2 text-muted-foreground",
		s.box,
		bumping && "motion-bump",
		className,
	)}
	aria-label="{count} day streak"
>
	<Flame class={cn(s.icon, isActive && "fill-streak/40")} />
	<span class="font-semibold tabular">{count}</span>
	<span class="text-muted-foreground/80 font-medium">
		day{count === 1 ? "" : "s"}
	</span>
</div>
