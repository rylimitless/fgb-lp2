<script lang="ts">
	import { page } from "$app/stores";
	import { Bell, Palette, ShieldCheck, UserRound } from "@lucide/svelte";
	import * as Button from "$lib/components/ui/button";
	import { GiaAvatar, PageHeader } from "$lib/components/brand";
	import { showToast } from "$lib/components/brand/toast.svelte";

	let user = $derived(($page.data as any)?.user);
	const preferenceCards = [
		{
			icon: Bell,
			title: "Learning reminders",
			desc: "Weekly nudges when your streak is at risk.",
			tone: "bg-info/10 text-info",
		},
		{
			icon: Palette,
			title: "Motion & visuals",
			desc: "Premium motion respects your system reduced-motion setting.",
			tone: "bg-accent-soft text-accent-foreground",
		},
		{
			icon: ShieldCheck,
			title: "Security posture",
			desc: "Your role and access are governed by FGB Academy permissions.",
			tone: "bg-success/10 text-success",
		},
	];
</script>

<svelte:head><title>Settings - FGB Academy</title></svelte:head>

<div class="mx-auto flex w-full max-w-5xl flex-col gap-6">
	<PageHeader
		title="Profile & settings"
		eyebrow="Account"
		description="Personalise your Academy experience and review your profile context."
	>
		{#snippet icon()}
			<UserRound class="size-6 text-primary" />
		{/snippet}
	</PageHeader>

	<section class="grid grid-cols-1 gap-6 lg:grid-cols-[320px_1fr]">
		<div class="rounded-3xl border border-border bg-card p-6 text-center shadow-sm">
			<GiaAvatar size={88} />
			<h2 class="mt-4 text-xl font-semibold text-foreground">{user?.name ?? "Academy user"}</h2>
			<p class="mt-1 text-sm text-muted-foreground">{user?.email}</p>
			<div class="mt-4 flex flex-wrap justify-center gap-2">
				{#each user?.roles ?? [user?.role ?? "learner"] as role}
					<span class="rounded-full border border-border bg-muted px-2.5 py-1 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
						{role}
					</span>
				{/each}
			</div>
		</div>

		<div class="grid grid-cols-1 gap-4">
			{#each preferenceCards as card}
				<div class="rounded-3xl border border-border bg-card p-5 lift">
					<div class="flex items-start gap-4">
						<span class={`flex size-11 items-center justify-center rounded-2xl ${card.tone}`}>
							<card.icon class="size-5" />
						</span>
						<div class="min-w-0 flex-1">
							<h3 class="text-base font-semibold text-foreground">{card.title}</h3>
							<p class="mt-1 text-sm text-muted-foreground">{card.desc}</p>
						</div>
						<Button.Root
							variant="outline"
							size="sm"
							onclick={() => showToast("Preference saved for this session.", { title: card.title, variant: "success" })}
						>
							Save
						</Button.Root>
					</div>
				</div>
			{/each}
		</div>
	</section>
</div>
