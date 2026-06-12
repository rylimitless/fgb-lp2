<script lang="ts">
    import {
        LayoutDashboard,
        FileText,
        BookOpen,
        ClipboardCheck,
        Brain,
        Sparkles,
        GraduationCap,
        ArrowRight,
        Bot,
    } from "@lucide/svelte";
    import { goto } from "$app/navigation";

    let { data } = $props();
    let stats = $derived(data.stats);
    let recentDocs = $derived(data.recentDocs ?? []);
    let recentCourses = $derived(data.recentCourses ?? []);

    function thetaLabel(t: number): string {
        if (t < -1) return "Beginner";
        if (t < 0) return "Developing";
        if (t < 0.5) return "Proficient";
        if (t < 1.5) return "Advanced";
        return "Expert";
    }

    let thetaPct = $derived(((stats.theta + 3) / 6) * 100);

    const features = [
        {
            icon: Bot,
            label: "Gia Coach",
            desc: "Ask AI about your documents",
            href: "/gia-coach",
            color: "bg-emerald-500/10 text-emerald-500",
        },
        {
            icon: Brain,
            label: "Adaptive Room",
            desc: "Personalized practice sessions",
            href: "/adaptive-room",
            color: "bg-violet-500/10 text-violet-500",
        },
        {
            icon: GraduationCap,
            label: "Lesson Player",
            desc: "Take guided courses",
            href: "/lesson-player",
            color: "bg-blue-500/10 text-blue-500",
        },
        {
            icon: FileText,
            label: "Content Studio",
            desc: "Upload & manage documents",
            href: "/content-studio",
            color: "bg-amber-500/10 text-amber-500",
        },
        {
            icon: Sparkles,
            label: "AI Generator",
            desc: "Create courses with AI",
            href: "/ai-content-generator",
            color: "bg-rose-500/10 text-rose-500",
        },
        {
            icon: ClipboardCheck,
            label: "Review Queue",
            desc: "Approve documents & courses",
            href: "/review-queue",
            color: "bg-cyan-500/10 text-cyan-500",
        },
    ];
</script>

