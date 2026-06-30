<script lang="ts">
	import { goto } from "$app/navigation";
	import {
		BookOpen,
		Play,
		CheckCircle,
		XCircle,
		Clock,
		ChevronRight,
		Search,
		GraduationCap,
		BarChart3,
		AlertCircle,
	} from "@lucide/svelte";
	import * as Button from "$lib/components/ui/button";
	import * as Tabs from "$lib/components/ui/tabs";
	import { onMount } from "svelte";

	let { data } = $props();
	let enrollments = $derived(data.enrollments ?? []);

	let activeTab = $state("active");
	let droppingId = $state<number | null>(null);

	let activeEnrollments = $derived(
		enrollments.filter((e: any) => e.status === "active")
	);
	let completedEnrollments = $derived(
		enrollments.filter((e: any) => e.status === "completed")
	);
	let droppedEnrollments = $derived(
		enrollments.filter((e: any) => e.status === "dropped")
	);

	let totalActive = $derived(activeEnrollments.length);
	let totalCompleted = $derived(completedEnrollments.length);
	let totalDropped = $derived(droppedEnrollments.length);

	function statusIcon(status: string) {
		if (status === "active") return Play;
		if (status === "completed") return CheckCircle;
		if (status === "dropped") return XCircle;
		return Clock;
	}

	function statusClass(status: string) {
		if (status === "active") return "text-info bg-info/10 border-info/20";
		if (status === "completed")
			return "text-success bg-success/10 border-success/20";
		if (status === "dropped")
			return "text-muted-foreground bg-muted/50 border-border";
		return "text-warning bg-warning/10 border-warning/20";
	}

	function formatDate(dateStr: string | null) {
		if (!dateStr) return "—";
		const d = new Date(dateStr);
		return d.toLocaleDateString("en-US", {
			month: "short",
			day: "numeric",
			year: "numeric",
		});
	}

	async function handleDrop(enrollment: any) {
		if (!confirm("Are you sure you want to drop this course?")) return;
		droppingId = enrollment.id;
		try {
			const res = await fetch(`/api/enrollments/${enrollment.id}/drop`, {
				method: "PUT",
				credentials: "include",
			});
			if (res.ok) {
				// Refresh the page data
				window.location.reload();
			}
		} catch {
			/* ignore */
		} finally {
			droppingId = null;
		}
	}

	function continueLearning(courseId: number) {
		goto(`/lesson-player?course_id=${courseId}`);
	}

	function viewCourse(courseId: number) {
		goto(`/lesson-player?course_id=${courseId}`);
	}
</script>

<svelte:head>
	<title>My Learning — FGB Academy</title>
</svelte:head>

