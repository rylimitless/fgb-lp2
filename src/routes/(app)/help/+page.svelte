<script lang="ts">
	import {
		LifeBuoy,
		Bot,
		Keyboard,
		Search,
		ShieldCheck,
		BookOpen,
		MessageSquare,
		Mail,
	} from "@lucide/svelte";
	import { PageHeader, GiaTip } from "$lib/components/brand";
	import { goto } from "$app/navigation";
	import * as Button from "$lib/components/ui/button";

	// Static, versionable help content. Keep entries short and task-focused —
	// this page is a rescue surface, not a manual.
	const shortcuts: { keys: string; description: string }[] = [
		{ keys: "Ctrl / Cmd K", description: "Open the global command palette" },
		{ keys: "Esc", description: "Close any dialog or menu" },
		{ keys: "Tab / Shift+Tab", description: "Move between form fields and buttons" },
		{ keys: "Enter", description: "Submit forms and confirm dialogs" },
		{ keys: "↑ / ↓", description: "Navigate the command palette results" },
	];

	const faqs: { question: string; answer: string; href?: string }[] = [
		{
			question: "How do I resume a course?",
			answer: "Open the Dashboard — the Continue Learning card picks up your last active module.",
			href: "/",
		},
		{
			question: "Where do I ask Gia a question?",
			answer: "Gia Coach reads from your team's approved documents and cites every source.",
			href: "/gia-coach",
		},
		{
			question: "What does 'Adaptive Room' do?",
			answer: "It quizzes you across your enrolled courses and adjusts difficulty from your responses.",
			href: "/adaptive-room",
		},
		{
			question: "Can I change reduced motion or notification preferences?",
			answer: "Yes — Settings has per-device toggles for motion, contrast, reminders, and pace.",
			href: "/settings",
		},
	];
</script>

<svelte:head><title>Help · FGB Academy</title></svelte:head>

<div class="mx-auto flex w-full max-w-5xl flex-col gap-6">
	<PageHeader
		title="Help & support"
		eyebrow="Support"
		description="Rescue surface for keyboard shortcuts, common questions, and how to reach the team."
	>
		{#snippet icon()}
			<LifeBuoy class="size-6 text-primary" />
		{/snippet}
	</PageHeader>

	<GiaTip
		size="md"
		tone="thinking"
		message="Not seeing what you need? Ask me directly in Gia Coach — I'll answer from your team's approved documents."
	/>

	<section class="grid grid-cols-1 gap-6 md:grid-cols-2">
		<!-- Keyboard shortcuts -->
		<article
			class="rounded-3xl border border-border bg-card p-5 lift motion-rise-in"
		>
			<div class="flex items-center gap-3 mb-4">
				<span
					class="flex size-10 items-center justify-center rounded-xl bg-primary-soft text-primary"
				>
					<Keyboard class="size-5" />
				</span>
				<div>
					<h2 class="text-sm font-semibold text-foreground">
						Keyboard shortcuts
					</h2>
					<p class="text-xs text-muted-foreground">
						Every surface responds to these.
					</p>
				</div>
			</div>
			<dl class="flex flex-col gap-2">
				{#each shortcuts as s}
					<div class="flex items-center justify-between gap-3 text-sm">
						<dt class="text-muted-foreground">{s.description}</dt>
						<dd>
							<kbd
								class="rounded border border-border-strong bg-muted px-2 py-0.5 text-[11px] font-semibold text-foreground tabular"
								>{s.keys}</kbd
							>
						</dd>
					</div>
				{/each}
			</dl>
		</article>

		<!-- Quick actions -->
		<article
			class="rounded-3xl border border-border bg-card p-5 lift motion-rise-in motion-stagger-1"
		>
			<div class="flex items-center gap-3 mb-4">
				<span
					class="flex size-10 items-center justify-center rounded-xl bg-info/10 text-info"
				>
					<Search class="size-5" />
				</span>
				<div>
					<h2 class="text-sm font-semibold text-foreground">
						Fast rescues
					</h2>
					<p class="text-xs text-muted-foreground">
						One tap and you're where you need to be.
					</p>
				</div>
			</div>
			<div class="flex flex-col gap-2">
				<Button.Root
					variant="outline"
					class="w-full justify-start"
					onclick={() => goto("/gia-coach")}
				>
					<Bot class="size-4" /> Ask Gia
				</Button.Root>
				<Button.Root
					variant="outline"
					class="w-full justify-start"
					onclick={() => goto("/content-repository")}
				>
					<BookOpen class="size-4" /> Browse catalog
				</Button.Root>
				<Button.Root
					variant="outline"
					class="w-full justify-start"
					onclick={() => goto("/settings")}
				>
					<ShieldCheck class="size-4" /> Preferences &amp; security
				</Button.Root>
			</div>
		</article>
	</section>

	<!-- FAQ -->
	<section
		class="rounded-3xl border border-border bg-card p-5 md:p-6 lift motion-rise-in motion-stagger-2"
	>
		<div class="flex items-center gap-3 mb-5">
			<span
				class="flex size-10 items-center justify-center rounded-xl bg-accent-soft text-accent-foreground"
			>
				<MessageSquare class="size-5" />
			</span>
			<div>
				<h2 class="text-sm font-semibold text-foreground">
					Common questions
				</h2>
				<p class="text-xs text-muted-foreground">
					Pulled from the top Gia queries this month.
				</p>
			</div>
		</div>
		<ul class="flex flex-col divide-y divide-border" role="list">
			{#each faqs as faq}
				<li class="py-3">
					<button
						type="button"
						class="flex w-full items-start justify-between gap-4 text-left group"
						onclick={() => faq.href && goto(faq.href)}
					>
						<div class="min-w-0 flex-1">
							<p
								class="text-sm font-semibold text-foreground group-hover:text-primary transition-colors"
							>
								{faq.question}
							</p>
							<p class="mt-1 text-sm text-muted-foreground">
								{faq.answer}
							</p>
						</div>
						{#if faq.href}
							<span
								class="text-[11px] uppercase tracking-wider text-muted-foreground shrink-0"
								>Go →</span
							>
						{/if}
					</button>
				</li>
			{/each}
		</ul>
	</section>

	<!-- Contact -->
	<section
		class="rounded-3xl border border-border-strong bg-surface-2 p-5 md:p-6 flex items-center gap-4 motion-rise-in motion-stagger-3"
	>
		<span
			class="flex size-10 items-center justify-center rounded-xl bg-primary text-primary-foreground shrink-0"
		>
			<Mail class="size-5" />
		</span>
		<div class="min-w-0 flex-1">
			<p class="text-sm font-semibold text-foreground">Still stuck?</p>
			<p class="text-xs text-muted-foreground">
				Email the Academy team at
				<a
					href="mailto:academy@fgb.com"
					class="text-primary underline-offset-4 hover:underline"
					>academy@fgb.com</a
				>
				— we reply within one business day.
			</p>
		</div>
	</section>
</div>