<div class="flex w-full max-w-6xl mx-auto flex-col gap-6">
    <div class="flex items-center gap-3">
        <LayoutDashboard class="size-6 text-primary" />
        <h1 class="text-2xl font-semibold text-foreground">Dashboard</h1>
    </div>

    <!-- Stats Cards -->
    <div class="grid grid-cols-2 md:grid-cols-5 gap-3">
        <div class="rounded-xl border border-border bg-card p-4">
            <p class="text-2xl font-bold text-foreground">{stats.totalDocs}</p>
            <p class="text-xs text-muted-foreground mt-0.5">Documents</p>
        </div>
        <div class="rounded-xl border border-border bg-card p-4">
            <p class="text-2xl font-bold text-emerald-500">
                {stats.approvedDocs}
            </p>
            <p class="text-xs text-muted-foreground mt-0.5">Approved</p>
        </div>
        <div class="rounded-xl border border-border bg-card p-4">
            <p class="text-2xl font-bold text-foreground">
                {stats.totalCourses}
            </p>
            <p class="text-xs text-muted-foreground mt-0.5">Courses</p>
        </div>
        <div class="rounded-xl border border-border bg-card p-4">
            <p class="text-2xl font-bold text-blue-500">
                {stats.publishedCourses}
            </p>
            <p class="text-xs text-muted-foreground mt-0.5">Published</p>
        </div>
        <div class="rounded-xl border border-border bg-card p-4">
            <p class="text-2xl font-bold text-amber-500">
                {stats.pendingReview}
            </p>
            <p class="text-xs text-muted-foreground mt-0.5">Pending Review</p>
        </div>
    </div>

    <!-- Ability + Quick Actions Row -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
        <!-- Ability Gauge -->
        <div
            class="rounded-xl border border-border bg-card p-5 flex flex-col gap-3"
        >
            <div class="flex items-center gap-2">
                <Brain class="size-4 text-primary" />
                <span class="text-sm font-semibold text-foreground"
                    >Your Ability</span
                >
            </div>
            <p class="text-3xl font-bold text-foreground">{stats.theta}</p>
            <p class="text-xs text-muted-foreground">
                Level: {thetaLabel(stats.theta)}
            </p>
            <div class="h-2 w-full rounded-full bg-muted overflow-hidden">
                <div
                    class="h-full rounded-full bg-primary transition-all"
                    style="width: {thetaPct}%"
                ></div>
            </div>
            <button
                class="text-xs text-primary hover:underline text-left mt-1"
                onclick={() => goto("/adaptive-room")}
            >
                Go to Adaptive Room →
            </button>
        </div>

        <!-- Quick Actions -->
        <div class="md:col-span-2 rounded-xl border border-border bg-card p-5">
            <h3 class="text-sm font-semibold text-foreground mb-4">
                Quick Actions
            </h3>
            <div class="grid grid-cols-2 sm:grid-cols-3 gap-3">
                {#each features as feat}
                    <button
                        class="flex flex-col items-center gap-2 rounded-xl border border-border p-4 text-center hover:border-primary/30 hover:bg-muted/20 transition-colors"
                        onclick={() => goto(feat.href)}
                    >
                        <div
                            class="size-9 rounded-full {feat.color} flex items-center justify-center"
                        >
                            <feat.icon class="size-4" />
                        </div>
                        <div>
                            <p class="text-xs font-medium text-foreground">
                                {feat.label}
                            </p>
                            <p class="text-[10px] text-muted-foreground mt-0.5">
                                {feat.desc}
                            </p>
                        </div>
                    </button>
                {/each}
            </div>
        </div>
    </div>

    <!-- Recent Activity -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <!-- Recent Documents -->
        <div class="rounded-xl border border-border bg-card p-5">
            <div class="flex items-center justify-between mb-4">
                <h3
                    class="text-sm font-semibold text-foreground flex items-center gap-2"
                >
                    <FileText class="size-4 text-muted-foreground" /> Recent Documents
                </h3>
                <button
                    class="text-xs text-primary hover:underline"
                    onclick={() => goto("/content-studio")}>View all →</button
                >
            </div>
            {#if recentDocs.length === 0}
                <p class="text-xs text-muted-foreground py-4 text-center">
                    No documents yet.
                </p>
            {:else}
                <div class="flex flex-col gap-2">
                    {#each recentDocs as doc}
                        <div class="flex items-center gap-2.5 text-sm">
                            <FileText
                                class="size-3.5 text-muted-foreground shrink-0"
                            />
                            <span class="truncate text-foreground"
                                >{doc.title}</span
                            >
                            <span
                                class="text-xs text-muted-foreground shrink-0 ml-auto"
                                >{doc.status}</span
                            >
                        </div>
                    {/each}
                </div>
            {/if}
        </div>

        <!-- Recent Courses -->
        <div class="rounded-xl border border-border bg-card p-5">
            <div class="flex items-center justify-between mb-4">
                <h3
                    class="text-sm font-semibold text-foreground flex items-center gap-2"
                >
                    <BookOpen class="size-4 text-muted-foreground" /> Recent Courses
                </h3>
                <button
                    class="text-xs text-primary hover:underline"
                    onclick={() => goto("/ai-content-generator")}
                    >View all →</button
                >
            </div>
            {#if recentCourses.length === 0}
                <p class="text-xs text-muted-foreground py-4 text-center">
                    No courses yet.
                </p>
            {:else}
                <div class="flex flex-col gap-2">
                    {#each recentCourses as course}
                        <div class="flex items-center gap-2.5 text-sm">
                            <BookOpen
                                class="size-3.5 text-muted-foreground shrink-0"
                            />
                            <span class="truncate text-foreground"
                                >{course.title}</span
                            >
                            <span
                                class="text-xs text-muted-foreground shrink-0 ml-auto"
                                >{course.status}</span
                            >
                        </div>
                    {/each}
                </div>
            {/if}
        </div>
    </div>
</div>
