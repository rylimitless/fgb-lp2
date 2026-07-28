<script lang="ts">
	import { cn } from "$lib/utils.js";
	import { CheckCircle, MapPin, XCircle, ListChecks } from "@lucide/svelte";

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
		image_url?: string;
		regions?: Region[];
		// AI sometimes generates these alternative shapes:
		hotspots?: any[];
		areas?: any[];
		options?: string[];
		correct?: any; // index, id, or label
		correct_answer?: string;
		hotspot_text?: string;
		content?: string;
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

	let selected = $derived(value);

	// ---- Normalization: extract a consistent `regions` array from whatever
	// shape the AI produced. The prompt asks for {regions: [{label, x, y, correct}]}
	// but the AI frequently generates {hotspots: [...]}, {areas: [...]}, or
	// MC-style {options: [...], correct: <index>} data instead.
	function normalizeRegions(d: Data): Region[] {
		// 1. Proper format: regions array with coordinates.
		if (d.regions?.length) return d.regions as Region[];

		// 2. "hotspots" array — may have x/y coords, or may be text-only.
		if (d.hotspots?.length) {
			return d.hotspots.map((h: any, i: number) => ({
				label: h.label ?? h.text ?? h.id ?? `Option ${i + 1}`,
				x: typeof h.x === "number" ? h.x : undefined,
				y: typeof h.y === "number" ? h.y : undefined,
				correct: resolveCorrect(h, d, i),
			}));
		}

		// 3. "areas" array — MC-like text options.
		if (d.areas?.length) {
			return d.areas.map((a: any, i: number) => ({
				label: a.text ?? a.label ?? `Option ${i + 1}`,
				correct: resolveCorrect(a, d, i),
			}));
		}

		// 4. MC-style options array.
		if (d.options?.length) {
			return d.options.map((opt: string, i: number) => ({
				label: opt,
				correct: resolveCorrect({}, d, i),
			}));
		}

		return [];
	}

	// Resolve whether a given option is the correct one, handling the AI's
	// inconsistent `correct` field shapes (boolean on the region, index on
	// the parent, string id, etc.).
	function resolveCorrect(option: any, parent: any, index: number): boolean {
		if (typeof option.correct === "boolean") return option.correct;
		if (parent.correct === index) return true;
		if (typeof parent.correct === "string") {
			return String(option.id ?? option.label ?? "").toLowerCase() ===
				parent.correct.toLowerCase();
		}
		if (typeof parent.correct === "number") return parent.correct === index;
		return false;
	}

	let regions = $derived(normalizeRegions(data));

	// Detect whether we have a REAL image (URL) or just a text description.
	// The AI often puts a text description of an image in the `image` field
	// instead of an actual URL.
	function hasRealImage(d: Data): boolean {
		const url = d.image_url ?? d.image;
		if (!url || typeof url !== "string") return false;
		// Real URLs start with / or http. Text descriptions don't.
		return url.startsWith("/") || url.startsWith("http");
	}

	let realImage = $derived(hasRealImage(data));
	let imageUrl = $derived(data.image_url ?? data.image ?? "");

	// If no real image and no regions at all, we can't render anything meaningful.
	let hasContent = $derived(regions.length > 0 || data.hotspot_text || data.content);

	function position(region: Region, index: number) {
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

	{#if !hasContent}
		<!-- No regions and no fallback content — nothing to interact with. -->
		<div class="rounded-lg border border-warning/30 bg-warning/5 px-3 py-2">
			<p class="text-xs text-warning">
				This hotspot question has no clickable regions. It may need to be regenerated.
			</p>
		</div>
	{:else if realImage && regions.length > 0}
		<!-- Visual hotspot mode: real image URL with positioned regions. -->
		<div class="grid grid-cols-1 gap-4 md:grid-cols-[minmax(0,1.4fr)_minmax(220px,0.6fr)]">
			<div class="relative overflow-hidden rounded-2xl border border-border bg-surface-2">
				<img
					src={imageUrl}
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
	{:else}
		<!-- Fallback mode: no real image. Render as a list-based selection
		     (like MC) so the question is still answerable. Also show the
		     content/hotspot_text context if the AI provided it. -->
		<div class="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
			<ListChecks class="size-3.5" />
			Text-based hotspot (no image)
		</div>

		{#if data.content}
			<div class="rounded-lg border border-border bg-muted/30 px-3 py-2 text-xs text-foreground/80 leading-relaxed">
				{data.content}
			</div>
		{/if}

		{#if regions.length > 0}
			<div class="flex flex-col gap-2">
				{#each regions as region, i}
					<button
						type="button"
						class={cn(
							"flex items-center gap-2.5 rounded-lg border px-3.5 py-2.5 text-left text-sm transition-colors",
							selected === i
								? "border-primary bg-primary/5 text-primary"
								: "border-border hover:border-muted-foreground/30",
							reveal && isCorrect(i) && "border-success/40 bg-success/5 text-success",
							reveal && selected === i && !isCorrect(i) && "border-destructive/40 bg-destructive/5 text-destructive",
						)}
						disabled={reveal}
						onclick={() => onChange?.(i)}
					>
						<div
							class="size-4 rounded-full border-2 flex items-center justify-center shrink-0 {selected === i ? 'border-primary' : 'border-muted-foreground/30'}"
						>
							{#if selected === i}<div class="size-2 rounded-full bg-primary"></div>{/if}
						</div>
						<span class="text-sm text-foreground">{region.label}</span>
						{#if reveal && isCorrect(i)}
							<CheckCircle class="size-4 ml-auto text-success" />
						{/if}
					</button>
				{/each}
			</div>
		{:else if data.hotspot_text}
			<!-- Single text-clicking hotspot with no options. Show the target text. -->
			<div class="rounded-lg border border-info/30 bg-info/5 px-3 py-2">
				<p class="text-[10px] font-semibold uppercase tracking-wider text-info mb-1">
					Target answer
				</p>
				<p class="text-xs text-foreground/90">{data.hotspot_text}</p>
			</div>
		{/if}
	{/if}
</div>
