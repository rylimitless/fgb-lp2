<script lang="ts">
	import { cn } from "$lib/utils.js";
	import { ArrowDown, ArrowUp, CheckCircle, GripVertical } from "@lucide/svelte";

	type Data = {
		question?: string;
		items?: string[];
		explanation?: string;
	};

	type Props = {
		data: Data;
		value?: number[];
		reveal?: boolean;
		onChange?: (value: number[]) => void;
		class?: string;
	};

	let {
		data,
		value,
		reveal = false,
		onChange,
		class: className,
	}: Props = $props();

	let dragging = $state<number | null>(null);

	let items = $derived(data.items ?? []);
	let order = $derived(value?.length === items.length ? value : defaultOrder(items.length));

	function defaultOrder(length: number): number[] {
		// Stable non-trivial order. For small lists this rotates by one; for
		// longer lists it interleaves odd/even indices.
		if (length <= 1) return Array.from({ length }, (_, i) => i);
		const oddEven = [
			...Array.from({ length }, (_, i) => i).filter((i) => i % 2 === 1),
			...Array.from({ length }, (_, i) => i).filter((i) => i % 2 === 0),
		];
		return oddEven.join(",") === Array.from({ length }, (_, i) => i).join(",")
			? [...oddEven.slice(1), oddEven[0]]
			: oddEven;
	}

	function commit(next: number[]) {
		onChange?.(next);
	}

	function move(from: number, to: number) {
		if (reveal || to < 0 || to >= order.length || from === to) return;
		const next = [...order];
		const [item] = next.splice(from, 1);
		if (item === undefined) return;
		next.splice(to, 0, item);
		commit(next);
	}

	function onDrop(to: number) {
		if (dragging === null) return;
		move(dragging, to);
		dragging = null;
	}

	function isCorrectPosition(position: number): boolean {
		return order[position] === position;
	}
</script>

<div class={cn("space-y-4", className)}>
	{#if data.question}
		<p class="text-sm font-medium text-foreground">{data.question}</p>
	{/if}

	<ol class="space-y-2" aria-label="Ordering options">
		{#each order as originalIndex, position (originalIndex)}
			<li
				class={cn(
					"group flex items-center gap-2 rounded-xl border bg-card p-2.5 transition-all",
					dragging === position && "opacity-50",
					reveal && isCorrectPosition(position) && "border-success/40 bg-success/5",
					reveal && !isCorrectPosition(position) && "border-destructive/40 bg-destructive/5",
				)}
				draggable={!reveal}
				ondragstart={(event) => {
					dragging = position;
					event.dataTransfer?.setData("text/plain", String(position));
				}}
				ondragover={(event) => event.preventDefault()}
				ondrop={(event) => {
					event.preventDefault();
					onDrop(position);
				}}
			>
				<span
					class="flex size-7 shrink-0 items-center justify-center rounded-full bg-muted text-xs font-semibold tabular text-muted-foreground"
				>
					{position + 1}
				</span>
				<span class="flex size-8 shrink-0 items-center justify-center rounded-lg text-muted-foreground">
					<GripVertical class="size-4" />
				</span>
				<span class="min-h-9 flex-1 content-center text-sm text-foreground">
					{items[originalIndex]}
				</span>
				{#if reveal && isCorrectPosition(position)}
					<CheckCircle class="size-4 shrink-0 text-success" />
				{/if}
				<div class="flex shrink-0 items-center gap-1">
					<button
						type="button"
						class="flex size-7 items-center justify-center rounded-full text-muted-foreground hover:bg-muted hover:text-foreground disabled:opacity-30"
						aria-label={`Move ${items[originalIndex]} up`}
						disabled={reveal || position === 0}
						onclick={() => move(position, position - 1)}
					>
						<ArrowUp class="size-3.5" />
					</button>
					<button
						type="button"
						class="flex size-7 items-center justify-center rounded-full text-muted-foreground hover:bg-muted hover:text-foreground disabled:opacity-30"
						aria-label={`Move ${items[originalIndex]} down`}
						disabled={reveal || position === order.length - 1}
						onclick={() => move(position, position + 1)}
					>
						<ArrowDown class="size-3.5" />
					</button>
				</div>
			</li>
		{/each}
	</ol>

	<p class="text-xs text-muted-foreground">
		Drag the rows or use the arrow buttons to put them in the correct order.
	</p>
</div>
