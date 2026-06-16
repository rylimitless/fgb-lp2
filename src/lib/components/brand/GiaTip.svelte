<script lang="ts">
	import { cn } from "$lib/utils.js";
	import GiaAvatar from "./GiaAvatar.svelte";
	import type { Snippet } from "svelte";

	type Tone = "default" | "celebrate" | "thinking";
	type Size = "sm" | "md";

	type Props = {
		message?: string;
		tone?: Tone;
		size?: Size;
		state?: "idle" | "thinking" | "celebrating";
		halo?: boolean;
		class?: string;
		children?: Snippet;
	};

	let {
		message,
		tone = "default",
		size = "md",
		state = "idle",
		halo = true,
		class: className,
		children,
	}: Props = $props();

	// Size + state map ---------------------------------------------------------
	const avatarSize: Record<Size, number> = { sm: 28, md: 36 };
	const padding: Record<Size, string> = {
		sm: "px-3 py-1.5 gap-2.5 text-xs",
		md: "px-4 py-2.5 gap-3 text-sm",
	};

	// Tone -> background + foreground tokens. Single-accent rule from
	// RESEARCH.md S2 — never two semantic colours on the same chip.
	const toneClass: Record<Tone, string> = {
		default: "bg-surface-2 text-foreground border-border",
		celebrate: "bg-accent-soft text-accent-foreground border-accent/40",
		thinking: "bg-info/10 text-info border-info/30",
	};

	// If caller supplied a Gia state explicitly, prefer it; otherwise derive
	// from tone so the avatar reads "thinking" inside a thinking tip.
	let resolvedState = $derived(
		state ?? (tone === "thinking" ? "thinking" : tone === "celebrate" ? "celebrating" : "idle"),
	);
</script>

<aside
	class={cn(
		"inline-flex items-center rounded-full border motion-rise-in",
		padding[size],
		toneClass[tone],
		className,
	)}
	role="note"
	aria-label="Coach tip from Gia"
>
	<GiaAvatar size={avatarSize[size]} state={resolvedState} {halo} />
	<span class="min-w-0 leading-snug">
		{#if children}
			{@render children()}
		{:else if message}
			{message}
		{/if}
	</span>
</aside>
