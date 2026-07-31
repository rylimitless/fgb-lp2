/**
 * FGB Academy — client-only user preferences store.
 *
 * Persisted to localStorage under the `fgb.prefs.v1` key. The store is intentionally
 * light-weight: reactive state via Svelte 5 runes, no backend round-trips, no
 * schema validation. Adding a new key is: extend `DEFAULTS`, add a UI binding
 * in `settings/+page.svelte`, and read `prefs.<key>` from anywhere.
 *
 * The `reducedMotion` preference also drives a `data-reduced-motion` attribute
 * on `<html>` so motion.css can respect it independently of the OS media query
 * (users on shared machines can opt in without changing system settings).
 */

const STORAGE_KEY = "fgb.prefs.v1";

export type DigestFrequency = "off" | "daily" | "weekly";
export type ThemeMode = "system" | "light" | "dark";
export type LearningPace = "gentle" | "balanced" | "aggressive";

type Prefs = {
	reducedMotion: boolean;
	highContrast: boolean;
	theme: ThemeMode;
	streakReminders: boolean;
	deadlineAlerts: boolean;
	digest: DigestFrequency;
	pace: LearningPace;
	showExplanationsFirst: boolean;
};

const DEFAULTS: Prefs = {
	reducedMotion: false,
	highContrast: false,
	theme: "system",
	streakReminders: true,
	deadlineAlerts: true,
	digest: "weekly",
	pace: "balanced",
	showExplanationsFirst: false,
};

function readStorage(): Prefs {
	if (typeof window === "undefined") return { ...DEFAULTS };
	try {
		const raw = window.localStorage.getItem(STORAGE_KEY);
		if (!raw) return { ...DEFAULTS };
		const parsed = JSON.parse(raw);
		return { ...DEFAULTS, ...parsed };
	} catch {
		return { ...DEFAULTS };
	}
}

function writeStorage(value: Prefs) {
	if (typeof window === "undefined") return;
	try {
		window.localStorage.setItem(STORAGE_KEY, JSON.stringify(value));
	} catch {
		/* localStorage disabled — non-fatal */
	}
}

function applySideEffects(p: Prefs) {
	if (typeof document === "undefined") return;
	const root = document.documentElement;
	if (p.reducedMotion) root.setAttribute("data-reduced-motion", "true");
	else root.removeAttribute("data-reduced-motion");
	if (p.highContrast) root.setAttribute("data-contrast", "high");
	else root.removeAttribute("data-contrast");
	const dark =
		p.theme === "dark" ||
		(p.theme === "system" &&
			window.matchMedia("(prefers-color-scheme: dark)").matches);
	root.classList.toggle("dark", dark);
	root.style.colorScheme = dark ? "dark" : "light";
	document
		.querySelectorAll<HTMLMetaElement>('meta[name="theme-color"]')
		.forEach((meta) => {
			meta.content = dark ? "#071625" : "#f2f5f7";
		});
	window.dispatchEvent(
		new CustomEvent("fgb:themechange", {
			detail: { theme: p.theme, resolvedTheme: dark ? "dark" : "light" },
		}),
	);
}

const state = $state<Prefs>(readStorage());
applySideEffects(state);

if (typeof window !== "undefined") {
	const media = window.matchMedia("(prefers-color-scheme: dark)");
	media.addEventListener("change", () => {
		if (state.theme === "system") applySideEffects(state);
	});
}

export const prefs = state;

export function setPref<K extends keyof Prefs>(key: K, value: Prefs[K]) {
	state[key] = value;
	writeStorage(state);
	applySideEffects(state);
}

export function resetPrefs() {
	Object.assign(state, DEFAULTS);
	writeStorage(state);
	applySideEffects(state);
}
