<script lang="ts">
	/**
	 * FGB Academy — mobile bottom dock.
	 *
	 * Shows on `<lg` viewports and gives learners one-tap access to the four
	 * routes they use every session (Dashboard, My Learning, Gia Coach,
	 * Settings). Admins still see the drawer for admin routes; the dock is
	 * additive, not a replacement.
	 *
	 * Motion respects `data-reduced-motion` (user pref) and the OS
	 * `prefers-reduced-motion` media query via `motion.css`.
	 */
	import { goto } from "$app/navigation";
	import { page } from "$app/stores";
	import {
		Home,
		GraduationCap,
		Bot,
		UserRound,
	} from "@lucide/svelte";
	import { cn } from "$lib/utils.js";

	type Props = {
		class?: string;
	};

	let { class: className }: Props = $props();

	const items = [
		{ label: "Home", href: "/", icon: Home },
		{ label: "Learn", href: "/lesson-player", icon: GraduationCap },
		{ label: "Coach", href: "/gia-coach", icon: Bot },
		{ label: "Profile", href: "/settings", icon: UserRound },
	];

	function isActive(href: string): boolean {
		const path = $page.url.pathname;
		return href === "/" ? path === "/" : path.startsWith(href);
	}
</script>

<nav
	class={cn(
		"lg:hidden fixed inset-x-0 bottom-0 z-30 border-t border-border-strong bg-card/95 backdrop-blur supports-[backdrop-filter]:bg-card/80 pb-[env(safe-area-inset-bottom)]",
		className,
	)}
	aria-label="Primary mobile"
>
	<ul class="flex items-stretch justify-around" role="list">
		{#each items as item (item.href)}
			{@const active = isActive(item.href)}
			<li class="flex-1">
				<button
					type="button"
					aria-current={active ? "page" : undefined}
					aria-label={item.label}
					onclick={() => goto(item.href)}
					class={cn(
						"flex w-full flex-col items-center justify-center gap-1 py-2 text-[10px] font-semibold uppercase tracking-wider transition-colors press",
						active
							? "text-primary"
							: "text-muted-foreground hover:text-foreground",
					)}
				>
					<span
						class={cn(
							"flex size-9 items-center justify-center rounded-2xl transition-all",
							active
								? "bg-primary-soft text-primary"
								: "text-muted-foreground",
						)}
					>
						<item.icon class="size-4" />
					</span>
					<span>{item.label}</span>
				</button>
			</li>
		{/each}
	</ul>
</nav>