<div class="flex w-full max-w-5xl mx-auto flex-col gap-6">
	<!-- Header -->
	<section class="flex flex-col gap-1">
		<div class="flex items-center gap-3">
			<div
				class="flex size-10 items-center justify-center rounded-xl bg-primary/10"
			>
				<GraduationCap class="size-5 text-primary" />
			</div>
			<div>
				<h1 class="text-xl font-bold text-foreground">My Learning</h1>
				<p class="text-sm text-muted-foreground">
					Track your enrolled courses and progress
				</p>
			</div>
		</div>
	</section>

	<!-- Stats row -->
	<div class="grid grid-cols-3 gap-3">
		<button
			class="flex flex-col items-center gap-1 rounded-xl border border-border bg-surface-1 p-4 text-center transition hover:border-info/30"
			onclick={() => (activeTab = "active")}
		>
			<span class="text-2xl font-bold tabular text-info">{totalActive}</span>
			<span class="text-xs text-muted-foreground">Active</span>
		</button>
		<button
			class="flex flex-col items-center gap-1 rounded-xl border border-border bg-surface-1 p-4 text-center transition hover:border-success/30"
			onclick={() => (activeTab = "completed")}
		>
			<span class="text-2xl font-bold tabular text-success"
				>{totalCompleted}</span
			>
			<span class="text-xs text-muted-foreground">Completed</span>
		</button>
		<button
			class="flex flex-col items-center gap-1 rounded-xl border border-border bg-surface-1 p-4 text-center transition hover:border-border/50"
			onclick={() => (activeTab = "dropped")}
		>
			<span class="text-2xl font-bold tabular text-muted-foreground"
				>{totalDropped}</span
			>
			<span class="text-xs text-muted-foreground">Dropped</span>
		</button>
	</div>

	<!-- Tabs -->
	<Tabs.Root value={activeTab} onValueChange={(v) => (activeTab = v)}>
		<Tabs.List class="w-full">
			<Tabs.Trigger value="active">Active ({totalActive})</Tabs.Trigger>
			<Tabs.Trigger value="completed"
				>Completed ({totalCompleted})</Tabs.Trigger
			>
			<Tabs.Trigger value="dropped">Dropped ({totalDropped})</Tabs.Trigger>
		</Tabs.List>

		<!-- Active -->
		<Tabs.Content value="active" class="mt-4">
			{#if activeEnrollments.length === 0}
				<div
					class="flex flex-col items-center gap-3 rounded-xl border border-dashed border-border p-10 text-center"
				>
					<BookOpen class="size-10 text-muted-foreground/40" />
					<div>
						<p class="text-sm font-medium text-muted-foreground">
							No active courses
						</p>
						<p class="text-xs text-muted-foreground/60 mt-1">
							Enroll in a course from the catalog to get started.
						</p>
					</div>
					<Button.Root onclick={() => goto("/content-repository")}>
						<Search class="size-4" />
						Browse courses
					</Button.Root>
				</div>
			{:else}
				<div class="flex flex-col gap-3">
					{#each activeEnrollments as enrollment}
						{@const StatusIcon = statusIcon(enrollment.status)}
						<div
							class="group flex items-center gap-4 rounded-xl border border-border bg-surface-1 p-4 transition hover:border-primary/20 hover:shadow-sm"
						>
							<!-- Course icon -->
							<div
								class="flex size-12 shrink-0 items-center justify-center rounded-lg bg-primary/10"
							>
								<BookOpen class="size-5 text-primary" />
							</div>

							<!-- Info -->
							<div class="min-w-0 flex-1">
								<div class="flex items-center gap-2">
									<p class="truncate text-sm font-semibold text-foreground">
										{enrollment.course_title}
									</p>
									<span
										class="inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-[11px] font-medium {statusClass(enrollment.status)}"
									>
										<StatusIcon class="size-3" />
										{enrollment.status}
									</span>
								</div>
								{#if enrollment.course_description}
									<p class="mt-0.5 truncate text-xs text-muted-foreground">
										{enrollment.course_description}
									</p>
								{/if}

								<!-- Progress bar -->
								<div class="mt-2 flex items-center gap-2">
									<div
										class="h-1.5 flex-1 rounded-full bg-muted overflow-hidden"
									>
										<div
											class="h-full rounded-full bg-primary transition-all duration-500"
											style="width: {enrollment.progress_pct ?? 0}%"
										></div>
									</div>
									<span class="text-[11px] font-medium tabular text-muted-foreground"
										>{Math.round(enrollment.progress_pct ?? 0)}%</span
									>
								</div>

								<p class="mt-1 text-[11px] text-muted-foreground">
									Enrolled {formatDate(enrollment.enrolled_at)}
								</p>
							</div>

							<!-- Actions -->
							<div class="flex shrink-0 items-center gap-2">
								<Button.Root
									size="sm"
									onclick={() => continueLearning(enrollment.course_id)}
								>
									<Play class="size-3.5" />
									Continue
								</Button.Root>
								<button
									class="rounded-lg p-2 text-muted-foreground transition hover:bg-destructive/10 hover:text-destructive"
									title="Drop course"
									onclick={() => handleDrop(enrollment)}
									disabled={droppingId === enrollment.id}
								>
									<XCircle class="size-4" />
								</button>
							</div>
						</div>
					{/each}
				</div>
			{/if}
		</Tabs.Content>

		<!-- Completed -->
		<Tabs.Content value="completed" class="mt-4">
			{#if completedEnrollments.length === 0}
				<div
					class="flex flex-col items-center gap-3 rounded-xl border border-dashed border-border p-10 text-center"
				>
					<CheckCircle class="size-10 text-muted-foreground/40" />
					<p class="text-sm font-medium text-muted-foreground">
						No completed courses yet
					</p>
					<p class="text-xs text-muted-foreground/60">
						Complete an active course to see it here.
					</p>
				</div>
			{:else}
				<div class="flex flex-col gap-3">
					{#each completedEnrollments as enrollment}
						{@const StatusIcon = statusIcon(enrollment.status)}
						<div
							class="flex items-center gap-4 rounded-xl border border-border bg-surface-1 p-4 transition hover:border-success/20"
						>
							<div
								class="flex size-12 shrink-0 items-center justify-center rounded-lg bg-success/10"
							>
								<CheckCircle class="size-5 text-success" />
							</div>
							<div class="min-w-0 flex-1">
								<div class="flex items-center gap-2">
									<p class="truncate text-sm font-semibold text-foreground">
										{enrollment.course_title}
									</p>
									<span
										class="inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-[11px] font-medium {statusClass(enrollment.status)}"
									>
										<StatusIcon class="size-3" />
										Completed
									</span>
								</div>
								<p class="mt-0.5 text-[11px] text-muted-foreground">
									Completed {formatDate(enrollment.completed_at)}
								</p>
							</div>
							<Button.Root
								size="sm"
								variant="outline"
								onclick={() => viewCourse(enrollment.course_id)}
							>
								<BarChart3 class="size-3.5" />
								Review
							</Button.Root>
						</div>
					{/each}
				</div>
			{/if}
		</Tabs.Content>

		<!-- Dropped -->
		<Tabs.Content value="dropped" class="mt-4">
			{#if droppedEnrollments.length === 0}
				<div
					class="flex flex-col items-center gap-3 rounded-xl border border-dashed border-border p-10 text-center"
				>
					<XCircle class="size-10 text-muted-foreground/40" />
					<p class="text-sm font-medium text-muted-foreground">
						No dropped courses
					</p>
				</div>
			{:else}
				<div class="flex flex-col gap-3">
					{#each droppedEnrollments as enrollment}
						{@const StatusIcon = statusIcon(enrollment.status)}
						<div
							class="flex items-center gap-4 rounded-xl border border-border bg-surface-1 p-4 opacity-60"
						>
							<div
								class="flex size-12 shrink-0 items-center justify-center rounded-lg bg-muted"
							>
								<XCircle class="size-5 text-muted-foreground" />
							</div>
							<div class="min-w-0 flex-1">
								<div class="flex items-center gap-2">
									<p class="truncate text-sm font-semibold text-foreground">
										{enrollment.course_title}
									</p>
									<span
										class="inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-[11px] font-medium {statusClass(enrollment.status)}"
									>
										<StatusIcon class="size-3" />
										Dropped
									</span>
								</div>
								<p class="mt-0.5 text-[11px] text-muted-foreground">
									Dropped {formatDate(enrollment.dropped_at)}
								</p>
							</div>
							<Button.Root
								size="sm"
								variant="ghost"
								onclick={() => viewCourse(enrollment.course_id)}
							>
								View
								<ChevronRight class="size-3.5" />
							</Button.Root>
						</div>
					{/each}
				</div>
			{/if}
		</Tabs.Content>
	</Tabs.Root>
</div>
