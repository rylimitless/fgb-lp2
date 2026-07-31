<script lang="ts">
	import { Moon, Sun } from "@lucide/svelte";
	import { prefs, setPref } from "$lib/preferences.svelte";

	let resolvedDark = $state(false);

	function syncResolvedTheme() {
		resolvedDark = document.documentElement.classList.contains("dark");
	}

	$effect(() => {
		if (typeof window === "undefined") return;
		syncResolvedTheme();
		const handler = () => syncResolvedTheme();
		window.addEventListener("fgb:themechange", handler);
		return () => window.removeEventListener("fgb:themechange", handler);
	});

	function toggleTheme() {
		setPref("theme", resolvedDark ? "light" : "dark");
	}
</script>

<button
	type="button"
	class="focus-premium inline-flex size-9 items-center justify-center rounded-lg border border-border bg-card text-muted-foreground shadow-sm transition-colors hover:border-border-strong hover:text-foreground"
	onclick={toggleTheme}
	aria-label={`Switch to ${resolvedDark ? "light" : "dark"} theme`}
	title={`Theme: ${prefs.theme}`}
>
	{#if resolvedDark}
		<Sun class="size-4" aria-hidden="true" />
	{:else}
		<Moon class="size-4" aria-hidden="true" />
	{/if}
</button>
