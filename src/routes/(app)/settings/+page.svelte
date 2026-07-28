<script lang="ts">
	import { page } from "$app/stores";
	import {
		Bell,
		Palette,
		ShieldCheck,
		UserRound,
		GraduationCap,
		LogOut,
		Sparkles,
		Moon,
	} from "@lucide/svelte";
	import * as Button from "$lib/components/ui/button";
	import { GiaAvatar, PageHeader, showToast } from "$lib/components/brand";
	import {
		prefs,
		setPref,
		type DigestFrequency,
		type ThemeMode,
		type LearningPace,
	} from "$lib/preferences.svelte";
	import { goto } from "$app/navigation";

	let user = $derived(($page.data as any)?.user);
	let signingOutAll = $state(false);

	async function signOutAllSessions() {
		signingOutAll = true;
		try {
			const res = await fetch("/api/logout", {
				method: "POST",
				credentials: "include",
			});
			if (res.ok || res.status === 401) {
				showToast("Signed out of every session on this account.", {
					title: "Sessions revoked",
					variant: "success",
				});
				await goto("/login");
			} else {
				showToast("Couldn't reach the server.", {
					title: "Try again",
					variant: "warning",
				});
			}
		} catch {
			showToast("Network error while signing out.", {
				title: "Retry",
				variant: "warning",
			});
		} finally {
			signingOutAll = false;
		}
	}

	function announce(label: string) {
		showToast(`${label} saved.`, { title: "Preferences", variant: "success" });
	}
</script>

<svelte:head><title>Settings · FGB Academy</title></svelte:head>

