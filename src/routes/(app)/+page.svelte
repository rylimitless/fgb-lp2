<script lang="ts">
    import { onMount } from "svelte";
    import { goto } from "$app/navigation";
    import { Chart, registerables } from "chart.js";
    import {
        ArrowRight,
        Play,
        Check,
        Lock,
        Clock,
        Sparkles,
        Trophy,
        Flame,
        Target,
        TrendingUp,
        BookOpen,
        ChevronRight,
        Zap,
        Award,
        Star,
        Brain,
        Users,
        MessageSquare,
        Calendar,
        Bell,
        GraduationCap,
    } from "@lucide/svelte";
    import * as Button from "$lib/components/ui/button";
    import {
        GiaAvatar,
        Spotlight,
        Particles,
        AnimatedGrid,
        XpRing,
        NumberTicker,
        BadgeMedal,
        StreakFlame,
        BorderBeam,
        AnimatedList,
        Marquee,
    } from "$lib/components/brand";

    Chart.register(...registerables);

    let { data } = $props();
    let stats = $derived(data.stats);
    let recentCourses = $derived(data.recentCourses ?? []);
    let userName: string = $derived((data as any)?.user?.name ?? "");
    let firstName = $derived(
        userName ? (userName.split(" ")[0] ?? "") : "there",
    );
    let gamification = $derived(
        (data as any).gamification ?? { streakDays: 0, totalScore: 0 },
    );

    function timeOfDayGreeting(): string {
        const h = new Date().getHours();
        if (h < 5) return "Working late";
        if (h < 12) return "Good morning";
        if (h < 17) return "Good afternoon";
        if (h < 22) return "Good evening";
        return "Working late";
    }
    function thetaLabel(t: number): string {
        if (t < -1) return "Beginner";
        if (t < 0) return "Developing";
        if (t < 0.5) return "Proficient";
        if (t < 1.5) return "Advanced";
        return "Expert";
    }

    // ---- Real data only, no fallback ----
    let completedCourses = $derived(
        recentCourses.filter((c: any) => c.progress?.completed),
    );
    let inProgressCourses = $derived(
        recentCourses.filter(
            (c: any) =>
                c.progress &&
                !c.progress.completed &&
                c.progress.current_module !== undefined,
        ),
    );
    let progressPct = $derived(
        stats.totalCourses > 0
            ? Math.round((completedCourses.length / stats.totalCourses) * 100)
            : 0,
    );
    let completedN = $derived(completedCourses.length);
    let inProgressN = $derived(inProgressCourses.length);
    let overdueN = $derived(Math.max(0, stats.pendingReview ?? 0));

    // Level from theta
    let levelNum = $derived(
        stats.theta !== undefined
            ? Math.min(
                  10,
                  Math.max(1, Math.round(((stats.theta + 3) / 6) * 8) + 1),
              )
            : 1,
    );
    let levelTitle = $derived(
        stats.theta !== undefined
            ? thetaLabel(stats.theta) === "Expert"
                ? "Visionary"
                : thetaLabel(stats.theta) === "Advanced"
                  ? "Rising Achiever"
                  : thetaLabel(stats.theta) === "Proficient"
                    ? "Rising Achiever"
                    : "Fast Starter"
            : "",
    );
    let levelProgressPct = $derived(
        stats.theta !== undefined
            ? Math.round((((stats.theta + 3) / 6) * 100) % 100)
            : 0,
    );

    // Continue learning — live from dashboard only, no fallback
    let continueCourse = $derived(
        data.dashboard?.continue_learning
            ? ({
                  id: data.dashboard.continue_learning.id,
                  title: data.dashboard.continue_learning.title,
                  progress: {
                      current_module:
                          data.dashboard.continue_learning.current_module,
                      completed: false,
                  },
                  image: data.dashboard.continue_learning.image,
              } as any)
            : null,
    );
    let continueTitle = $derived(continueCourse?.title ?? "");
    let continuePct = $derived(
        data.dashboard?.continue_learning?.progress_pct ?? 0,
    );
    let continueImage = $derived(
        data.dashboard?.continue_learning?.image ?? "",
    );
    let continueCourseId = $derived(
        data.dashboard?.continue_learning?.id ?? null,
    );

    // Pathway image resolver (shared convention)
    const HINTS: Array<[RegExp, string]> = [
        [/cyber|security|fraud|infosec|information/i, "cybersecurity"],
        [/compliance|regulator|policy|aml|privacy/i, "compliance"],
        [/risk|credit|portfolio/i, "risk-credit"],
        [/leader|executive|management|team/i, "leadership"],
        [
            /customer|service|relationship|experience|retail/i,
            "customer-service",
        ],
        [/banking|foundation|introduction|fundamental/i, "banking-foundations"],
    ];
    function pathImg(title: string): string {
        for (const [re, slug] of HINTS)
            if (re.test(title)) return `/brand/pathways/${slug}.png`;
        return "/brand/pathways/banking-foundations.png";
    }

    // Recommended — live from dashboard only
    let recommended = $derived(data.dashboard?.recommended ?? []);

    // Learning path — live from dashboard only
    let learningPath = $derived(data.dashboard?.learning_path ?? []);

    // Achievements — live from dashboard only
    const iconMap: Record<string, typeof Zap> = {
        zap: Zap,
        flame: Flame,
        "book-open": BookOpen,
        users: Users,
    };
    let achievements = $derived(
        (data.dashboard?.achievements ?? []).map(
            (a: { label: string; tier: string; icon: string }) => ({
                label: a.label,
                tier: a.tier as "gold" | "silver" | "bronze",
                icon: iconMap[a.icon] ?? Zap,
            }),
        ),
    );

    // Leaderboard (loaded client-side for live data)
    let leaderboard = $state([
        { name: userName || "You", xp: gamification.totalScore, me: true },
    ]);
    let leaderboardLoading = $state(false);

    async function fetchLeaderboard() {
        leaderboardLoading = true;
        try {
            const res = await fetch("/api/gamification/leaderboard?limit=10", {
                credentials: "include",
            });
            if (res.ok) {
                const d = await res.json();
                leaderboard = (d.leaderboard ?? []).map((row: any) => ({
                    name: row.name,
                    xp: row.total_score,
                    me: row.is_me,
                }));
            }
        } catch {
            /* keep fallback */
        }
        leaderboardLoading = false;
    }

    // Deadlines — live from dashboard only
    const deadlineIconMap: Record<string, typeof Clock> = {
        clock: Clock,
    };
    let deadlines = $derived(
        (data.dashboard?.deadlines ?? []).map(
            (d: { title: string; due: string; icon: string }) => ({
                title: d.title,
                due: d.due,
                icon: deadlineIconMap[d.icon] ?? Clock,
            }),
        ),
    );

    const whatsNewIconMap: Record<string, typeof Sparkles> = {
        sparkles: Sparkles,
        "trending-up": TrendingUp,
    };
    let whatsNew = $derived(
        (data.dashboard?.whats_new ?? []).map(
            (w: { title: string; meta: string; icon: string }) => ({
                title: w.title,
                meta: w.meta,
                icon: whatsNewIconMap[w.icon] ?? Sparkles,
            }),
        ),
    );
    let featuredLearning = $derived(data.dashboard?.featured_learning ?? []);

    let weeklyBars = $derived(
        data.dashboard?.weekly_activity?.length === 7
            ? data.dashboard.weekly_activity
            : [0, 0, 0, 0, 0, 0, 0],
    );

    // ---- Skill radar (Chart.js) ----
    let radarCanvas = $state<HTMLCanvasElement>();
    onMount(() => {
        fetchLeaderboard();
        if (!radarCanvas) return;
        const skillData = data.dashboard?.skill_radar;
        const labels = skillData?.labels?.length ? skillData.labels : [];
        const values =
            skillData?.data?.length === labels.length ? skillData.data : [];
        const chart = new Chart(radarCanvas, {
            type: "radar",
            data: {
                labels,
                datasets: [
                    {
                        label: "Skill level",
                        data: values,
                        backgroundColor: "#00548e22",
                        borderColor: "#00548e",
                        borderWidth: 2,
                        pointBackgroundColor: "#d6c47e",
                        pointBorderColor: "#00548e",
                        pointRadius: 4,
                    },
                ],
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: { legend: { display: false } },
                scales: {
                    r: {
                        min: 0,
                        max: 100,
                        ticks: { display: false, stepSize: 20 },
                        grid: { color: "#0000000f" },
                        angleLines: { color: "#0000000f" },
                        pointLabels: { font: { size: 11 }, color: "#5d6b80" },
                    },
                },
            },
        });
        return () => chart.destroy();
    });

    function statusIcon(s: string) {
        return s === "Completed" ? Check : s === "In Progress" ? Play : Lock;
    }
