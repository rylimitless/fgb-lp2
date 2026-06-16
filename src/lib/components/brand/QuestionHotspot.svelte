<script lang="ts">
	import { cn } from "$lib/utils.js";
	import { CheckCircle, MapPin, XCircle } from "@lucide/svelte";

	type Region = {
		label: string;
		x?: number; // percentage 0..100
		y?: number;
		w?: number;
		h?: number;
		correct?: boolean;
	};

	type Data = {
		question?: string;
		image?: string;
		regions?: Region[];
		explanation?: string;
	};

	type Props = {
		data: Data;
		value?: number;
		reveal?: boolean;
		onChange?: (value: number) => void;
		class?: string;
	};

	let {
		data,
		value,
		reveal = false,
		onChange,
		class: className,
	}: Props = $props();

	let regions = $derived(data.regions ?? []);
	let selected = $derived(value);

	function position(region: Region, index: number) {
		// AI can emit coordinates, but if it doesn't, place markers in a stable
		// diagonal arc so label-driven hotspots still work.
		const x = clamp(region.x ?? 18 + ((index * 23) % 64), 8, 92);
		const y = clamp(region.y ?? 22 + ((index * 17) % 56), 10, 90);
		return `left: ${x}%; top: ${y}%;`;
	}

	function clamp(n: number, min: number, max: number) {
		return Math.max(min, Math.min(max, n));
	}

	function isCorrect(index: number): boolean {
		return Boolean(regions[index]?.correct);
	}

	let selectedRegion = $derived(
		typeof selected === "number" ? regions[selected] : undefined,
	);
</script>

<div class={cn("space-y-4", className)}>
	{#if data.question}
		<p class="text-sm font-medium text-foreground">{data.question}</p>
	{/if}

	<div class="grid grid-cols-1 gap-4 md:grid-cols-[minmax(0,1.4fr)_minmax(220px,0.6fr)]">
		<div class="relative overflow-hidden rounded-2xl border border-border bg-surface-2">
			<img
				src={data.image || "/brand/questions/hotspot-cyber-risk.png"}
				alt=""
				class="aspect-video w-full object-cover"
				loading="lazy"
				decoding="async"
				onerror={(e) => ((e.currentTarget as HTMLImageElement).src = "/brand/pathways/cybersecurity.png")}
			/>
			<div class="absolute inset-0 bg-gradient-to-t from-background/45 via-transparent to-transparent"></div>
			{#each regions as region, i}
				<button
					type="button"
					class={cn(
						"absolute flex -translate-x-1/2 -translate-y-1/2 items-center gap-1 rounded-full border px-2.5 py-1 text-xs font-semibold shadow-md backdrop-blur transition-all focus-visible:ring-2 focus-visible:ring-ring/40",
						selected === i
							? "border-accent bg-accent text-accent-foreground scale-105"
							: "border-background/60 bg-background/75 text-foreground hover:bg-background",
						reveal && isCorrect(i) && "border-success bg-success text-success-foreground",
						reveal && selected === i && !isCorrect(i) && "border-destructive bg-destructive text-destructive-foreground",
					)}
					style={position(region, i)}
					aria-pressed={selected === i}
					aria-label={`Select hotspot: ${region.label}`}
					disabled={reveal}
					onclick={() => onChange?.(i)}
				>
					{#if reveal && isCorrect(i)}
						<CheckCircle class="size-3.5" />
					{:else if reveal && selected === i && !isCorrect(i)}
						<XCircle class="size-3.5" />
					{:else}
						<MapPin class="size-3.5" />
					{/if}
					<span class="hidden sm:inline">{region.label}</span>
				</button>
			{/each}
		</div>

		<div class="rounded-2xl border border-border bg-card p-4">
			<p class="text-[10px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">
				Hotspots
			</p>
			<div class="mt-3 flex flex-col gap-2">
				{#each regions as region, i}
					<button
						type="button"
						class={cn(
							"flex items-center justify-between gap-2 rounded-xl border px-3 py-2 text-left text-sm transition-colors",
							selected === i ? "border-primary bg-primary/5 text-primary" : "border-border hover:border-primary/30 hover:bg-muted/30",
							reveal && isCorrect(i) && "border-success/40 bg-success/5 text-success",
							reveal && selected === i && !isCorrect(i) && "border-destructive/40 bg-destructive/5 text-destructive",
						)}
						disabled={reveal}
						onclick={() => onChange?.(i)}
					>
						<span>{region.label}</span>
						{#if reveal && isCorrect(i)}
							<CheckCircle class="size-4" />
						{/if}
					</button>
				{/each}
			</div>
			{#if selectedRegion}
				<p class="mt-3 text-xs text-muted-foreground">
					Selected: <span class="font-medium text-foreground">{selectedRegion.label}</span>
				</p>
			{/if}
		</div>
	</div>
</div>