<div class="mx-auto flex w-full max-w-5xl flex-col gap-6">
	<PageHeader
		title="Profile & settings"
		eyebrow="Account"
		description="Personalise your Academy experience. Preferences are saved to this device and take effect immediately."
	>
		{#snippet icon()}
			<UserRound class="size-6 text-primary" />
		{/snippet}
	</PageHeader>

	<section class="grid grid-cols-1 gap-6 lg:grid-cols-[320px_1fr]">
		<!-- Identity card -->
		<div
			class="rounded-3xl border border-border bg-card p-6 text-center shadow-sm motion-rise-in"
		>
			<GiaAvatar size={88} />
			<h2 class="mt-4 text-xl font-semibold text-foreground">
				{user?.name ?? "Academy user"}
			</h2>
			<p class="mt-1 text-sm text-muted-foreground">{user?.email}</p>
			<div class="mt-4 flex flex-wrap justify-center gap-2">
				{#each user?.roles ?? [user?.role ?? "learner"] as role}
					<span
						class="rounded-full border border-border-strong bg-muted px-2.5 py-1 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground"
					>
						{role}
					</span>
				{/each}
			</div>
			<p class="mt-6 text-[11px] uppercase tracking-wider text-muted-foreground">
				Preferences store
			</p>
			<p class="mt-1 text-xs text-muted-foreground">
				This device only · not synced across devices
			</p>
		</div>

		<!-- Bento grid of preference groups -->
		<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
			<!-- Motion & display -->
			<article
				class="rounded-3xl border border-border bg-card p-5 lift motion-rise-in"
			>
				<div class="flex items-center gap-3 mb-4">
					<span
						class="flex size-10 items-center justify-center rounded-xl bg-accent-soft text-accent-foreground"
					>
						<Palette class="size-5" />
					</span>
					<div>
						<h3 class="text-sm font-semibold text-foreground">Motion & display</h3>
						<p class="text-xs text-muted-foreground">
							Overrides your OS reduced-motion setting.
						</p>
					</div>
				</div>
				<div class="flex flex-col gap-3">
					<label
						class="flex cursor-pointer items-center justify-between gap-3 rounded-xl border border-border-strong bg-surface-1 px-3 py-2.5 text-sm"
					>
						<span class="flex-1">
							<span class="font-medium text-foreground">Reduced motion</span>
							<span class="mt-0.5 block text-[11px] text-muted-foreground">
								Fades and lifts freeze. Great for vestibular sensitivity.
							</span>
						</span>
						<input
							type="checkbox"
							class="sr-only peer"
							checked={prefs.reducedMotion}
							onchange={(e) => {
								setPref(
									"reducedMotion",
									(e.currentTarget as HTMLInputElement).checked,
								);
								announce("Reduced motion");
							}}
						/>
						<span
							class="relative h-5 w-9 flex-shrink-0 rounded-full bg-muted transition-colors peer-checked:bg-primary"
						>
							<span
								class="absolute top-0.5 left-0.5 size-4 rounded-full bg-card shadow-sm transition-transform"
								class:translate-x-4={prefs.reducedMotion}
							></span>
						</span>
					</label>
					<label
						class="flex cursor-pointer items-center justify-between gap-3 rounded-xl border border-border-strong bg-surface-1 px-3 py-2.5 text-sm"
					>
						<span class="flex-1">
							<span class="font-medium text-foreground">High contrast</span>
							<span class="mt-0.5 block text-[11px] text-muted-foreground">
								Boost text contrast throughout the Academy.
							</span>
						</span>
						<input
							type="checkbox"
							class="sr-only peer"
							checked={prefs.highContrast}
							onchange={(e) => {
								setPref(
									"highContrast",
									(e.currentTarget as HTMLInputElement).checked,
								);
								announce("High contrast");
							}}
						/>
						<span
							class="relative h-5 w-9 flex-shrink-0 rounded-full bg-muted transition-colors peer-checked:bg-primary"
						>
							<span
								class="absolute top-0.5 left-0.5 size-4 rounded-full bg-card shadow-sm transition-transform"
								class:translate-x-4={prefs.highContrast}
							></span>
						</span>
					</label>
					<div class="flex items-center justify-between gap-3 rounded-xl border border-border-strong bg-surface-1 px-3 py-2.5 text-sm">
						<span class="flex-1 flex items-center gap-2">
							<Moon class="size-4 text-muted-foreground" />
							<span class="font-medium text-foreground">Theme</span>
						</span>
						<select
							value={prefs.theme}
							onchange={(e) => {
								setPref(
									"theme",
									(e.currentTarget as HTMLSelectElement).value as ThemeMode,
								);
								announce("Theme");
							}}
							class="rounded-md border border-border-strong bg-card px-2 py-1 text-sm text-foreground outline-none focus-visible:ring-2 focus-visible:ring-ring/40"
						>
							<option value="system">System</option>
							<option value="light">Light</option>
							<option value="dark">Dark</option>
						</select>
					</div>
				</div>
			</article>

			<!-- Notifications -->
			<article
				class="rounded-3xl border border-border bg-card p-5 lift motion-rise-in motion-stagger-1"
			>
				<div class="flex items-center gap-3 mb-4">
					<span
						class="flex size-10 items-center justify-center rounded-xl bg-info/10 text-info"
					>
						<Bell class="size-5" />
					</span>
					<div>
						<h3 class="text-sm font-semibold text-foreground">Notifications</h3>
						<p class="text-xs text-muted-foreground">
							Streak nudges, deadlines, and weekly digest.
						</p>
					</div>
				</div>
				<div class="flex flex-col gap-3">
					<label
						class="flex cursor-pointer items-center justify-between gap-3 rounded-xl border border-border-strong bg-surface-1 px-3 py-2.5 text-sm"
					>
						<span class="flex-1">
							<span class="font-medium text-foreground">Streak reminders</span>
							<span class="mt-0.5 block text-[11px] text-muted-foreground">
								Ping when your streak is at risk.
							</span>
						</span>
						<input
							type="checkbox"
							class="sr-only peer"
							checked={prefs.streakReminders}
							onchange={(e) => {
								setPref(
									"streakReminders",
									(e.currentTarget as HTMLInputElement).checked,
								);
								announce("Streak reminders");
							}}
						/>
						<span
							class="relative h-5 w-9 flex-shrink-0 rounded-full bg-muted transition-colors peer-checked:bg-primary"
						>
							<span
								class="absolute top-0.5 left-0.5 size-4 rounded-full bg-card shadow-sm transition-transform"
								class:translate-x-4={prefs.streakReminders}
							></span>
						</span>
					</label>
					<label
						class="flex cursor-pointer items-center justify-between gap-3 rounded-xl border border-border-strong bg-surface-1 px-3 py-2.5 text-sm"
					>
						<span class="flex-1">
							<span class="font-medium text-foreground">Deadline alerts</span>
							<span class="mt-0.5 block text-[11px] text-muted-foreground">
								Warn 48 hours before a course is due.
							</span>
						</span>
						<input
							type="checkbox"
							class="sr-only peer"
							checked={prefs.deadlineAlerts}
							onchange={(e) => {
								setPref(
									"deadlineAlerts",
									(e.currentTarget as HTMLInputElement).checked,
								);
								announce("Deadline alerts");
							}}
						/>
						<span
							class="relative h-5 w-9 flex-shrink-0 rounded-full bg-muted transition-colors peer-checked:bg-primary"
						>
							<span
								class="absolute top-0.5 left-0.5 size-4 rounded-full bg-card shadow-sm transition-transform"
								class:translate-x-4={prefs.deadlineAlerts}
							></span>
						</span>
					</label>
					<div class="flex items-center justify-between gap-3 rounded-xl border border-border-strong bg-surface-1 px-3 py-2.5 text-sm">
						<span class="flex-1 font-medium text-foreground">Digest</span>
						<select
							value={prefs.digest}
							onchange={(e) => {
								setPref(
									"digest",
									(e.currentTarget as HTMLSelectElement)
										.value as DigestFrequency,
								);
								announce("Digest frequency");
							}}
							class="rounded-md border border-border-strong bg-card px-2 py-1 text-sm text-foreground outline-none focus-visible:ring-2 focus-visible:ring-ring/40"
						>
							<option value="off">Off</option>
							<option value="daily">Daily</option>
							<option value="weekly">Weekly</option>
						</select>
					</div>
				</div>
			</article>

			<!-- Learning preferences -->
			<article
				class="rounded-3xl border border-border bg-card p-5 lift motion-rise-in motion-stagger-2"
			>
				<div class="flex items-center gap-3 mb-4">
					<span
						class="flex size-10 items-center justify-center rounded-xl bg-primary-soft text-primary"
					>
						<GraduationCap class="size-5" />
					</span>
					<div>
						<h3 class="text-sm font-semibold text-foreground">Learning</h3>
						<p class="text-xs text-muted-foreground">
							How the Adaptive Room paces you.
						</p>
					</div>
				</div>
				<div class="flex flex-col gap-3">
					<div class="flex items-center justify-between gap-3 rounded-xl border border-border-strong bg-surface-1 px-3 py-2.5 text-sm">
						<span class="flex-1 font-medium text-foreground">Preferred pace</span>
						<select
							value={prefs.pace}
							onchange={(e) => {
								setPref(
									"pace",
									(e.currentTarget as HTMLSelectElement).value as LearningPace,
								);
								announce("Learning pace");
							}}
							class="rounded-md border border-border-strong bg-card px-2 py-1 text-sm text-foreground outline-none focus-visible:ring-2 focus-visible:ring-ring/40"
						>
							<option value="gentle">Gentle</option>
							<option value="balanced">Balanced</option>
							<option value="aggressive">Aggressive</option>
						</select>
					</div>
					<label
						class="flex cursor-pointer items-center justify-between gap-3 rounded-xl border border-border-strong bg-surface-1 px-3 py-2.5 text-sm"
					>
						<span class="flex-1">
							<span class="font-medium text-foreground"
								>Show explanations first</span
							>
							<span class="mt-0.5 block text-[11px] text-muted-foreground">
								Read the answer key before attempting the quiz.
							</span>
						</span>
						<input
							type="checkbox"
							class="sr-only peer"
							checked={prefs.showExplanationsFirst}
							onchange={(e) => {
								setPref(
									"showExplanationsFirst",
									(e.currentTarget as HTMLInputElement).checked,
								);
								announce("Explanations first");
							}}
						/>
						<span
							class="relative h-5 w-9 flex-shrink-0 rounded-full bg-muted transition-colors peer-checked:bg-primary"
						>
							<span
								class="absolute top-0.5 left-0.5 size-4 rounded-full bg-card shadow-sm transition-transform"
								class:translate-x-4={prefs.showExplanationsFirst}
							></span>
						</span>
					</label>
				</div>
			</article>

			<!-- Security -->
			<article
				class="rounded-3xl border border-border bg-card p-5 lift motion-rise-in motion-stagger-3"
			>
				<div class="flex items-center gap-3 mb-4">
					<span
						class="flex size-10 items-center justify-center rounded-xl bg-success/10 text-success"
					>
						<ShieldCheck class="size-5" />
					</span>
					<div>
						<h3 class="text-sm font-semibold text-foreground">Security</h3>
						<p class="text-xs text-muted-foreground">
							Manage your sessions and access.
						</p>
					</div>
				</div>
				<div class="flex flex-col gap-3">
					<div
						class="rounded-xl border border-border-strong bg-surface-1 p-3 text-sm"
					>
						<p class="font-medium text-foreground">Role & access</p>
						<p class="mt-0.5 text-xs text-muted-foreground">
							{(user?.roles ?? [user?.role ?? "learner"]).join(", ")}
						</p>
					</div>
					<div
						class="rounded-xl border border-border-strong bg-surface-1 p-3 text-sm"
					>
						<p class="font-medium text-foreground">Active session</p>
						<p class="mt-0.5 text-xs text-muted-foreground">
							Signed in as {user?.email ?? "—"}
						</p>
					</div>
					<Button.Root
						variant="outline"
						class="w-full justify-center"
						onclick={signOutAllSessions}
						disabled={signingOutAll}
					>
						<LogOut class="size-4" />
						{signingOutAll
							? "Signing out…"
							: "Sign out of all sessions"}
					</Button.Root>
				</div>
			</article>
		</div>
	</section>

	<!-- Tips -->
	<div
		class="rounded-2xl border border-border bg-surface-2 p-4 text-sm text-muted-foreground motion-rise-in flex items-center gap-3"
	>
		<Sparkles class="size-4 text-accent" />
		<span>
			Preferences take effect the instant you toggle them and persist on this
			device.
		</span>
	</div>
</div>
