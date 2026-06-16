<script lang="ts">
	import { cn } from "$lib/utils.js";
	import { BookOpen, Compass, Search, X } from "@lucide/svelte";
	import { goto } from "$app/navigation";

	type Item = {
		label: string;
		href: string;
		description?: string;
		icon?: any;
		kind?: "nav" | "course";
	};

	type Props = {
		open?: boolean;
		items?: Item[];
		onOpenChange?: (open: boolean) => void;
		class?: string;
	};

	let {
		open = false,
		items = [],
		onOpenChange,
		class: className,
	}: Props = $props();

	let query = $state("");
	let selected = $state(0);
	let inputRef = $state<HTMLInputElement>();
	let remoteCourses = $state<Item[]>([]);
	let loadingCourses = $state(false);

	$effect(() => {
		if (open) {
			query = "";
			selected = 0;
			requestAnimationFrame(() => inputRef?.focus());
			loadCourses();
		}
	});

	async function loadCourses() {
		if (typeof window === "undefined") return;
		if (remoteCourses.length || loadingCourses) return;
		loadingCourses = true;
		try {
			const res = await fetch("/api/courses/published", { credentials: "include" });
			if (res.ok) {
				const courses = await res.json();
				remoteCourses = (courses ?? []).slice(0, 12).map((c: any) => ({
					label: c.title,
					href: "/lesson-player",
					description: c.description ?? "Published course",
					icon: BookOpen,
					kind: "course",
				}));
			}
		} catch {
			/* non-blocking */
		}
		loadingCourses = false;
	}

	let allItems = $derived([...items, ...remoteCourses]);
	let filtered = $derived(
		allItems
			.filter((item) => {
				const q = query.toLowerCase().trim();
				if (!q) return true;
				return (
					item.label.toLowerCase().includes(q) ||
					(item.description ?? "").toLowerCase().includes(q) ||
					(item.kind ?? "").includes(q)
				);
			})
			.slice(0, 10),
	);

	$effect(() => {
		query;
		selected = 0;
	});

	function close() {
		onOpenChange?.(false);
	}

	function go(item: Item | undefined) {
		if (!item) return;
		close();
		goto(item.href);
	}

	function onKeydown(e: KeyboardEvent) {
		if (e.key === "Escape") {
			e.preventDefault();
			close();
		}
		if (e.key === "ArrowDown") {
			e.preventDefault();
			selected = Math.min(selected + 1, Math.max(filtered.length - 1, 0));
		}
		if (e.key === "ArrowUp") {
			e.preventDefault();
			selected = Math.max(selected - 1, 0);
		}
		if (e.key === "Enter") {
			e.preventDefault();
			go(filtered[selected]);
		}
	}
</script>

{#if open}
	<div
		class={cn("fixed inset-0 z-[100] flex items-start justify-center bg-black/50 px-4 pt-[12vh] backdrop-blur-sm", className)}
		role="dialog"
		aria-modal="true"
		aria-label="Command palette"
		tabindex="-1"
		onkeydown={onKeydown}
	>
		<button
			type="button"
			class="absolute inset-0 cursor-default"
			aria-label="Close command palette"
			onclick={close}
		></button>
		<div class="relative w-full max-w-2xl overflow-hidden rounded-3xl border border-border bg-background shadow-lg">
			<div class="flex items-center gap-3 border-b border-border px-4 py-3">
				<Search class="size-4 text-muted-foreground" />
				<input
					bind:this={inputRef}
					bind:value={query}
					placeholder="Search courses, skills, topics, pages…"
					class="h-10 flex-1 bg-transparent text-sm text-foreground outline-none placeholder:text-muted-foreground"
				/>
				<kbd class="hidden rounded border border-border bg-muted px-1.5 py-0.5 text-[10px] text-muted-foreground sm:inline">
					Esc
				</kbd>
				<button
					type="button"
					class="flex size-8 items-center justify-center rounded-full text-muted-foreground hover:bg-muted hover:text-foreground"
					aria-label="Close"
					onclick={close}
				>
					<X class="size-4" />
				</button>
			</div>

			<div class="max-h-[56vh] overflow-y-auto p-2">
				{#if filtered.length === 0}
					<div class="flex flex-col items-center justify-center gap-2 px-6 py-12 text-center">
						<Compass class="size-8 text-muted-foreground/40" />
						<p class="text-sm text-muted-foreground">No results. Try a shorter search.</p>
					</div>
				{:else}
					{#each filtered as item, i}
						<button
							type="button"
							class={cn(
								"flex w-full items-center gap-3 rounded-2xl px-3 py-3 text-left transition-colors",
								selected === i ? "bg-primary/10 text-foreground" : "text-muted-foreground hover:bg-muted hover:text-foreground",
							)}
							onmouseenter={() => (selected = i)}
							onclick={() => go(item)}
						>
							<span class="flex size-9 items-center justify-center rounded-xl bg-muted text-primary">
								{#if item.icon}
									<item.icon class="size-4" />
								{:else}
									<Compass class="size-4" />
								{/if}
							</span>
							<span class="min-w-0 flex-1">
								<span class="block truncate text-sm font-semibold text-foreground">{item.label}</span>
								{#if item.description}
									<span class="block truncate text-xs text-muted-foreground">{item.description}</span>
								{/if}
							</span>
							<span class="rounded-full border border-border px-2 py-0.5 text-[10px] uppercase tracking-wider text-muted-foreground">
								{item.kind ?? "nav"}
							</span>
						</button>
					{/each}
				{/if}
			</div>
		</div>
	</div>
{/if}
