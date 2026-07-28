<script lang="ts">
	import { cn } from "$lib/utils.js";
	import {
		BookOpen,
		Compass,
		FileText,
		Search,
		User,
		X,
		LoaderCircle,
	} from "@lucide/svelte";
	import { goto } from "$app/navigation";

	type ItemKind = "nav" | "course" | "document" | "person";
	type Item = {
		label: string;
		href: string;
		description?: string;
		icon?: any;
		kind?: ItemKind;
	};

	type Props = {
		open?: boolean;
		items?: Item[];
		onOpenChange?: (open: boolean) => void;
		class?: string;
		/** Seed the input on open — used when the header search bar is a live
		 *  input that opens the palette pre-populated. */
		initialQuery?: string;
	};

	let {
		open = false,
		items = [],
		onOpenChange,
		class: className,
		initialQuery = "",
	}: Props = $props();

	let query = $state("");
	let selected = $state(0);
	let inputRef = $state<HTMLInputElement>();
	let publishedCourses = $state<Item[]>([]);
	let searchCourses = $state<Item[]>([]);
	let searchDocs = $state<Item[]>([]);
	let searchPeople = $state<Item[]>([]);
	let coursesLoaded = $state(false);
	let searchLoading = $state(false);
	let debounceHandle: ReturnType<typeof setTimeout> | null = null;
	let inflight: AbortController | null = null;

	$effect(() => {
		if (open) {
			query = initialQuery;
			selected = 0;
			requestAnimationFrame(() => {
				inputRef?.focus();
				// Put cursor at end so the user can keep typing seamlessly.
				const len = query.length;
				inputRef?.setSelectionRange(len, len);
			});
			loadCourses();
		} else {
			// Reset transient search state so the next open starts clean.
			searchCourses = [];
			searchDocs = [];
			searchPeople = [];
			searchLoading = false;
			if (inflight) {
				inflight.abort();
				inflight = null;
			}
			if (debounceHandle) {
				clearTimeout(debounceHandle);
				debounceHandle = null;
			}
		}
	});

	async function loadCourses() {
		if (typeof window === "undefined") return;
		if (coursesLoaded) return;
		coursesLoaded = true;
		try {
			const res = await fetch("/api/courses/published", {
				credentials: "include",
			});
			if (res.ok) {
				const courses = await res.json();
				publishedCourses = (courses ?? []).slice(0, 40).map((c: any) => ({
					label: c.title,
					href: c.id ? `/lesson-player?id=${c.id}` : "/lesson-player",
					description: c.description ?? "Published course",
					icon: BookOpen,
					kind: "course" as const,
				}));
			}
		} catch {
			/* non-blocking — command palette still works with local items */
		}
	}

	// Runs on every query change. Debounced ~180ms. Fires document + user
	// searches once the term is >=2 chars. Course results come from the
	// preloaded published list, filtered client-side (fast, no round-trip).
	$effect(() => {
		const q = query.trim();
		if (debounceHandle) {
			clearTimeout(debounceHandle);
			debounceHandle = null;
		}
		if (inflight) {
			inflight.abort();
			inflight = null;
		}
		selected = 0;
		if (q.length < 2) {
			searchDocs = [];
			searchPeople = [];
			searchCourses = publishedCourses.slice(0, 8);
			searchLoading = false;
			return;
		}
		searchLoading = true;
		debounceHandle = setTimeout(() => runRemoteSearch(q), 180);
	});

	async function runRemoteSearch(q: string) {
		const lower = q.toLowerCase();
		searchCourses = publishedCourses
			.filter(
				(c) =>
					c.label.toLowerCase().includes(lower) ||
					(c.description ?? "").toLowerCase().includes(lower),
			)
			.slice(0, 6);

		const ctrl = new AbortController();
		inflight = ctrl;
		const opts = { credentials: "include" as const, signal: ctrl.signal };
		try {
			const [docsRes, peopleRes] = await Promise.allSettled([
				fetch("/api/documents", opts),
				fetch("/api/admin/users", opts),
			]);

			if (docsRes.status === "fulfilled" && docsRes.value.ok) {
				const docs = await docsRes.value.json();
				searchDocs = (docs ?? [])
					.filter((d: any) => {
						const title = (d.title ?? "").toLowerCase();
						return title.includes(lower);
					})
					.slice(0, 5)
					.map((d: any) => ({
						label: d.title ?? "Untitled document",
						href: "/gia-coach",
						description: d.approved
							? "Approved document"
							: d.status
								? `Document · ${d.status}`
								: "Document",
						icon: FileText,
						kind: "document" as const,
					}));
			} else {
				searchDocs = [];
			}

			if (peopleRes.status === "fulfilled" && peopleRes.value.ok) {
				const people = await peopleRes.value.json();
				searchPeople = (people?.users ?? people ?? [])
					.filter((u: any) => {
						const name = (u.name ?? "").toLowerCase();
						const email = (u.email ?? "").toLowerCase();
						return name.includes(lower) || email.includes(lower);
					})
					.slice(0, 4)
					.map((u: any) => ({
						label: u.name ?? u.email ?? "User",
						href: "/user-management",
						description:
							(u.roles && u.roles.join(", ")) ??
							u.role ??
							"Academy user",
						icon: User,
						kind: "person" as const,
					}));
			} else {
				searchPeople = [];
			}
		} catch (err: any) {
			if (err?.name !== "AbortError") {
				searchDocs = [];
				searchPeople = [];
			}
		} finally {
			if (inflight === ctrl) inflight = null;
			searchLoading = false;
		}
	}

	// Nav items: local filter always (pinned to the bottom so content wins).
	let navItems = $derived(
		items.filter((item) => {
			const q = query.toLowerCase().trim();
			if (!q) return true;
			return (
				item.label.toLowerCase().includes(q) ||
				(item.description ?? "").toLowerCase().includes(q)
			);
		}),
	);

	type Group = { key: string; label: string; items: Item[] };
	let groups = $derived<Group[]>(
		[
			{ key: "course", label: "Courses", items: searchCourses },
			{ key: "document", label: "Documents", items: searchDocs },
			{ key: "person", label: "People", items: searchPeople },
			{ key: "nav", label: "Navigation", items: navItems },
		].filter((g) => g.items.length > 0),
	);

	let flatOrdered = $derived(groups.flatMap((g) => g.items));

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
			selected = Math.min(selected + 1, Math.max(flatOrdered.length - 1, 0));
		}
		if (e.key === "ArrowUp") {
			e.preventDefault();
			selected = Math.max(selected - 1, 0);
		}
		if (e.key === "Enter") {
			e.preventDefault();
			go(flatOrdered[selected]);
		}
	}

	function kindBadge(kind?: ItemKind): string {
		switch (kind) {
			case "course":
				return "bg-primary-soft text-primary";
			case "document":
				return "bg-info/10 text-info";
			case "person":
				return "bg-success/10 text-success";
			default:
				return "bg-muted text-muted-foreground";
		}
	}
