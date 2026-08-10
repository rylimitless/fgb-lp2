<script lang="ts">
	import { Send } from "@lucide/svelte";
	import { goto } from "$app/navigation";
	import GiaAvatar from "$lib/components/brand/GiaAvatar.svelte";

	let message = $state("");

	function openCoach(event: SubmitEvent) {
		event.preventDefault();
		const query = message.trim();
		goto(query ? `/gia-coach?q=${encodeURIComponent(query)}` : "/gia-coach");
	}
</script>

<section class="academy-card relative flex min-h-[120px] overflow-hidden border-info/35 bg-[radial-gradient(circle_at_85%_20%,color-mix(in_oklab,var(--info)_16%,transparent),transparent_48%),var(--card)] px-4 py-3">
	<div class="flex w-full items-center gap-3">
		<div class="relative shrink-0">
			<div class="rounded-full border border-accent/55 bg-surface-2 p-1 shadow-[var(--glow-gold)]">
				<GiaAvatar size={68} halo={false} />
			</div>
			<span class="absolute bottom-1 right-1 size-2.5 rounded-full border-2 border-card bg-success" aria-label="Gia is available"></span>
		</div>
		<div class="min-w-0 flex-1">
			<h2 class="academy-heading text-base text-foreground">Hi, I'm Gia!</h2>
			<p class="text-xs uppercase tracking-[0.1em] text-muted-foreground">Your AI learning coach</p>
			<p class="mt-1 text-sm text-muted-foreground">How can I help you today?</p>
			<form class="mt-3 flex h-9 overflow-hidden rounded-md border border-accent/55 bg-background/60 shadow-[var(--glow-gold)] focus-within:ring-2 focus-within:ring-ring/40" onsubmit={openCoach}>
				<label class="sr-only" for="gia-dashboard-message">Message Gia</label>
				<input id="gia-dashboard-message" bind:value={message} class="min-w-0 flex-1 bg-transparent px-3 text-sm text-foreground outline-none placeholder:text-muted-foreground" placeholder="Message Gia…" autocomplete="off" />
				<button type="submit" class="focus-premium flex w-9 items-center justify-center text-accent hover:bg-accent/10" aria-label="Open Gia coach with this message">
					<Send class="size-4" />
				</button>
			</form>
		</div>
	</div>
</section>
