<script lang="ts">
	import { cn } from "$lib/utils.js";
	import { Check, Link2, X } from "@lucide/svelte";

	type Pair = {
		left: string;
		right: string;
	};

	type Data = {
		question?: string;
		pairs?: Pair[];
		explanation?: string;
	};

	type Props = {
		data: Data;
		value?: Record<string, number>;
		reveal?: boolean;
		onChange?: (value: Record<string, number>) => void;
		class?: string;
	};

	let {
		data,
		value = {},
		reveal = false,
		onChange,
		class: className,
	}: Props = $props();

	let selectedLeft = $state<number | null>(null);

	let pairs = $derived(data.pairs ?? []);
	let matches = $derived(value ?? {});

	// Deterministic right-side shuffle so matching isn't trivial, but SSR and
	// hydration remain stable. Keeps the original pair index for scoring.
	let rightItems = $derived(
		pairs
			.map((pair, index) => ({ label: pair.right, index }))
			.sort((a, b) => ((a.index * 7 + 3) % 11) - ((b.index * 7 + 3) % 11)),
	);

	function setMatch(leftIndex: number, rightIndex: number) {
		const next: Record<string, number> = { ...matches };

		// One-to-one mapping: remove any previous left assigned to this right.
		for (const key of Object.keys(next)) {
			if (next[key] === rightIndex) delete next[key];
		}
		next[String(leftIndex)] = rightIndex;
		onChange?.(next);
		selectedLeft = null;
	}

	function clearMatch(leftIndex: number) {
		const next: Record<string, number> = { ...matches };
		delete next[String(leftIndex)];
		onChange?.(next);
	}

	function rightFor(leftIndex: number): number | undefined {
		return matches[String(leftIndex)];
	}

	function isRightUsed(rightIndex: number): boolean {
		return Object.values(matches).includes(rightIndex);
	}

	function leftIsCorrect(leftIndex: number): boolean {
		return rightFor(leftIndex) === leftIndex;
	}

	function rightIsCorrect(rightIndex: number): boolean {
		return matches[String(rightIndex)] === rightIndex;
	}
</script>

<div class={cn("space-y-4", className)}>
	{#if data.question}
		<p class="text-sm font-medium text-foreground">{data.question}</p>
	{/if}

	<div class="grid grid-cols-1 gap-4 md:grid-cols-[1fr_auto_1fr] md:items-start">
		<div class="space-y-2">
			<p class="text-[10px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">
				Terms
			</p>
			{#each pairs as pair, i}
				<div
					class={cn(
						"flex items-center gap-2 rounded-xl border bg-card p-2.5 transition-colors",
						selectedLeft === i && "border-primary bg-primary/5",
						reveal && leftIsCorrect(i) && "border-success/40 bg-success/5",
						reveal && rightFor(i) !== undefined && !leftIsCorrect(i) && "border-destructive/40 bg-destructive/5",
					)}
				>
					<button
						type="button"
						class="min-h-9 flex-1 rounded-lg px-2 text-left text-sm text-foreground outline-none transition-colors hover:bg-muted focus-visible:ring-2 focus-visible:ring-ring/30"
						aria-pressed={selectedLeft === i}
						onclick={() => (selectedLeft = selectedLeft === i ? null : i)}
					>
						{pair.left}
					</button>
					{#if rightFor(i) !== undefined}
						<span class="hidden max-w-32 truncate text-xs text-muted-foreground sm:inline">
							{pairs[rightFor(i) ?? 0]?.right}
						</span>
						<button
							type="button"
							class="flex size-7 items-center justify-center rounded-full text-muted-foreground hover:bg-muted hover:text-foreground"
							aria-label={`Clear match for ${pair.left}`}
							onclick={() => clearMatch(i)}
							disabled={reveal}
						>
							<X class="size-3.5" />
						</button>
					{/if}
				</div>
			{/each}
		</div>

		<div class="hidden pt-8 text-muted-foreground md:block" aria-hidden="true">
			<Link2 class="size-5" />
		</div>

		<div class="space-y-2">
			<p class="text-[10px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">
				Definitions
			</p>
			{#each rightItems as right}
				<button
					type="button"
					class={cn(
						"min-h-11 w-full rounded-xl border bg-surface-1 px-3 py-2 text-left text-sm transition-all focus-visible:ring-2 focus-visible:ring-ring/30",
						selectedLeft !== null && !reveal && "hover:border-primary/40 hover:bg-primary/5",
						isRightUsed(right.index) && "border-primary/30 bg-primary/5",
						reveal && rightIsCorrect(right.index) && "border-success/40 bg-success/5 text-success",
						reveal && isRightUsed(right.index) && !rightIsCorrect(right.index) && "border-destructive/40 bg-destructive/5 text-destructive",
					)}
					disabled={selectedLeft === null || reveal}
					onclick={() => selectedLeft !== null && setMatch(selectedLeft, right.index)}
				>
					<span class="inline-flex items-start gap-2">
						{#if reveal && rightIsCorrect(right.index)}
							<Check class="mt-0.5 size-3.5 shrink-0" />
						{/if}
						<span>{right.label}</span>
					</span>
				</button>
			{/each}
		</div>
	</div>

	<p class="text-xs text-muted-foreground">
		Select a term, then choose its matching definition.
	</p>
</div>