</script>

{#if open}
	<div
		class={cn(
			"fixed inset-0 z-[100] flex items-start justify-center bg-black/50 px-4 pt-[12vh] backdrop-blur-sm",
			className,
		)}
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
		<div
			class="relative w-full max-w-2xl overflow-hidden rounded-3xl border border-border bg-background shadow-lg"
		>
			<div class="flex items-center gap-3 border-b border-border px-4 py-3">
				<Search class="size-4 text-muted-foreground" />
				<input
					bind:this={inputRef}
					bind:value={query}
					placeholder="Search courses, documents, people, pages…"
					class="h-10 flex-1 bg-transparent text-sm text-foreground outline-none placeholder:text-muted-foreground"
				/>
				{#if searchLoading}
					<LoaderCircle class="size-4 animate-spin text-muted-foreground" />
				{/if}
				<kbd
					class="hidden rounded border border-border-strong bg-muted px-1.5 py-0.5 text-[10px] text-muted-foreground sm:inline"
				>
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

			<div class="max-h-[60vh] overflow-y-auto p-2">
				{#if flatOrdered.length === 0}
					<div
						class="flex flex-col items-center justify-center gap-2 px-6 py-12 text-center"
					>
						<Compass class="size-8 text-muted-foreground/40" />
						{#if searchLoading}
							<p class="text-sm text-muted-foreground">Searching…</p>
						{:else if query.trim().length >= 2}
							<p class="text-sm text-muted-foreground">
								No matches for "{query.trim()}".
							</p>
						{:else}
							<p class="text-sm text-muted-foreground">
								Type at least two characters to search across content.
							</p>
						{/if}
					</div>
				{:else}
					{#each groups as group (group.key)}
						{@const startIndex = groups
							.filter((g) => groups.indexOf(g) < groups.indexOf(group))
							.reduce((n, g) => n + g.items.length, 0)}
						<div class="flex flex-col">
							<div
								class="px-3 pt-3 pb-1 text-[10px] font-semibold uppercase tracking-[0.14em] text-muted-foreground flex items-center justify-between"
							>
								<span>{group.label}</span>
								<span class="tabular">{group.items.length}</span>
							</div>
							{#each group.items as item, j}
								{@const globalIndex = startIndex + j}
								<button
									type="button"
									class={cn(
										"flex w-full items-center gap-3 rounded-2xl px-3 py-2.5 text-left transition-colors",
										selected === globalIndex
											? "bg-primary/10 text-foreground"
											: "text-muted-foreground hover:bg-muted hover:text-foreground",
									)}
									onmouseenter={() => (selected = globalIndex)}
									onclick={() => go(item)}
								>
									<span
										class={cn(
											"flex size-9 items-center justify-center rounded-xl",
											kindBadge(item.kind),
										)}
									>
										{#if item.icon}
											<item.icon class="size-4" />
										{:else}
											<Compass class="size-4" />
										{/if}
									</span>
									<span class="min-w-0 flex-1">
										<span class="block truncate text-sm font-semibold text-foreground"
											>{item.label}</span
										>
										{#if item.description}
											<span
												class="block truncate text-xs text-muted-foreground"
												>{item.description}</span
											>
										{/if}
									</span>
									<span
										class="rounded-full border border-border-strong px-2 py-0.5 text-[10px] uppercase tracking-wider text-muted-foreground"
									>
										{item.kind ?? "nav"}
									</span>
								</button>
							{/each}
						</div>
					{/each}
				{/if}
			</div>

			<div
				class="flex items-center justify-between border-t border-border px-4 py-2 text-[10px] uppercase tracking-wider text-muted-foreground"
			>
				<span
					>↑↓ navigate · <kbd
						class="rounded border border-border-strong bg-muted px-1 py-0.5 tabular"
						>↵</kbd
					> open</span
				>
				<span>Courses · Documents · People · Nav</span>
			</div>
		</div>
	</div>
{/if}