</script>

<div class="flex w-full max-w-7xl mx-auto flex-col gap-5">
    <!-- ===== GIA HERO PANEL ===== -->
    <section
        class="relative overflow-hidden rounded-3xl border border-border brand-gradient text-primary-foreground motion-rise-in"
    >
        <img
            src="/brand/hero/dashboard-ambient.png"
            alt=""
            aria-hidden="true"
            class="absolute inset-0 size-full object-cover opacity-15 mix-blend-luminosity"
        />
        <AnimatedGrid size={56} />
        <Particles quantity={46} color="#d6c47e" />
        <Spotlight class="h-full w-full" opacity={0.4} />
        <div
            class="relative z-10 flex flex-col md:flex-row items-center justify-between gap-6 px-6 py-7 md:px-10 md:py-8"
        >
            <div class="flex-1 min-w-0">
                <span
                    class="text-[11px] font-semibold uppercase tracking-[0.18em] text-accent"
                    >25 Years of Excellence</span
                >
                <h1
                    class="mt-2 text-display-md font-bold tracking-tight text-primary-foreground"
                >
                    {timeOfDayGreeting()}, {firstName}!
                    <span class="inline-block motion-float">👋</span>
                </h1>
                <p class="mt-2 text-sm text-primary-foreground/80">
                    Let's continue your learning journey.
                </p>
                <div class="mt-5 flex flex-wrap gap-3">
                    <Button.Root
                        size="lg"
                        class="bg-accent text-accent-foreground hover:bg-accent/85 shadow-glow h-10 px-5 text-sm"
                        onclick={() => goto("/gia-coach")}
                    >
                        <MessageSquare class="size-4" /> Chat with Gia
                    </Button.Root>
                    <Button.Root
                        variant="outline"
                        size="lg"
                        class="h-10 px-5 text-sm border-primary-foreground/30 bg-background/10 text-primary-foreground hover:bg-background/20"
                        onclick={() => goto("/lesson-player")}
                    >
                        Resume learning <ArrowRight class="size-4" />
                    </Button.Root>
                </div>
            </div>
            <!-- Gia character + speech bubble -->
            <div class="relative shrink-0">
                <div
                    class="absolute -left-44 top-2 hidden lg:block w-40 rounded-2xl rounded-br-sm bg-card text-card-foreground p-3 shadow-lg"
                >
                    <p class="text-[11px] leading-snug text-foreground">
                        Hi! I'm <span class="font-semibold text-primary"
                            >Gia</span
                        >, your AI learning coach. How can I help today?
                    </p>
                    <span
                        class="absolute -right-1.5 bottom-3 size-3 rotate-45 bg-card"
                    ></span>
                </div>
                <div
                    class="relative size-32 md:size-40 rounded-full overflow-hidden ring-4 ring-accent/30 bg-surface-1 motion-float"
                >
                    <img
                        src="/brand/gia/gia-3d.png"
                        alt="Gia, your AI learning coach"
                        class="size-full object-cover"
                    />
                </div>
                <span
                    class="absolute -top-1 right-2 flex size-6 items-center justify-center rounded-full bg-accent text-accent-foreground shadow-glow"
                    ><Sparkles class="size-3.5" /></span
                >
            </div>
        </div>
    </section>

    <!-- ===== FEATURED LEARNING MARQUEE ===== -->
    <section
        class="rounded-3xl border border-border bg-card p-4 lift motion-rise-in"
    >
        <div class="mb-3 flex items-center justify-between">
            <h3
                class="text-sm font-semibold text-foreground inline-flex items-center gap-2"
            >
                <Sparkles class="size-4 text-accent" /> Featured learning
            </h3>
            <span
                class="text-[10px] uppercase tracking-wider text-muted-foreground"
            >
                Live in the Academy
            </span>
        </div>
        <Marquee speed={34} class="py-1">
            {#snippet children()}
                {#each featuredLearning as item}
                    <span
                        class="inline-flex min-w-max items-center gap-2 rounded-full border border-border bg-surface-1 px-4 py-2 text-xs font-medium text-foreground"
                    >
                        <BookOpen class="size-3.5 text-primary" />
                        {item}
                    </span>
                {/each}
            {/snippet}
        </Marquee>
    </section>

    <!-- ===== PROGRESS TRIO ===== -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-5">
        <!-- Your Progress -->
        <div
            class="rounded-3xl border border-border bg-card p-5 lift motion-rise-in motion-stagger-1"
        >
            <h3 class="text-sm font-semibold text-foreground mb-4">
                Your Progress
            </h3>
            <div class="flex items-center gap-5">
                <XpRing
                    value={progressPct}
                    size={104}
                    stroke={10}
                    ring="primary"
                >
                    {#snippet center()}
                        <div class="flex flex-col items-center leading-none">
                            <span
                                class="text-xl font-bold tabular text-foreground"
                                ><NumberTicker
                                    value={progressPct}
                                    suffix="%"
                                /></span
                            >
                            <span
                                class="text-[9px] uppercase tracking-wider text-muted-foreground"
                                >Progress</span
                            >
                        </div>
                    {/snippet}
                </XpRing>
                <div class="flex flex-col gap-2 text-sm">
                    <span class="inline-flex items-center gap-2"
                        ><span class="size-2 rounded-full bg-success"></span>
                        <span class="font-semibold tabular text-foreground"
                            >{completedN}</span
                        >
                        <span class="text-muted-foreground">Completed</span
                        ></span
                    >
                    <span class="inline-flex items-center gap-2"
                        ><span class="size-2 rounded-full bg-info"></span>
                        <span class="font-semibold tabular text-foreground"
                            >{inProgressN}</span
                        >
                        <span class="text-muted-foreground">In Progress</span
                        ></span
                    >
                    <span class="inline-flex items-center gap-2"
                        ><span class="size-2 rounded-full bg-warning"></span>
                        <span class="font-semibold tabular text-foreground"
                            >{overdueN}</span
                        >
                        <span class="text-muted-foreground">To review</span
                        ></span
                    >
                </div>
            </div>
        </div>

        <!-- Current Streak -->
        <div
            class="rounded-3xl border border-border bg-card p-5 lift motion-rise-in motion-stagger-2"
        >
            <div class="flex items-center justify-between mb-4">
                <h3 class="text-sm font-semibold text-foreground">
                    Current Streak
                </h3>
                <StreakFlame count={gamification.streakDays} size="sm" />
            </div>
            <div class="flex items-end justify-between gap-2 h-20">
                {#each weeklyBars as b}
                    <div
                        class="flex-1 rounded-t-md bg-gradient-to-t from-primary to-accent min-h-1"
                        style:height="{b}%"
                    ></div>
                {/each}
            </div>
            <div class="flex justify-between gap-2 mt-1.5">
                {#each ["M", "T", "W", "T", "F", "S", "S"] as d}
                    <span
                        class="flex-1 text-center text-[9px] text-muted-foreground"
                        >{d}</span
                    >
                {/each}
            </div>
            <p class="mt-3 text-xs text-muted-foreground">
                {gamification.streakDays > 0
                    ? `Keep it up! Show up tomorrow to reach ${gamification.streakDays + 1} days.`
                    : "Start a course to begin your streak!"}
            </p>
        </div>

        <!-- Your Level -->
        <div
            class="relative overflow-hidden rounded-3xl border border-border bg-card p-5 lift motion-rise-in motion-stagger-3"
        >
            <div class="flex items-center justify-between mb-4">
                <h3 class="text-sm font-semibold text-foreground">
                    Your Level
                </h3>
                <span
                    class="flex size-9 items-center justify-center rounded-xl bg-accent-soft text-accent-foreground"
                    ><Award class="size-4" /></span
                >
            </div>
            <p
                class="text-display-sm text-2xl font-bold text-foreground leading-none"
            >
                Level {levelNum}
            </p>
            <p class="text-xs text-accent-foreground/80 mt-1 font-medium">
                {levelTitle}
            </p>
            <div class="mt-4 h-2 w-full rounded-full bg-muted overflow-hidden">
                <div
                    class="h-full rounded-full bg-gradient-to-r from-primary to-accent transition-all duration-700"
                    style:width="{levelProgressPct}%"
                ></div>
            </div>
            <p class="mt-2 text-[11px] text-muted-foreground tabular">
                1,250 XP to Level {levelNum + 1}
            </p>
        </div>
    </div>

    <!-- ===== CONTINUE LEARNING + RECOMMENDED ===== -->
    <div class="grid grid-cols-1 lg:grid-cols-5 gap-5">
        <!-- Continue Learning (dominant) -->
        <div class="lg:col-span-2 motion-rise-in motion-stagger-1">
            <button
                class="group relative h-full w-full overflow-hidden rounded-3xl border border-border bg-card text-left lift press"
                onclick={() =>
                    goto(
                        continueCourseId
                            ? `/lesson-player?id=${continueCourseId}`
                            : "/lesson-player",
                    )}
            >
                <BorderBeam size={140} duration={9} color="var(--accent)" />
                <div class="relative aspect-[16/10] w-full overflow-hidden">
                    <img
                        src={continueImage}
                        alt=""
                        class="size-full object-cover transition-transform duration-500 group-hover:scale-105"
                        onerror={(e) =>
                            ((
                                e.currentTarget as HTMLImageElement
                            ).style.display = "none")}
                    />
                    <div
                        class="absolute inset-0 bg-gradient-to-t from-black/80 via-black/20 to-transparent"
                    ></div>
                    <span
                        class="absolute top-3 left-3 inline-flex items-center gap-1 rounded-full bg-accent/90 px-2.5 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-accent-foreground"
                        >In Progress</span
                    >
                    <div class="absolute bottom-0 inset-x-0 p-5">
                        <p
                            class="text-[10px] uppercase tracking-wider text-white/70"
                        >
                            Continue learning
                        </p>
                        <h3
                            class="text-lg font-bold text-white leading-tight mt-1"
                        >
                            {continueTitle}
                        </h3>
                        <div class="mt-3 flex items-center gap-3">
                            <div
                                class="h-1.5 flex-1 rounded-full bg-white/25 overflow-hidden"
                            >
                                <div
                                    class="h-full rounded-full bg-accent"
                                    style:width="{continuePct}%"
                                ></div>
                            </div>
                            <span class="text-xs font-bold text-white tabular"
                                >{continuePct}%</span
                            >
                        </div>
                    </div>
                </div>
                <div class="flex items-center justify-between p-4">
                    <span class="text-xs text-muted-foreground"
                        >Pick up where you left off</span
                    >
                    <span
                        class="inline-flex items-center gap-1.5 rounded-full bg-primary px-4 py-1.5 text-xs font-semibold text-primary-foreground group-hover:bg-primary/85 transition-colors"
                    >
                        <Play class="size-3 fill-current" /> Continue
                    </span>
                </div>
            </button>
        </div>

        <!-- Recommended -->
        <div
            class="lg:col-span-3 rounded-3xl border border-border bg-card p-5 lift motion-rise-in motion-stagger-2"
        >
            <div class="flex items-center justify-between mb-4">
                <div>
                    <h3 class="text-sm font-semibold text-foreground">
                        Recommended for You
                    </h3>
                    <p class="text-[11px] text-muted-foreground">
                        Based on your role and interests
                    </p>
                </div>
                <button
                    class="text-xs font-medium text-primary hover:underline inline-flex items-center gap-1"
                    onclick={() => goto("/lesson-player")}
                    >View all <ChevronRight class="size-3" /></button
                >
            </div>
            <div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
                {#each recommended as rec, i}
                    <button
                        class="group flex flex-col overflow-hidden rounded-2xl border border-border bg-surface-1 text-left lift press motion-rise-in"
                        style="animation-delay: {i * 60}ms"
                        onclick={() =>
                            goto(
                                rec.id
                                    ? `/lesson-player?id=${rec.id}`
                                    : "/lesson-player",
                            )}
                    >
                        <div
                            class="relative aspect-video w-full overflow-hidden"
                        >
                            <img
                                src={rec.image}
                                alt=""
                                class="size-full object-cover transition-transform duration-500 group-hover:scale-105"
                                onerror={(e) =>
                                    ((
                                        e.currentTarget as HTMLImageElement
                                    ).style.display = "none")}
                            />
                            <div
                                class="absolute inset-0 bg-gradient-to-t from-black/50 to-transparent"
                            ></div>
                        </div>
                        <div class="p-3 flex flex-col gap-1.5">
                            <p
                                class="text-sm font-semibold text-foreground leading-snug line-clamp-2"
                            >
                                {rec.title}
                            </p>
                            <div
                                class="flex items-center gap-2 text-[11px] text-muted-foreground"
                            >
                                <span class="inline-flex items-center gap-1"
                                    ><Clock class="size-3" />
                                    {rec.duration}</span
                                >
                                <span
                                    class="size-1 rounded-full bg-muted-foreground/40"
                                ></span>
                                <span>{rec.level}</span>
                            </div>
                        </div>
                    </button>
                {/each}
            </div>
        </div>
    </div>

    <!-- ===== LEARNING PATH ===== -->
    <section
        class="rounded-3xl border border-border bg-card p-5 md:p-6 lift motion-rise-in"
    >
        <div class="flex items-center justify-between mb-6">
            <div>
                <h3
                    class="text-sm font-semibold text-foreground inline-flex items-center gap-2"
                >
                    <Target class="size-4 text-accent" /> Your Learning Path
                </h3>
                <p class="text-[11px] text-muted-foreground mt-0.5">
                    Leadership Excellence Path
                </p>
            </div>
            <button
                class="text-xs font-medium text-primary hover:underline inline-flex items-center gap-1"
                onclick={() => goto("/adaptive-room")}
                >See full path <ChevronRight class="size-3" /></button
            >
        </div>
        <ol class="flex flex-col md:flex-row md:items-start gap-4 md:gap-0">
            {#each learningPath as step, i}
                {@const Icon = statusIcon(step.status)}
                {@const done = step.status === "Completed"}
                {@const current = step.status === "In Progress"}
                <li
                    class="relative flex md:flex-col md:flex-1 items-center gap-3 md:gap-2 md:text-center"
                >
                    {#if i < learningPath.length - 1}
                        <span
                            class="hidden md:block absolute top-5 left-1/2 w-full h-0.5 {done
                                ? 'bg-accent'
                                : 'bg-border'}"
                        ></span>
                    {/if}
                    <span
                        class={`relative z-10 flex size-10 items-center justify-center rounded-full border-2 shrink-0 ${done ? "bg-accent border-accent text-accent-foreground" : current ? "bg-primary border-primary text-primary-foreground shadow-glow motion-glow" : "bg-card border-border text-muted-foreground"}`}
                    >
                        <Icon class="size-4" />
                    </span>
                    <div class="md:px-2">
                        <p
                            class={`text-xs font-medium ${current ? "text-foreground" : "text-muted-foreground"}`}
                        >
                            {step.label}
                        </p>
                        <p
                            class={`text-[10px] mt-0.5 ${done ? "text-success" : current ? "text-primary" : "text-muted-foreground/60"}`}
                        >
                            {step.status}
                        </p>
                    </div>
                </li>
            {/each}
        </ol>
    </section>

    <!-- ===== ACHIEVEMENTS + RADAR + LEADERBOARD ===== -->
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-5">
        <!-- Achievements -->
        <div
            class="rounded-3xl border border-border bg-card p-5 lift motion-rise-in motion-stagger-1"
        >
            <div class="flex items-center justify-between mb-4">
                <h3
                    class="text-sm font-semibold text-foreground inline-flex items-center gap-2"
                >
                    <Trophy class="size-4 text-accent" /> Achievements
                </h3>
                <span
                    class="text-[10px] uppercase tracking-wider text-muted-foreground"
                    >Recent</span
                >
            </div>
            <div class="grid grid-cols-2 gap-3">
                {#each achievements as a}
                    <div
                        class="flex flex-col items-center gap-2 rounded-2xl border border-border bg-surface-1 p-3 text-center"
                    >
                        <BadgeMedal tier={a.tier} size={48} />
                        <span
                            class="text-[11px] font-medium text-foreground leading-tight"
                            >{a.label}</span
                        >
                    </div>
                {/each}
            </div>
        </div>

        <!-- Skill Radar -->
        <div
            class="rounded-3xl border border-border bg-card p-5 lift motion-rise-in motion-stagger-2"
        >
            <div class="flex items-center justify-between mb-2">
                <h3
                    class="text-sm font-semibold text-foreground inline-flex items-center gap-2"
                >
                    <Brain class="size-4 text-primary" /> Skill Radar
                </h3>
                <span
                    class="text-[10px] uppercase tracking-wider text-muted-foreground"
                    >Top skills</span
                >
            </div>
            <div class="h-52"><canvas bind:this={radarCanvas}></canvas></div>
        </div>

        <!-- Leaderboard -->
        <div
            class="rounded-3xl border border-border bg-card p-5 lift motion-rise-in motion-stagger-3"
        >
            <div class="flex items-center justify-between mb-4">
                <h3
                    class="text-sm font-semibold text-foreground inline-flex items-center gap-2"
                >
                    <Star class="size-4 text-accent" /> Leaderboard
                </h3>
                <span
                    class="text-[10px] uppercase tracking-wider text-muted-foreground"
                    >This month</span
                >
            </div>
            <ol class="flex flex-col gap-1.5">
                {#each leaderboard as row, i}
                    <li
                        class={`flex items-center gap-3 rounded-xl px-3 py-2 ${row.me ? "bg-accent-soft border border-accent/30" : ""}`}
                    >
                        <span
                            class={`flex size-6 items-center justify-center rounded-full text-[11px] font-bold tabular shrink-0 ${i === 0 ? "bg-accent text-accent-foreground" : row.me ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground"}`}
                            >{i + 1}</span
                        >
                        <span
                            class={`text-sm truncate flex-1 ${row.me ? "font-semibold text-foreground" : "text-foreground"}`}
                            >{row.name}{row.me ? " (You)" : ""}</span
                        >
                        <span
                            class="text-xs font-bold tabular text-muted-foreground"
                            >{row.xp.toLocaleString()}</span
                        >
                    </li>
                {/each}
            </ol>
        </div>
    </div>

    <!-- ===== DEADLINES + WHAT'S NEW ===== -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-5">
        <div
            class="rounded-3xl border border-border bg-card p-5 lift motion-rise-in motion-stagger-1"
        >
            <h3
                class="text-sm font-semibold text-foreground inline-flex items-center gap-2 mb-4"
            >
                <Calendar class="size-4 text-warning" /> Upcoming Deadlines
            </h3>
            <ul class="flex flex-col">
                {#each deadlines as d}
                    <li
                        class="flex items-center gap-3 py-2.5 border-b border-border/40 last:border-0"
                    >
                        <span
                            class="flex size-8 items-center justify-center rounded-lg bg-warning/10 text-warning shrink-0"
                            ><d.icon class="size-4" /></span
                        >
                        <span class="text-sm text-foreground flex-1 truncate"
                            >{d.title}</span
                        >
                        <span class="text-[11px] text-muted-foreground shrink-0"
                            >{d.due}</span
                        >
                    </li>
                {/each}
            </ul>
        </div>
        <div
            class="rounded-3xl border border-border bg-card p-5 lift motion-rise-in motion-stagger-2"
        >
            <h3
                class="text-sm font-semibold text-foreground inline-flex items-center gap-2 mb-4"
            >
                <Bell class="size-4 text-info" /> What's New
            </h3>
            <AnimatedList items={whatsNew} getKey={(n, i) => `${n.title}-${i}`}>
                {#snippet children(n, _i)}
                    <div class="flex items-center gap-3 py-2.5">
                        <span
                            class="flex size-8 items-center justify-center rounded-lg bg-info/10 text-info shrink-0"
                            ><n.icon class="size-4" /></span
                        >
                        <span class="text-sm text-foreground flex-1 truncate"
                            >{n.title}</span
                        >
                        <span class="text-[11px] text-muted-foreground shrink-0"
                            >{n.meta}</span
                        >
                    </div>
                {/snippet}
            </AnimatedList>
        </div>
    </div>
</div>
