<script lang="ts">
	import { onMount } from "svelte";
	import { goto } from "$app/navigation";
	import {
		Bell,
		Check,
		LoaderCircle,
		BellRing,
		Inbox,
	} from "@lucide/svelte";
	import {
		PageHeader,
		EmptyState,
		LoadingDots,
	} from "$lib/components/brand";
	import * as Button from "$lib/components/ui/button";

	type Notification = {
		id: number;
		title: string;
		message: string;
		link?: string | null;
		is_read: boolean;
		created_at: string;
	};

	let notifications = $state<Notification[]>([]);
	let unread = $state(0);
	let loading = $state(true);
	let filter = $state<"all" | "unread">("all");
	let markingAll = $state(false);

	// Filtered client-side. Not enough volume today to warrant server-side
	// filtering; can be lifted later behind a query param if the list grows.
	let visible = $derived(
		filter === "unread"
			? notifications.filter((n) => !n.is_read)
			: notifications,
	);

	async function fetchAll() {
		loading = true;
		try {
			const res = await fetch("/api/notifications?limit=200", {
				credentials: "include",
			});
			if (res.ok) {
				const data = await res.json();
				notifications = data.items ?? [];
				unread = data.unread ?? 0;
			}
		} catch {
			/* keep whatever we have */
		}
		loading = false;
	}

	async function markRead(id: number) {
		try {
			await fetch(`/api/notifications/${id}/read`, {
				method: "PUT",
				credentials: "include",
			});
			notifications = notifications.map((n) =>
				n.id === id ? { ...n, is_read: true } : n,
			);
			unread = Math.max(0, unread - 1);
		} catch {
			/* ignore */
		}
	}

	async function markAllRead() {
		markingAll = true;
		try {
			const res = await fetch(`/api/notifications/read-all`, {
				method: "PUT",
				credentials: "include",
			});
			if (res.ok) {
				notifications = notifications.map((n) => ({
					...n,
					is_read: true,
				}));
				unread = 0;
			}
		} catch {
			/* ignore */
		}
		markingAll = false;
	}

	function open(n: Notification) {
		if (!n.is_read) markRead(n.id);
		if (n.link) goto(n.link);
	}

	function formatDate(d: string): string {
		if (!d) return "";
		const dt = new Date(d);
		const now = new Date();
		const mins = Math.floor((now.getTime() - dt.getTime()) / 60000);
		if (mins < 1) return "Just now";
		if (mins < 60) return `${mins}m ago`;
		const hrs = Math.floor(mins / 60);
		if (hrs < 24) return `${hrs}h ago`;
		const days = Math.floor(hrs / 24);
		if (days < 7) return `${days}d ago`;
		return dt.toLocaleDateString("en-US", {
			month: "short",
			day: "numeric",
			year:
				dt.getFullYear() === now.getFullYear() ? undefined : "numeric",
		});
	}

	onMount(fetchAll);
</script>

<svelte:head><title>Notifications · FGB Academy</title></svelte:head>

<div class="mx-auto flex w-full max-w-3xl flex-col gap-6">
	<PageHeader
		title="Notifications"
		eyebrow="Inbox"
		description="Every deadline, review request, and streak nudge across the Academy."
	>
		{#snippet icon()}
			<BellRing class="size-6 text-primary" />
		{/snippet}
		{#snippet actions()}
			<Button.Root
				variant="outline"
				size="sm"
				onclick={markAllRead}
				disabled={markingAll || unread === 0}
			>
				<Check class="size-3.5" />
				{markingAll ? "Marking…" : `Mark all read${unread ? ` (${unread})` : ""}`}
			</Button.Root>
		{/snippet}
	</PageHeader>

	<!-- Filter pills -->
	<div class="flex items-center gap-2" role="tablist" aria-label="Filter">
		<button
			type="button"
			role="tab"
			aria-selected={filter === "all"}
			onclick={() => (filter = "all")}
			class="rounded-full border border-border-strong px-3 py-1.5 text-xs font-medium transition-colors {filter ===
			'all'
				? 'bg-primary text-primary-foreground border-primary'
				: 'bg-card text-muted-foreground hover:text-foreground'}"
		>
			All · {notifications.length}
		</button>
		<button
			type="button"
			role="tab"
			aria-selected={filter === "unread"}
			onclick={() => (filter = "unread")}
			class="rounded-full border border-border-strong px-3 py-1.5 text-xs font-medium transition-colors {filter ===
			'unread'
				? 'bg-primary text-primary-foreground border-primary'
				: 'bg-card text-muted-foreground hover:text-foreground'}"
		>
			Unread · {unread}
		</button>
	</div>

	<!-- List -->
	{#if loading}
		<div class="flex items-center justify-center py-16 gap-2">
			<LoaderCircle class="size-5 animate-spin text-muted-foreground" />
			<LoadingDots label="Loading notifications" />
		</div>
	{:else if visible.length === 0}
		<EmptyState
			title={filter === "unread"
				? "You're all caught up"
				: "No notifications yet"}
			description={filter === "unread"
				? "Every notification has been read. Come back when the Academy pings you."
				: "The Academy hasn't posted anything for you. Deadlines, review requests, and streak reminders will appear here."}
		>
			{#snippet icon()}
				<Inbox class="size-10" />
			{/snippet}
		</EmptyState>
	{:else}
		<ul class="flex flex-col gap-2" role="list">
			{#each visible as n (n.id)}
				<li>
					<button
						type="button"
						onclick={() => open(n)}
						class="w-full text-left rounded-2xl border {n.is_read
							? 'border-border bg-card'
							: 'border-info/30 bg-info/5'} p-4 lift press transition-colors"
						aria-label={(n.is_read ? "" : "Unread — ") +
							n.title +
							" · " +
							formatDate(n.created_at)}
					>
						<div class="flex items-start gap-3">
							<span
								class="mt-1 size-2 rounded-full shrink-0 {n.is_read
									? 'bg-transparent'
									: 'bg-info'}"
								aria-hidden="true"
							></span>
							<div class="flex-1 min-w-0">
								<div
									class="flex items-start justify-between gap-3"
								>
									<p
										class="text-sm font-semibold text-foreground truncate"
									>
										{n.title}
									</p>
									<span
										class="text-[11px] text-muted-foreground shrink-0 tabular"
									>
										{formatDate(n.created_at)}
									</span>
								</div>
								<p
									class="text-xs text-muted-foreground mt-1 leading-relaxed"
								>
									{n.message}
								</p>
								{#if n.link}
									<p
										class="text-[11px] text-primary mt-2 font-medium"
									>
										Open →
									</p>
								{/if}
							</div>
						</div>
					</button>
				</li>
			{/each}
		</ul>
	{/if}
</div>
