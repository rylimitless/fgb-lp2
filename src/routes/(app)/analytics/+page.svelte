<script lang="ts">
    import {
        BarChart3,
        Users,
        MessageSquare,
        Target,
        Zap,
        ClipboardList,
        ArrowRight,
        Building2,
        AlertTriangle,
        TrendingUp,
        GraduationCap,
        CheckCircle,
        Clock,
        Award,
        Medal,
        Trophy,
        UserPlus,
        Shield,
        Route,
        Library,
        Flame,
        FileText,
        FileWarning,
        Calendar,
        Download,
    } from "@lucide/svelte";
    import { Chart, registerables } from "chart.js";
    import { goto } from "$app/navigation";
    import {
        PageHeader,
        StatCard,
    } from "$lib/components/brand";

    Chart.register(...registerables);

    let { data } = $props();

    let overview = $derived(data.overview);
    let mostFailed = $derived(data.mostFailed ?? []);
    let courseEffectiveness = $derived(data.courseEffectiveness ?? []);
    let coachUsage = $derived(data.coachUsage ?? []);
    let enrollmentTimeline = $derived(data.enrollmentTimeline ?? []);
    let credentialTimeline = $derived(data.credentialTimeline ?? []);
    let userGrowth = $derived(data.userGrowth ?? []);
    let timeRange = $state<number>(data.days ? parseInt(data.days) : 30);

    function setTimeRange(days: number) {
        timeRange = days;
        goto(`/analytics?days=${days}`, { keepFocus: true, noScroll: true });
    }

    // --- CSV export helper (#7) ---
    function exportCSV(filename: string, rows: any[]) {
        if (!rows || rows.length === 0) return;
        const headers = Object.keys(rows[0]);
        const escape = (v: any) => {
            const s = v == null ? "" : String(v);
            return /[",\n]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s;
        };
        const csv = [
            headers.join(","),
            ...rows.map((r) => headers.map((h) => escape(r[h])).join(",")),
        ].join("\n");
        const blob = new Blob([csv], { type: "text/csv;charset=utf-8;" });
        const url = URL.createObjectURL(blob);
        const link = document.createElement("a");
        link.href = url;
        link.download = filename;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        URL.revokeObjectURL(url);
    }
    let auditLog = $derived(data.auditLog ?? []);
    let teamOverview = $derived(data.teamOverview ?? []);
    let strugglingLearners = $derived(data.strugglingLearners ?? []);
    let completionRates = $derived(data.completionRates ?? []);

    // Enrollment dashboard (#1)
    let enrollmentOverview = $derived(data.enrollmentOverview ?? null);
    let stalledEnrollments = $derived(data.stalledEnrollments ?? []);

    // Credentials dashboard (#2)
    let credentialOverview = $derived(data.credentialOverview ?? null);

    // User growth & composition (#3)
    let userOverview = $derived(data.userOverview ?? null);

    // Learning paths (#4)
    let learningPathOverview = $derived(data.learningPathOverview ?? null);

    // Content (#5)
    let contentOverview = $derived(data.contentOverview ?? null);

    // Engagement (#6)
    let engagementOverview = $derived(data.engagementOverview ?? null);

    let activeTab = $state<"platform" | "team">("platform");
    // Sub-tabs within the Platform tab — group the dashboards so the page
    // isn't one long scroll. Defaults to "overview" (the at-a-glance cards).
    let platformSubtab = $state<
        "overview" | "enrollment" | "credentials" | "users" | "content" | "paths"
    >("overview");
    let userRoles: string[] = $derived((data as any)?.user?.roles ?? []);
    let isManager = $derived(userRoles.includes("manager") || userRoles.includes("admin"));

    // Chart.js instances
    let failedCanvas = $state<HTMLCanvasElement>();
    let effectivenessCanvas = $state<HTMLCanvasElement>();
    let radarCanvas = $state<HTMLCanvasElement>();
    let usageCanvas = $state<HTMLCanvasElement>();
    let enrollmentTimelineCanvas = $state<HTMLCanvasElement>();
    let credentialTimelineCanvas = $state<HTMLCanvasElement>();
    let userGrowthCanvas = $state<HTMLCanvasElement>();
    let userRoleCanvas = $state<HTMLCanvasElement>();

    /**
     * FGB Academy chart palette — derived from the brand tokens defined in
     * layout.css (chart-1..5). These mirror the navy/info/gold/success/streak
     * palette so charts read consistently with the rest of the platform.
     * Hex strings are needed because Chart.js renders to canvas and cannot
     * resolve CSS custom properties.
     */
    const brandPalette = {
        navy:    "#00548e",
        navySoft:"#00548e33",
        info:    "#3a87cf",
        infoSoft:"#3a87cf33",
        gold:    "#d6c47e",
        goldSoft:"#d6c47e44",
        success: "#3fa46a",
        successSoft:"#3fa46a44",
        streak:  "#e07a3e",
        streakSoft:"#e07a3e44",
        danger:  "#d6464d",
        dangerSoft:"#d6464d33",
    };
    const accentColors = [
        brandPalette.navy,
        brandPalette.info,
        brandPalette.gold,
        brandPalette.success,
        brandPalette.streak,
        brandPalette.danger,
        "#8b5cf6",
        "#06b6d4",
        "#64748b",
        "#14b8a6",
    ];

    function truncate(str: string, len: number): string {
        return str.length > len ? str.slice(0, len) + "…" : str;
    }

    function actionBadge(action: string): string {
        // Mirrors /audit-log: 4-token semantic mapping. See BRAND.md §4.3.
        if (action.includes("failed") || action.includes("deleted"))
            return "bg-destructive/10 text-destructive";
        if (action.includes("created") || action.includes("uploaded") || action.includes("approved"))
            return "bg-success/10 text-success";
        if (action.includes("reviewed") || action.includes("resubmitted"))
            return "bg-warning/10 text-warning";
        if (action.startsWith("login") || action.startsWith("logout") || action.startsWith("gia") || action.includes("user_") || action.includes("admin"))
            return "bg-info/10 text-info";
        return "bg-muted text-muted-foreground";
    }

    function formatDetails(entry: any): string {
        const d = entry.details;
        if (!d || typeof d !== "object") return "—";
        // Show msg first, then other keys
        const parts: string[] = [];
        if (d.msg) parts.push(d.msg);
        for (const [k, v] of Object.entries(d)) {
            if (k === "msg" || k === "user_id" || k === "reason") continue;
            if (typeof v === "string") parts.push(v);
            else if (typeof v === "number") parts.push(`${k}: ${v}`);
        }
        if (d.reason && d.email) return `Failed login for ${d.email}`;
        return parts.join(" · ") || "—";
    }

    // Charts are built in an $effect so they rebuild when the underlying
    // data changes (e.g. when the date-range filter triggers a navigation).
    // Previous Chart instances are destroyed on each re-run to avoid leaks.
    let activeCharts: Chart[] = [];
    function mountChart(canvas: HTMLCanvasElement | undefined, config: any) {
        if (!canvas) return;
        const chart = new Chart(canvas, config);
        activeCharts.push(chart);
        return chart;
    }

    $effect(() => {
        // Tear down charts from the previous run before rebuilding.
        activeCharts.forEach((c) => c.destroy());
        activeCharts = [];

        // 1. Most Failed Questions - horizontal bar
        if (failedCanvas && mostFailed.length > 0) {
            const labels = mostFailed.map((f: any) =>
                truncate(f.topic || "Untitled", 30),
            );
            const wrongData = mostFailed.map((f: any) => f.total_wrong ?? 0);
            const correctData = mostFailed.map(
                (f: any) => f.total_correct ?? 0,
            );

            mountChart(failedCanvas, {
                type: "bar",
                data: {
                    labels,
                    datasets: [
                        {
                            label: "Wrong",
                            data: wrongData,
                            backgroundColor: brandPalette.dangerSoft,
                            borderColor: brandPalette.danger,
                            borderWidth: 1,
                            borderRadius: 4,
                        },
                        {
                            label: "Correct",
                            data: correctData,
                            backgroundColor: brandPalette.successSoft,
                            borderColor: brandPalette.success,
                            borderWidth: 1,
                            borderRadius: 4,
                        },
                    ],
                },
                options: {
                    indexAxis: "y",
                    responsive: true,
                    maintainAspectRatio: false,
                    plugins: {
                        legend: { display: true, position: "top" },
                        title: {
                            display: true,
                            text: "Most Challenging Topics",
                            font: { size: 14 },
                        },
                    },
                    scales: {
                        x: {
                            stacked: true,
                            title: { display: true, text: "Questions" },
                        },
                        y: { stacked: true },
                    },
                },
            });
        }

        // 2. Content Effectiveness - vertical bar
        if (effectivenessCanvas && courseEffectiveness.length > 0) {
            const labels = courseEffectiveness.map((c: any) =>
                truncate(c.course_title || "Untitled", 25),
            );
            const scores = courseEffectiveness.map((c: any) => {
                const s =
                    typeof c.avg_score === "string"
                        ? parseFloat(c.avg_score)
                        : c.avg_score;
                return s ?? 0;
            });
            const learners = courseEffectiveness.map(
                (c: any) => c.learner_count ?? 0,
            );

            mountChart(effectivenessCanvas, {
                type: "bar",
                data: {
                    labels,
                    datasets: [
                        {
                            label: "Avg Score %",
                            data: scores,
                            backgroundColor: scores.map(
                                (_: number, i: number) =>
                                    accentColors[i % accentColors.length] +
                                    "99",
                            ),
                            borderColor: scores.map(
                                (_: number, i: number) =>
                                    accentColors[i % accentColors.length],
                            ),
                            borderWidth: 1,
                            yAxisID: "y",
                        },
                        {
                            label: "Learners",
                            data: learners,
                            type: "line",
                            borderColor: brandPalette.gold,
                            backgroundColor: brandPalette.goldSoft,
                            borderWidth: 2,
                            pointRadius: 3,
                            pointBackgroundColor: brandPalette.gold,
                            tension: 0.3,
                            yAxisID: "y1",
                        },
                    ],
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    plugins: {
                        legend: { display: true, position: "top" },
                        title: {
                            display: true,
                            text: "Content Effectiveness by Course",
                            font: { size: 14 },
                        },
                    },
                    scales: {
                        y: {
                            type: "linear",
                            position: "left",
                            title: { display: true, text: "Avg Score %" },
                            min: 0,
                            max: 100,
                        },
                        y1: {
                            type: "linear",
                            position: "right",
                            title: { display: true, text: "Learners" },
                            grid: { drawOnChartArea: false },
                        },
                    },
                },
            });
        }

        // 3. Radar chart - Adaptive Learning Profile
        if (radarCanvas && overview?.adaptive) {
            const ad = overview.adaptive;
            // Normalize values to 0-100 scale for radar
            const thetaNorm = ((ad.avg_theta + 3) / 6) * 100;
            const spread = Math.min(((ad.stddev_theta ?? 0) / 1.5) * 100, 100);

            // Compute course effectiveness averages
            const courseCount = courseEffectiveness.length;
            const avgCompletion =
                courseCount > 0
                    ? (courseEffectiveness.reduce(
                          (sum: number, c: any) =>
                              sum + (c.completed_count ?? 0),
                          0,
                      ) /
                          Math.max(
                              1,
                              courseEffectiveness.reduce(
                                  (sum: number, c: any) =>
                                      sum + (c.learner_count ?? 0),
                                  0,
                              ),
                          )) *
                      100
                    : 0;
            const avgCourseScore =
                courseCount > 0
                    ? courseEffectiveness.reduce((sum: number, c: any) => {
                          const s =
                              typeof c.avg_score === "string"
                                  ? parseFloat(c.avg_score)
                                  : c.avg_score;
                          return sum + (s ?? 0);
                      }, 0) / courseCount
                    : 0;

            const engagement =
                overview.total_users > 0
                    ? (overview.active_learners / overview.total_users) * 100
                    : 0;

            mountChart(radarCanvas, {
                type: "radar",
                data: {
                    labels: [
                        "Avg Proficiency",
                        "Knowledge Spread",
                        "Course Score",
                        "Completion Rate",
                        "Engagement",
                    ],
                    datasets: [
                        {
                            label: "Learning Profile",
                            data: [
                                thetaNorm,
                                spread,
                                avgCourseScore,
                                avgCompletion,
                                engagement,
                            ],
                            backgroundColor: brandPalette.navySoft,
                            borderColor: brandPalette.navy,
                            borderWidth: 2,
                            pointBackgroundColor: brandPalette.gold,
                            pointBorderColor: brandPalette.navy,
                            pointRadius: 4,
                        },
                    ],
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    plugins: {
                        legend: { display: true, position: "top" },
                        title: {
                            display: true,
                            text: "Learning Effectiveness Radar",
                            font: { size: 14 },
                        },
                    },
                    scales: {
                        r: {
                            min: 0,
                            max: 100,
                            ticks: {
                                stepSize: 20,
                                backdropColor: "transparent",
                            },
                        },
                    },
                },
            });
        }

        // 4. GIA Usage Over Time - line chart
        if (usageCanvas && coachUsage.length > 0) {
            const sorted = [...coachUsage].sort(
                (a: any, b: any) =>
                    new Date(a.day).getTime() - new Date(b.day).getTime(),
            );
            const labels = sorted.map((d: any) => d.day);
            const counts = sorted.map((d: any) => d.query_count);

            mountChart(usageCanvas, {
                type: "line",
                data: {
                    labels,
                    datasets: [
                        {
                            label: "Questions",
                            data: counts,
                            borderColor: brandPalette.info,
                            backgroundColor: brandPalette.infoSoft,
                            fill: true,
                            tension: 0.3,
                            pointRadius: 2,
                            pointBackgroundColor: brandPalette.info,
                            borderWidth: 2,
                        },
                    ],
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    plugins: {
                        legend: { display: true, position: "top" },
                        title: {
                            display: true,
                            text: "GIA Usage Over Time",
                            font: { size: 14 },
                        },
                    },
                    scales: {
                        x: {
                            title: { display: true, text: "Date" },
                            ticks: { maxTicksLimit: 10 },
                        },
                        y: {
                            title: { display: true, text: "Questions" },
                            min: 0,
                        },
                    },
                },
            });
        }

        // 5. Enrollment timeline - enrollments + completions over time
        if (enrollmentTimelineCanvas && enrollmentTimeline.length > 0) {
            const labels = enrollmentTimeline.map((d: any) => d.date);
            const enrolled = enrollmentTimeline.map((d: any) => d.enrolled ?? 0);
            const completed = enrollmentTimeline.map(
                (d: any) => d.completed ?? 0,
            );

            mountChart(enrollmentTimelineCanvas, {
                type: "line",
                data: {
                    labels,
                    datasets: [
                        {
                            label: "New enrollments",
                            data: enrolled,
                            borderColor: brandPalette.navy,
                            backgroundColor: brandPalette.navySoft,
                            fill: true,
                            tension: 0.3,
                            pointRadius: 2,
                            pointBackgroundColor: brandPalette.navy,
                            borderWidth: 2,
                        },
                        {
                            label: "Completions",
                            data: completed,
                            borderColor: brandPalette.success,
                            backgroundColor: brandPalette.successSoft,
                            fill: true,
                            tension: 0.3,
                            pointRadius: 2,
                            pointBackgroundColor: brandPalette.success,
                            borderWidth: 2,
                        },
                    ],
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    plugins: {
                        legend: { display: true, position: "top" },
                        title: {
                            display: true,
                            text: "Enrollments & Completions (last 30 days)",
                            font: { size: 14 },
                        },
                    },
                    scales: {
                        x: {
                            title: { display: true, text: "Date" },
                            ticks: { maxTicksLimit: 10 },
                        },
                        y: {
                            title: { display: true, text: "Count" },
                            min: 0,
                        },
                    },
                },
            });
        }

        // 6. Credentials timeline - certs + badges issued over time
        if (credentialTimelineCanvas && credentialTimeline.length > 0) {
            const labels = credentialTimeline.map((d: any) => d.date);
            const certs = credentialTimeline.map(
                (d: any) => d.certificates ?? 0,
            );
            const badges = credentialTimeline.map(
                (d: any) => d.badges ?? 0,
            );

            mountChart(credentialTimelineCanvas, {
                type: "bar",
                data: {
                    labels,
                    datasets: [
                        {
                            label: "Certificates",
                            data: certs,
                            backgroundColor: brandPalette.goldSoft,
                            borderColor: brandPalette.gold,
                            borderWidth: 1,
                            borderRadius: 4,
                        },
                        {
                            label: "Badges",
                            data: badges,
                            backgroundColor: brandPalette.successSoft,
                            borderColor: brandPalette.success,
                            borderWidth: 1,
                            borderRadius: 4,
                        },
                    ],
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    plugins: {
                        legend: { display: true, position: "top" },
                        title: {
                            display: true,
                            text: "Credentials Issued (last 30 days)",
                            font: { size: 14 },
                        },
                    },
                    scales: {
                        x: {
                            stacked: true,
                            ticks: { maxTicksLimit: 10 },
                        },
                        y: {
                            stacked: true,
                            title: { display: true, text: "Count" },
                            min: 0,
                        },
                    },
                },
            });
        }

        // 7. User growth over time
        if (userGrowthCanvas && userGrowth.length > 0) {
            const labels = userGrowth.map((d: any) => d.date);
            const counts = userGrowth.map((d: any) => d.new_users ?? 0);

            mountChart(userGrowthCanvas, {
                type: "line",
                data: {
                    labels,
                    datasets: [
                        {
                            label: "New users",
                            data: counts,
                            borderColor: brandPalette.streak,
                            backgroundColor: brandPalette.streakSoft,
                            fill: true,
                            tension: 0.3,
                            pointRadius: 2,
                            pointBackgroundColor: brandPalette.streak,
                            borderWidth: 2,
                        },
                    ],
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    plugins: {
                        legend: { display: true, position: "top" },
                        title: {
                            display: true,
                            text: "New User Signups (last 30 days)",
                            font: { size: 14 },
                        },
                    },
                    scales: {
                        x: {
                            ticks: { maxTicksLimit: 10 },
                        },
                        y: {
                            title: { display: true, text: "New users" },
                            min: 0,
                        },
                    },
                },
            });
        }

        // 8. Users by role - doughnut
        if (userRoleCanvas && userOverview && (userOverview.by_role ?? []).length > 0) {
            const roles = userOverview.by_role;
            mountChart(userRoleCanvas, {
                type: "doughnut",
                data: {
                    labels: roles.map((r: any) => r.role),
                    datasets: [
                        {
                            data: roles.map((r: any) => r.count),
                            backgroundColor: roles.map(
                                (_: any, i: number) =>
                                    accentColors[i % accentColors.length] +
                                    "cc",
                            ),
                            borderColor: "var(--card)",
                            borderWidth: 2,
                        },
                    ],
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    plugins: {
                        legend: { display: true, position: "right" },
                        title: {
                            display: true,
                            text: "Users by Role",
                            font: { size: 14 },
                        },
                    },
                },
            });
        }
    });
</script>

<svelte:head>
    <title>Analytics - FGB</title>
</svelte:head>

<div class="flex w-full max-w-7xl mx-auto flex-col gap-6">
    <PageHeader
        title="Analytics"
        eyebrow="Academy intelligence"
        description="Operating signals across documents, courses, the Adaptive Room, and the Gia coach."
    >
        {#snippet icon()}
            <BarChart3 class="size-6 text-primary" />
        {/snippet}
    </PageHeader>

    <!-- Tabs -->
    <div class="flex gap-1 rounded-lg bg-muted p-1 w-fit">
        <button
            onclick={() => (activeTab = "platform")}
            class="px-4 py-1.5 rounded-md text-xs font-medium transition-colors {activeTab === 'platform' ? 'bg-background text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'}"
        >
            Platform
        </button>
        {#if isManager}
            <button
                onclick={() => (activeTab = "team")}
                class="px-4 py-1.5 rounded-md text-xs font-medium transition-colors {activeTab === 'team' ? 'bg-background text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'}"
            >
                Team
            </button>
        {/if}
    </div>

    {#if activeTab === "platform"}
        <!-- Date-range selector (applies to all time-series charts) -->
        <div class="flex items-center justify-between gap-3 flex-wrap">
            <div class="flex items-center gap-2 text-xs text-muted-foreground">
                <Calendar class="size-3.5" />
                <span>Time window</span>
            </div>
            <div class="flex gap-1 rounded-lg bg-muted p-1">
                {#each [[7, '7d'], [30, '30d'], [90, '90d'], [365, '1y']] as [d, label]}
                    {@const days = d as number}
                    <button
                        onclick={() => setTimeRange(days)}
                        class="px-3 py-1 rounded-md text-xs font-medium transition-colors {timeRange === days ? 'bg-background text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'}"
                    >
                        {label}
                    </button>
                {/each}
            </div>
        </div>

        <!-- Platform sub-tabs -->
        <div class="flex gap-1 flex-wrap">
            {#each [['overview','Overview'], ['enrollment','Enrollment'], ['credentials','Credentials'], ['users','Users'], ['content','Content'], ['paths','Learning Paths']] as [key, label]}
                <button
                    onclick={() => (platformSubtab = key as typeof platformSubtab)}
                    class="px-3 py-1.5 rounded-md text-xs font-medium border transition-colors {platformSubtab === key ? 'border-primary bg-primary/5 text-primary' : 'border-border text-muted-foreground hover:text-foreground hover:bg-muted/50'}"
                >
                    {label}
                </button>
            {/each}
        </div>

        <!-- ====== Enrollment & Completion Dashboard ====== -->
        {#if platformSubtab === "enrollment" && enrollmentOverview}
            {@const es = enrollmentOverview.summary}
            <section class="flex flex-col gap-4">
                <div class="flex items-center gap-2">
                    <GraduationCap class="size-4 text-primary" />
                    <h2 class="text-sm font-semibold text-foreground">
                        Enrollment & Completion
                    </h2>
                </div>

                <!-- Stat cards -->
                <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
                    <div class="motion-rise-in motion-stagger-1">
                        <StatCard
                            label="Total enrollments"
                            value={es.total_enrollments ?? 0}
                            tone="primary"
                            hint={`${es.unique_learners ?? 0} unique learners`}
                        >
                            {#snippet icon()}
                                <GraduationCap class="size-3.5" />
                            {/snippet}
                        </StatCard>
                    </div>
                    <div class="motion-rise-in motion-stagger-2">
                        <StatCard
                            label="Completed"
                            value={es.completed_count ?? 0}
                            tone="success"
                            hint={`${es.completion_rate ?? 0}% completion rate`}
                        >
                            {#snippet icon()}
                                <CheckCircle class="size-3.5" />
                            {/snippet}
                        </StatCard>
                    </div>
                    <div class="motion-rise-in motion-stagger-3">
                        <StatCard
                            label="In progress"
                            value={es.active_count ?? 0}
                            tone="info"
                            hint={`avg ${es.avg_progress ?? 0}% through`}
                        >
                            {#snippet icon()}
                                <Clock class="size-3.5" />
                            {/snippet}
                        </StatCard>
                    </div>
                    <div class="motion-rise-in motion-stagger-4">
                        <StatCard
                            label="Avg time to complete"
                            value={es.avg_days_to_complete ?? 0}
                            tone="accent"
                            suffix="d"
                            hint="across completed courses"
                        >
                            {#snippet icon()}
                                <Clock class="size-3.5" />
                            {/snippet}
                        </StatCard>
                    </div>
                </div>

                <!-- Timeline + top courses -->
                <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
                    <div
                        class="rounded-2xl border border-border bg-card p-5 lift"
                    >
                        {#if enrollmentTimeline.length > 0}
                            <div class="h-80">
                                <canvas
                                    bind:this={enrollmentTimelineCanvas}
                                ></canvas>
                            </div>
                        {:else}
                            <div
                                class="flex items-center justify-center h-80"
                            >
                                <p class="text-sm text-muted-foreground">
                                    No enrollment activity yet.
                                </p>
                            </div>
                        {/if}
                    </div>

                    <div
                        class="rounded-2xl border border-border bg-card overflow-hidden lift"
                    >
                        <div
                            class="px-5 py-4 border-b border-border"
                        >
                            <h3
                                class="text-sm font-semibold text-foreground"
                            >
                                Top courses by enrollment
                            </h3>
                            <button
                                type="button"
                                class="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
                                title="Export CSV"
                                onclick={() => exportCSV('top-courses.csv', enrollmentOverview?.top_courses ?? [])}
                            >
                                <Download class="size-3.5" />
                                CSV
                            </button>
                        </div>
                        {#if (enrollmentOverview.top_courses ?? []).length === 0}
                            <p
                                class="text-sm text-muted-foreground py-8 text-center"
                            >
                                No courses with enrollments yet.
                            </p>
                        {:else}
                            <div class="overflow-x-auto max-h-80">
                                <table class="w-full text-sm">
                                    <thead
                                        class="text-left text-xs text-muted-foreground sticky top-0 bg-card"
                                    >
                                        <tr
                                            class="border-b border-border"
                                        >
                                            <th
                                                class="px-5 py-3 font-medium"
                                            >
                                                Course
                                            </th>
                                            <th
                                                class="px-3 py-3 font-medium text-right"
                                            >
                                                Enrolled
                                            </th>
                                            <th
                                                class="px-3 py-3 font-medium text-right"
                                            >
                                                Done
                                            </th>
                                            <th
                                                class="px-5 py-3 font-medium"
                                            >
                                                Completion
                                            </th>
                                        </tr>
                                    </thead>
                                    <tbody>
                                        {#each enrollmentOverview.top_courses ?? [] as tc}
                                            <tr
                                                class="border-b border-border/50 hover:bg-muted/30"
                                            >
                                                <td
                                                    class="px-5 py-3"
                                                >
                                                    <span
                                                        class="font-medium text-foreground"
                                                    >
                                                        {truncate(tc.course_title || "Untitled", 32)}
                                                    </span>
                                                </td>
                                                <td
                                                    class="px-3 py-3 text-right text-muted-foreground tabular-nums"
                                                >
                                                    {tc.enrolled}
                                                </td>
                                                <td
                                                    class="px-3 py-3 text-right text-muted-foreground tabular-nums"
                                                >
                                                    {tc.completed}
                                                </td>
                                                <td
                                                    class="px-5 py-3"
                                                >
                                                    <div
                                                        class="flex items-center gap-2"
                                                    >
                                                        <div
                                                            class="w-16 h-2 rounded-full bg-muted overflow-hidden"
                                                        >
                                                            <div
                                                                class="h-full rounded-full {tc.completion_rate >= 70 ? 'bg-success' : tc.completion_rate >= 40 ? 'bg-warning' : 'bg-destructive'}"
                                                                style="width: {tc.completion_rate}%"
                                                            ></div>
                                                        </div>
                                                        <span
                                                            class="text-xs text-muted-foreground tabular-nums"
                                                        >
                                                            {tc.completion_rate}%
                                                        </span>
                                                    </div>
                                                </td>
                                            </tr>
                                        {/each}
                                    </tbody>
                                </table>
                            </div>
                        {/if}
                    </div>
                </div>

                <!-- Stalled enrollments -->
                <div
                    class="rounded-2xl border border-border bg-card overflow-hidden lift"
                >
                    <div class="px-5 py-4 border-b border-border">
                        <h3
                            class="text-sm font-semibold text-foreground flex items-center gap-2"
                        >
                            <AlertTriangle class="size-4 text-warning" />
                            Stalled enrollments
                            <span class="text-muted-foreground font-normal">
                                (no activity in 14+ days)
                            </span>
                        </h3>
                    </div>
                    {#if stalledEnrollments.length === 0}
                        <p
                            class="text-sm text-muted-foreground py-8 text-center"
                        >
                            No stalled enrollments. Everyone’s active.
                        </p>
                    {:else}
                        <div class="overflow-x-auto">
                            <table class="w-full text-sm">
                                <thead
                                    class="text-left text-xs text-muted-foreground"
                                >
                                    <tr class="border-b border-border">
                                        <th
                                            class="px-5 py-3 font-medium"
                                        >
                                            Learner
                                        </th>
                                        <th
                                            class="px-5 py-3 font-medium"
                                        >
                                            Course
                                        </th>
                                        <th
                                            class="px-5 py-3 font-medium text-right"
                                        >
                                            Progress
                                        </th>
                                        <th
                                            class="px-5 py-3 font-medium text-right"
                                        >
                                            Last active
                                        </th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {#each stalledEnrollments as s}
                                        <tr
                                            class="border-b border-border/50 hover:bg-muted/30"
                                        >
                                            <td class="px-5 py-3">
                                                <div class="flex flex-col">
                                                    <span
                                                        class="font-medium text-foreground"
                                                    >
                                                        {s.user_name}
                                                    </span>
                                                    <span
                                                        class="text-xs text-muted-foreground"
                                                    >
                                                        {s.user_email}
                                                    </span>
                                                </div>
                                            </td>
                                            <td
                                                class="px-5 py-3 text-muted-foreground"
                                            >
                                                {s.course_title}
                                            </td>
                                            <td
                                                class="px-5 py-3 text-right tabular-nums"
                                            >
                                                {Math.round(s.progress_pct ?? 0)}%
                                            </td>
                                            <td
                                                class="px-5 py-3 text-right text-xs text-muted-foreground tabular-nums"
                                            >
                                                {s.last_active}
                                            </td>
                                        </tr>
                                    {/each}
                                </tbody>
                            </table>
                        </div>
                    {/if}
                </div>
            </section>
        {/if}

        <!-- ====== Certificates & Badges Dashboard ====== -->
        {#if platformSubtab === "credentials" && credentialOverview}
            {@const cs = credentialOverview.summary}
            <section class="flex flex-col gap-4">
                <div class="flex items-center gap-2">
                    <Award class="size-4 text-primary" />
                    <h2 class="text-sm font-semibold text-foreground">
                        Certificates & Badges
                    </h2>
                </div>

                <!-- Stat cards -->
                <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
                    <div class="motion-rise-in motion-stagger-1">
                        <StatCard
                            label="Certificates issued"
                            value={cs.certificates_issued ?? 0}
                            tone="accent"
                            hint={`avg score ${cs.avg_cert_score ?? 0}%`}
                        >
                            {#snippet icon()}
                                <Award class="size-3.5" />
                            {/snippet}
                        </StatCard>
                    </div>
                    <div class="motion-rise-in motion-stagger-2">
                        <StatCard
                            label="Badges awarded"
                            value={cs.badges_awarded ?? 0}
                            tone="success"
                            hint={`${cs.distinct_recipients ?? 0} earners`}
                        >
                            {#snippet icon()}
                                <Medal class="size-3.5" />
                            {/snippet}
                        </StatCard>
                    </div>
                    <div class="motion-rise-in motion-stagger-3">
                        <StatCard
                            label="Gold certificates"
                            value={cs.cert_tiers.gold ?? 0}
                            tone="warning"
                            hint={`${cs.cert_tiers.silver ?? 0} silver · ${cs.cert_tiers.bronze ?? 0} bronze`}
                        >
                            {#snippet icon()}
                                <Trophy class="size-3.5" />
                            {/snippet}
                        </StatCard>
                    </div>
                    <div class="motion-rise-in motion-stagger-4">
                        <StatCard
                            label="Gold badges"
                            value={cs.badge_tiers.gold ?? 0}
                            tone="warning"
                            hint={`${cs.badge_tiers.silver ?? 0} silver · ${cs.badge_tiers.bronze ?? 0} bronze`}
                        >
                            {#snippet icon()}
                                <Trophy class="size-3.5" />
                            {/snippet}
                        </StatCard>
                    </div>
                </div>

                <!-- Timeline + top earners -->
                <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
                    <div
                        class="rounded-2xl border border-border bg-card p-5 lift"
                    >
                        {#if credentialTimeline.length > 0}
                            <div class="h-80">
                                <canvas
                                    bind:this={credentialTimelineCanvas}
                                ></canvas>
                            </div>
                        {:else}
                            <div
                                class="flex items-center justify-center h-80"
                            >
                                <p class="text-sm text-muted-foreground">
                                    No credentials issued yet.
                                </p>
                            </div>
                        {/if}
                    </div>

                    <div
                        class="rounded-2xl border border-border bg-card overflow-hidden lift"
                    >
                        <div class="px-5 py-4 border-b border-border">
                            <h3
                                class="text-sm font-semibold text-foreground flex items-center gap-2"
                            >
                                <Trophy class="size-4 text-warning" />
                                Top earners
                            </h3>
                            <button
                                type="button"
                                class="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
                                title="Export CSV"
                                onclick={() => exportCSV('top-earners.csv', credentialOverview?.top_earners ?? [])}
                            >
                                <Download class="size-3.5" />
                                CSV
                            </button>
                        </div>
                        {#if (credentialOverview.top_earners ?? []).length === 0}
                            <p
                                class="text-sm text-muted-foreground py-8 text-center"
                            >
                                No credentials earned yet.
                            </p>
                        {:else}
                            <div class="overflow-x-auto max-h-80">
                                <table class="w-full text-sm">
                                    <thead
                                        class="text-left text-xs text-muted-foreground sticky top-0 bg-card"
                                    >
                                        <tr class="border-b border-border">
                                            <th
                                                class="px-5 py-3 font-medium"
                                            >
                                                Learner
                                            </th>
                                            <th
                                                class="px-3 py-3 font-medium text-right"
                                            >
                                                Certs
                                            </th>
                                            <th
                                                class="px-3 py-3 font-medium text-right"
                                            >
                                                Badges
                                            </th>
                                            <th
                                                class="px-5 py-3 font-medium text-right"
                                            >
                                                Total
                                            </th>
                                        </tr>
                                    </thead>
                                    <tbody>
                                        {#each credentialOverview.top_earners ?? [] as earner, i}
                                            <tr
                                                class="border-b border-border/50 hover:bg-muted/30"
                                            >
                                                <td class="px-5 py-3">
                                                    <div
                                                        class="flex items-center gap-2"
                                                    >
                                                        {#if i < 3}
                                                            <span
                                                                class="inline-flex items-center justify-center size-5 rounded-full text-[10px] font-bold {i === 0 ? 'bg-warning/15 text-warning' : i === 1 ? 'bg-muted text-muted-foreground' : 'bg-accent/15 text-accent'}"
                                                            >
                                                                {i + 1}
                                                            </span>
                                                        {:else}
                                                            <span
                                                                class="inline-flex items-center justify-center size-5 rounded-full text-[10px] text-muted-foreground"
                                                            >
                                                                {i + 1}
                                                            </span>
                                                        {/if}
                                                        <div class="flex flex-col">
                                                            <span
                                                                class="font-medium text-foreground"
                                                            >
                                                                {earner.user_name}
                                                            </span>
                                                            <span
                                                                class="text-xs text-muted-foreground"
                                                            >
                                                                {earner.user_email}
                                                            </span>
                                                        </div>
                                                    </div>
                                                </td>
                                                <td
                                                    class="px-3 py-3 text-right text-muted-foreground tabular-nums"
                                                >
                                                    {earner.certificates}
                                                </td>
                                                <td
                                                    class="px-3 py-3 text-right text-muted-foreground tabular-nums"
                                                >
                                                    {earner.badges}
                                                </td>
                                                <td
                                                    class="px-5 py-3 text-right font-semibold text-foreground tabular-nums"
                                                >
                                                    {earner.total_credentials}
                                                </td>
                                            </tr>
                                        {/each}
                                    </tbody>
                                </table>
                            </div>
                        {/if}
                    </div>
                </div>

                <!-- Most-awarded badges -->
                <div
                    class="rounded-2xl border border-border bg-card overflow-hidden lift"
                >
                    <div class="px-5 py-4 border-b border-border">
                        <h3
                            class="text-sm font-semibold text-foreground flex items-center gap-2"
                        >
                            <Medal class="size-4 text-accent" />
                            Most-awarded badges
                        </h3>
                    </div>
                    {#if (credentialOverview.top_badges ?? []).length === 0}
                        <p
                            class="text-sm text-muted-foreground py-8 text-center"
                        >
                            No badges defined or awarded yet.
                        </p>
                    {:else}
                        <div class="overflow-x-auto">
                            <table class="w-full text-sm">
                                <thead
                                    class="text-left text-xs text-muted-foreground"
                                >
                                    <tr class="border-b border-border">
                                        <th class="px-5 py-3 font-medium">
                                            Badge
                                        </th>
                                        <th class="px-3 py-3 font-medium">
                                            Category
                                        </th>
                                        <th class="px-3 py-3 font-medium">
                                            Tier
                                        </th>
                                        <th
                                            class="px-5 py-3 font-medium text-right"
                                        >
                                            Awards
                                        </th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {#each credentialOverview.top_badges ?? [] as b}
                                        <tr
                                            class="border-b border-border/50 hover:bg-muted/30"
                                        >
                                            <td class="px-5 py-3">
                                                <span
                                                    class="font-medium text-foreground"
                                                >
                                                    {b.badge_name}
                                                </span>
                                            </td>
                                            <td
                                                class="px-3 py-3 text-muted-foreground capitalize"
                                            >
                                                {b.category}
                                            </td>
                                            <td class="px-3 py-3">
                                                <span
                                                    class="inline-flex items-center rounded px-1.5 py-0.5 text-[10px] capitalize {b.tier === 'gold' ? 'bg-warning/10 text-warning' : b.tier === 'silver' ? 'bg-muted text-muted-foreground' : 'bg-accent/10 text-accent'}"
                                                >
                                                    {b.tier}
                                                </span>
                                            </td>
                                            <td
                                                class="px-5 py-3 text-right font-semibold text-foreground tabular-nums"
                                            >
                                                {b.awards}
                                            </td>
                                        </tr>
                                    {/each}
                                </tbody>
                            </table>
                        </div>
                    {/if}
                </div>
            </section>
        {/if}

        <!-- ====== User Growth & Composition ====== -->
        {#if platformSubtab === "users" && userOverview}
            {@const uo = userOverview}
            {@const bulkPct = uo.total_users > 0 ? Math.round((uo.bulk_created / uo.total_users) * 100) : 0}
            <section class="flex flex-col gap-4">
                <div class="flex items-center gap-2">
                    <UserPlus class="size-4 text-primary" />
                    <h2 class="text-sm font-semibold text-foreground">
                        User Growth & Composition
                    </h2>
                </div>

                <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
                    <div class="motion-rise-in motion-stagger-1">
                        <StatCard
                            label="Total users"
                            value={uo.total_users ?? 0}
                            tone="primary"
                            hint="across the platform"
                        >
                            {#snippet icon()}
                                <Users class="size-3.5" />
                            {/snippet}
                        </StatCard>
                    </div>
                    <div class="motion-rise-in motion-stagger-2">
                        <StatCard
                            label="Bulk-created"
                            value={uo.bulk_created ?? 0}
                            tone="info"
                            hint={`${bulkPct}% of all users`}
                        >
                            {#snippet icon()}
                                <UserPlus class="size-3.5" />
                            {/snippet}
                        </StatCard>
                    </div>
                    <div class="motion-rise-in motion-stagger-3">
                        <StatCard
                            label="No department"
                            value={uo.orphan_count ?? 0}
                            tone="destructive"
                            hint="unassigned users"
                        >
                            {#snippet icon()}
                                <Shield class="size-3.5" />
                            {/snippet}
                        </StatCard>
                    </div>
                    <div class="motion-rise-in motion-stagger-4">
                        <StatCard
                            label="Departments"
                            value={(uo.by_department ?? []).filter((d: any) => d.department !== 'Unassigned').length}
                            tone="accent"
                            hint="with members"
                        >
                            {#snippet icon()}
                                <Building2 class="size-3.5" />
                            {/snippet}
                        </StatCard>
                    </div>
                </div>

                <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
                    <div class="rounded-2xl border border-border bg-card p-5 lift">
                        {#if userGrowth.length > 0}
                            <div class="h-80">
                                <canvas bind:this={userGrowthCanvas}></canvas>
                            </div>
                        {:else}
                            <div class="flex items-center justify-center h-80">
                                <p class="text-sm text-muted-foreground">
                                    No new signups recorded yet.
                                </p>
                            </div>
                        {/if}
                    </div>
                    <div class="rounded-2xl border border-border bg-card p-5 lift">
                        {#if (uo.by_role ?? []).length > 0}
                            <div class="h-80">
                                <canvas bind:this={userRoleCanvas}></canvas>
                            </div>
                        {:else}
                            <div class="flex items-center justify-center h-80">
                                <p class="text-sm text-muted-foreground">
                                    No user roles assigned.
                                </p>
                            </div>
                        {/if}
                    </div>
                </div>

                <!-- Users by department -->
                <div class="rounded-2xl border border-border bg-card overflow-hidden lift">
                    <div class="px-5 py-4 border-b border-border">
                        <div class="flex items-center justify-between">
                            <h3 class="text-sm font-semibold text-foreground flex items-center gap-2">
                                <Building2 class="size-4 text-muted-foreground" />
                                Users by department
                            </h3>
                            <button
                                type="button"
                                class="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
                                title="Export CSV"
                                onclick={() => exportCSV('users-by-department.csv', userOverview?.by_department ?? [])}
                            >
                                <Download class="size-3.5" />
                                CSV
                            </button>
                        </div>
                    </div>
                    {#if (uo.by_department ?? []).length === 0}
                        <p class="text-sm text-muted-foreground py-8 text-center">
                            No departments yet.
                        </p>
                    {:else}
                        <div class="overflow-x-auto">
                            <table class="w-full text-sm">
                                <thead class="text-left text-xs text-muted-foreground">
                                    <tr class="border-b border-border">
                                        <th class="px-5 py-3 font-medium">Department</th>
                                        <th class="px-5 py-3 font-medium text-right">Members</th>
                                        <th class="px-5 py-3 font-medium">Share</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {#each uo.by_department ?? [] as d}
                                        {@const share = uo.total_users > 0 ? Math.round((d.count / uo.total_users) * 100) : 0}
                                        <tr class="border-b border-border/50 hover:bg-muted/30">
                                            <td class="px-5 py-3 font-medium text-foreground">
                                                {d.department === 'Unassigned' ? 'Unassigned (no dept)' : d.department}
                                            </td>
                                            <td class="px-5 py-3 text-right text-muted-foreground tabular-nums">
                                                {d.count}
                                            </td>
                                            <td class="px-5 py-3">
                                                <div class="flex items-center gap-2">
                                                    <div class="w-24 h-2 rounded-full bg-muted overflow-hidden">
                                                        <div class="h-full rounded-full bg-info" style="width: {share}%"></div>
                                                    </div>
                                                    <span class="text-xs text-muted-foreground tabular-nums">{share}%</span>
                                                </div>
                                            </td>
                                        </tr>
                                    {/each}
                                </tbody>
                            </table>
                        </div>
                    {/if}
                </div>
            </section>
        {/if}

        <!-- ====== Learning Paths ====== -->
        {#if platformSubtab === "paths" && learningPathOverview}
            {@const lp = learningPathOverview.summary}
            <section class="flex flex-col gap-4">
                <div class="flex items-center gap-2">
                    <Route class="size-4 text-primary" />
                    <h2 class="text-sm font-semibold text-foreground">
                        Learning Paths
                    </h2>
                </div>

                <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
                    <div class="motion-rise-in motion-stagger-1">
                        <StatCard
                            label="Total paths"
                            value={lp.total_paths ?? 0}
                            tone="primary"
                            hint={`${lp.published_paths ?? 0} published`}
                        >
                            {#snippet icon()}
                                <Route class="size-3.5" />
                            {/snippet}
                        </StatCard>
                    </div>
                    <div class="motion-rise-in motion-stagger-2">
                        <StatCard
                            label="Enrollments"
                            value={lp.total_enrollments ?? 0}
                            tone="info"
                            hint="across all paths"
                        >
                            {#snippet icon()}
                                <Users class="size-3.5" />
                            {/snippet}
                        </StatCard>
                    </div>
                    <div class="motion-rise-in motion-stagger-3">
                        <StatCard
                            label="Completed"
                            value={lp.completed_count ?? 0}
                            tone="success"
                            hint={`${lp.completion_rate ?? 0}% completion rate`}
                        >
                            {#snippet icon()}
                                <CheckCircle class="size-3.5" />
                            {/snippet}
                        </StatCard>
                    </div>
                    <div class="motion-rise-in motion-stagger-4">
                        <StatCard
                            label="Avg progress"
                            value={lp.avg_progress ?? 0}
                            tone="accent"
                            suffix="%"
                            hint="across active enrollments"
                        >
                            {#snippet icon()}
                                <TrendingUp class="size-3.5" />
                            {/snippet}
                        </StatCard>
                    </div>
                </div>

                <div class="rounded-2xl border border-border bg-card overflow-hidden lift">
                    <div class="px-5 py-4 border-b border-border">
                        <h3 class="text-sm font-semibold text-foreground">Most popular paths</h3>
                    </div>
                    {#if (learningPathOverview.top_paths ?? []).length === 0}
                        <p class="text-sm text-muted-foreground py-8 text-center">
                            No learning paths with enrollments yet.
                        </p>
                    {:else}
                        <div class="overflow-x-auto">
                            <table class="w-full text-sm">
                                <thead class="text-left text-xs text-muted-foreground">
                                    <tr class="border-b border-border">
                                        <th class="px-5 py-3 font-medium">Path</th>
                                        <th class="px-3 py-3 font-medium text-right">Enrolled</th>
                                        <th class="px-3 py-3 font-medium text-right">Completed</th>
                                        <th class="px-5 py-3 font-medium">Completion</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {#each learningPathOverview.top_paths ?? [] as p}
                                        <tr class="border-b border-border/50 hover:bg-muted/30">
                                            <td class="px-5 py-3 font-medium text-foreground">
                                                {truncate(p.path_title || 'Untitled', 40)}
                                            </td>
                                            <td class="px-3 py-3 text-right text-muted-foreground tabular-nums">{p.enrolled}</td>
                                            <td class="px-3 py-3 text-right text-muted-foreground tabular-nums">{p.completed}</td>
                                            <td class="px-5 py-3">
                                                <div class="flex items-center gap-2">
                                                    <div class="w-16 h-2 rounded-full bg-muted overflow-hidden">
                                                        <div class="h-full rounded-full {p.completion_rate >= 70 ? 'bg-success' : p.completion_rate >= 40 ? 'bg-warning' : 'bg-destructive'}" style="width: {p.completion_rate}%"></div>
                                                    </div>
                                                    <span class="text-xs text-muted-foreground tabular-nums">{p.completion_rate}%</span>
                                                </div>
                                            </td>
                                        </tr>
                                    {/each}
                                </tbody>
                            </table>
                        </div>
                    {/if}
                </div>
            </section>
        {/if}

        <!-- ====== Content ====== -->
        {#if platformSubtab === "content" && contentOverview}
            {@const co = contentOverview.summary}
            {@const rh = contentOverview.review_health ?? {}}
            <section class="flex flex-col gap-4">
                <div class="flex items-center gap-2">
                    <Library class="size-4 text-primary" />
                    <h2 class="text-sm font-semibold text-foreground">
                        Content Inventory
                    </h2>
                </div>

                <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
                    <div class="motion-rise-in motion-stagger-1">
                        <StatCard
                            label="Courses"
                            value={co.courses_total ?? 0}
                            tone="primary"
                            hint={`${co.courses_published ?? 0} published · ${co.courses_draft ?? 0} draft`}
                        >
                            {#snippet icon()}
                                <Library class="size-3.5" />
                            {/snippet}
                        </StatCard>
                    </div>
                    <div class="motion-rise-in motion-stagger-2">
                        <StatCard
                            label="Documents"
                            value={co.documents_total ?? 0}
                            tone="info"
                            hint={`${co.documents_ready ?? 0} ready`}
                        >
                            {#snippet icon()}
                                <FileText class="size-3.5" />
                            {/snippet}
                        </StatCard>
                    </div>
                    <div class="motion-rise-in motion-stagger-3">
                        <StatCard
                            label="Failed docs"
                            value={co.documents_failed ?? 0}
                            tone="destructive"
                            hint="need attention"
                        >
                            {#snippet icon()}
                                <FileWarning class="size-3.5" />
                            {/snippet}
                        </StatCard>
                    </div>
                    <div class="motion-rise-in motion-stagger-4">
                        <StatCard
                            label="Awaiting review"
                            value={rh.pending ?? 0}
                            tone="warning"
                            hint="courses in review"
                        >
                            {#snippet icon()}
                                <Clock class="size-3.5" />
                            {/snippet}
                        </StatCard>
                    </div>
                </div>

                <!-- Top creators + review health -->
                <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
                    <div class="rounded-2xl border border-border bg-card overflow-hidden lift">
                        <div class="px-5 py-4 border-b border-border">
                            <h3 class="text-sm font-semibold text-foreground">Most prolific creators</h3>
                        </div>
                        {#if (contentOverview.top_creators ?? []).length === 0}
                            <p class="text-sm text-muted-foreground py-8 text-center">No content authored yet.</p>
                        {:else}
                            <div class="overflow-x-auto">
                                <table class="w-full text-sm">
                                    <thead class="text-left text-xs text-muted-foreground">
                                        <tr class="border-b border-border">
                                            <th class="px-5 py-3 font-medium">Creator</th>
                                            <th class="px-3 py-3 font-medium text-right">Courses</th>
                                            <th class="px-5 py-3 font-medium text-right">Documents</th>
                                        </tr>
                                    </thead>
                                    <tbody>
                                        {#each contentOverview.top_creators ?? [] as c}
                                            <tr class="border-b border-border/50 hover:bg-muted/30">
                                                <td class="px-5 py-3 font-medium text-foreground">{c.creator_name}</td>
                                                <td class="px-3 py-3 text-right text-muted-foreground tabular-nums">{c.courses}</td>
                                                <td class="px-5 py-3 text-right text-muted-foreground tabular-nums">{c.documents}</td>
                                            </tr>
                                        {/each}
                                    </tbody>
                                </table>
                            </div>
                        {/if}
                    </div>

                    <div class="rounded-2xl border border-border bg-card p-5 lift flex flex-col gap-3">
                        <h3 class="text-sm font-semibold text-foreground">Review queue health</h3>
                        <div class="flex flex-col gap-2">
                            {#each [['pending','Pending','bg-warning/10 text-warning'], ['approved','Approved','bg-success/10 text-success'], ['rejected','Rejected','bg-destructive/10 text-destructive'], ['none','Not submitted','bg-muted text-muted-foreground']] as [key, label, cls]}
                                <div class="flex items-center justify-between">
                                    <span class="inline-flex items-center rounded px-2 py-0.5 text-xs font-medium {cls}">{label}</span>
                                    <span class="text-sm font-semibold text-foreground tabular-nums">{rh[key] ?? 0}</span>
                                </div>
                            {/each}
                        </div>
                    </div>
                </div>
            </section>
        {/if}

        <!-- ====== Overview (at-a-glance) ====== -->
        {#if platformSubtab === "overview"}
        <!-- Stat Cards -->
        <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
            <div class="motion-rise-in motion-stagger-1">
                <StatCard
                    label="Active learners"
                    value={overview?.active_learners ?? 0}
                    tone="info"
                    hint={`of ${overview?.total_users ?? 0} total users`}
                >
                    {#snippet icon()}
                        <Users class="size-3.5" />
                    {/snippet}
                </StatCard>
            </div>
            <div class="motion-rise-in motion-stagger-2">
                <StatCard
                    label="Gia questions"
                    value={overview?.coach_queries ?? 0}
                    tone="success"
                    hint="total asked"
                >
                    {#snippet icon()}
                        <MessageSquare class="size-3.5" />
                    {/snippet}
                </StatCard>
            </div>
            <div class="motion-rise-in motion-stagger-3">
                <StatCard
                    label="Avg proficiency"
                    value={overview?.adaptive?.avg_theta ?? 0}
                    tone="accent"
                    decimals={2}
                    hint="theta score (−3 to +3)"
                >
                    {#snippet icon()}
                        <Target class="size-3.5" />
                    {/snippet}
                </StatCard>
            </div>
            <div class="motion-rise-in motion-stagger-4">
                <StatCard
                    label="Knowledge spread"
                    value={overview?.adaptive?.stddev_theta ?? 0}
                    tone="primary"
                    decimals={2}
                    hint="standard deviation"
                >
                    {#snippet icon()}
                        <Zap class="size-3.5" />
                    {/snippet}
                </StatCard>
            </div>
        </div>

        <!-- Chart Row 1 -->
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div class="rounded-2xl border border-border bg-card p-5 lift">
                {#if mostFailed.length > 0}
                    <div class="h-80">
                        <canvas
                            bind:this={failedCanvas}
                            role="img"
                            aria-label="Most challenging topics — stacked bar chart of wrong vs correct responses"
                        ></canvas>
                    </div>
                {:else}
                    <div class="flex items-center justify-center h-80">
                        <p class="text-sm text-muted-foreground">
                            No practice data available yet.
                        </p>
                    </div>
                {/if}
            </div>

            <div class="rounded-2xl border border-border bg-card p-5 lift">
                {#if courseEffectiveness.length > 0}
                    <div class="h-80">
                        <canvas
                            bind:this={effectivenessCanvas}
                            role="img"
                            aria-label="Content effectiveness — average score and learner count per course"
                        ></canvas>
                    </div>
                {:else}
                    <div class="flex items-center justify-center h-80">
                        <p class="text-sm text-muted-foreground">
                            No course data available yet.
                        </p>
                    </div>
                {/if}
            </div>
        </div>

        <!-- Chart Row 2 -->
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div class="rounded-2xl border border-border bg-card p-5 lift">
                {#if overview?.adaptive}
                    <div class="h-80">
                        <canvas bind:this={radarCanvas}></canvas>
                    </div>
                {:else}
                    <div class="flex items-center justify-center h-80">
                        <p class="text-sm text-muted-foreground">
                            No adaptive learning data available yet.
                        </p>
                    </div>
                {/if}
            </div>

            <div class="rounded-2xl border border-border bg-card p-5 lift">
                {#if coachUsage.length > 0}
                    <div class="h-80">
                        <canvas
                            bind:this={usageCanvas}
                            role="img"
                            aria-label="Gia coach usage over time — daily query volume"
                        ></canvas>
                    </div>
                {:else}
                    <div class="flex items-center justify-center h-80">
                        <p class="text-sm text-muted-foreground">
                            No Gia usage data yet.
                        </p>
                    </div>
                {/if}
            </div>
        </div>

        <!-- Audit Log Preview -->
        <div class="rounded-2xl border border-border bg-card p-5 lift">
            <div class="flex items-center justify-between mb-4">
                <h3
                    class="text-sm font-semibold text-foreground flex items-center gap-2"
                >
                    <ClipboardList class="size-4 text-muted-foreground" />
                    Recent Activity
                </h3>
                <button
                    class="text-xs text-primary hover:underline flex items-center gap-1"
                    onclick={() => goto("/audit-log")}
                >
                    View full audit log <ArrowRight class="size-3" />
                </button>
            </div>
            {#if auditLog.length === 0}
                <p class="text-sm text-muted-foreground py-4 text-center">
                    No audit events recorded yet.
                </p>
            {:else}
                <div class="overflow-x-auto">
                    <table class="w-full text-sm">
                        <thead>
                            <tr
                                class="text-left text-xs text-muted-foreground border-b border-border"
                            >
                                <th class="pb-2 pr-3 font-medium">Time</th>
                                <th class="pb-2 pr-3 font-medium">User</th>
                                <th class="pb-2 pr-3 font-medium">Action</th>
                                <th class="pb-2 font-medium">Details</th>
                            </tr>
                        </thead>
                        <tbody>
                            {#each auditLog.slice(0, 10) as entry}
                                <tr class="border-b border-border/50">
                                    <td
                                        class="py-2 pr-3 text-xs text-muted-foreground whitespace-nowrap"
                                    >
                                        {new Date(
                                            entry.created_at,
                                        ).toLocaleString()}
                                    </td>
                                    <td class="py-2 pr-3 text-xs whitespace-nowrap">
                                        {#if entry.user_name}
                                            <span class="text-foreground"
                                                >{entry.user_name}</span
                                            >
                                            <span class="text-muted-foreground ml-1"
                                                >({entry.user_role})</span
                                            >
                                        {:else}
                                            <span class="text-muted-foreground"
                                                >System</span
                                            >
                                        {/if}
                                    </td>
                                    <td class="py-2 pr-3 whitespace-nowrap">
                                        <span
                                            class="inline-block px-1.5 py-0.5 rounded text-xs font-medium {actionBadge(
                                                entry.action,
                                            )}"
                                        >
                                            {entry.action}
                                        </span>
                                    </td>
                                    <td
                                        class="py-2 text-xs text-muted-foreground max-w-xs truncate"
                                    >
                                        {formatDetails(entry)}
                                    </td>
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                </div>
            {/if}
        </div>
        {/if}

        <!-- Engagement (at-a-glance, under Overview) -->
        {#if platformSubtab === "overview" && engagementOverview}
            {@const eo = engagementOverview}
            <section class="flex flex-col gap-4">
                <div class="flex items-center gap-2">
                    <Flame class="size-4 text-primary" />
                    <h2 class="text-sm font-semibold text-foreground">
                        Engagement
                    </h2>
                </div>
                <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
                    <div class="motion-rise-in motion-stagger-1">
                        <StatCard
                            label="Active streaks (7d)"
                            value={eo.active_streaks ?? 0}
                            tone="warning"
                            hint={`avg ${eo.avg_streak_length ?? 0} days`}
                        >
                            {#snippet icon()}
                                <Flame class="size-3.5" />
                            {/snippet}
                        </StatCard>
                    </div>
                    <div class="motion-rise-in motion-stagger-2">
                        <StatCard
                            label="Longest streak"
                            value={eo.longest_streak ?? 0}
                            tone="accent"
                            suffix="d"
                            hint="best active run"
                        >
                            {#snippet icon()}
                                <Trophy class="size-3.5" />
                            {/snippet}
                        </StatCard>
                    </div>
                    <div class="motion-rise-in motion-stagger-3">
                        <StatCard
                            label="Practice sessions"
                            value={eo.total_practice_sessions ?? 0}
                            tone="info"
                            hint={`avg ${eo.avg_practice_score ?? 0}% score`}
                        >
                            {#snippet icon()}
                                <Target class="size-3.5" />
                            {/snippet}
                        </StatCard>
                    </div>
                    <div class="motion-rise-in motion-stagger-4">
                        <StatCard
                            label="Unread notifications"
                            value={eo.unread_notifications ?? 0}
                            tone="neutral"
                            hint="pending delivery"
                        >
                            {#snippet icon()}
                                <MessageSquare class="size-3.5" />
                            {/snippet}
                        </StatCard>
                    </div>
                </div>
            </section>
        {/if}
    {:else if activeTab === "team"}
        <!-- Team Overview Stats -->
        <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
            <div class="motion-rise-in motion-stagger-1">
                <StatCard
                    label="Departments"
                    value={teamOverview.length}
                    tone="info"
                    hint="including unassigned"
                >
                    {#snippet icon()}
                        <Building2 class="size-3.5" />
                    {/snippet}
                </StatCard>
            </div>
            <div class="motion-rise-in motion-stagger-2">
                <StatCard
                    label="Total members"
                    value={teamOverview.reduce((sum: number, d: any) => sum + d.total_members, 0)}
                    tone="primary"
                    hint="across all departments"
                >
                    {#snippet icon()}
                        <Users class="size-3.5" />
                    {/snippet}
                </StatCard>
            </div>
            <div class="motion-rise-in motion-stagger-3">
                <StatCard
                    label="Struggling"
                    value={strugglingLearners.length}
                    tone="warning"
                    hint="below 50% avg score"
                >
                    {#snippet icon()}
                        <AlertTriangle class="size-3.5" />
                    {/snippet}
                </StatCard>
            </div>
            <div class="motion-rise-in motion-stagger-4">
                <StatCard
                    label="Avg completion"
                    value={completionRates.length > 0
                        ? completionRates.reduce((sum: number, d: any) => sum + d.completion_rate, 0) / completionRates.length
                        : 0}
                    tone="success"
                    decimals={1}
                    hint="% across departments"
                >
                    {#snippet icon()}
                        <TrendingUp class="size-3.5" />
                    {/snippet}
                </StatCard>
            </div>
        </div>

        <!-- Department Overview Table -->
        <div class="rounded-2xl border border-border bg-card overflow-hidden lift">
            <div class="px-5 py-4 border-b border-border">
                <h3 class="text-sm font-semibold text-foreground flex items-center gap-2">
                    <Building2 class="size-4 text-muted-foreground" />
                    Department Overview
                </h3>
            </div>
            {#if teamOverview.length === 0}
                <p class="text-sm text-muted-foreground py-8 text-center">
                    No department data available yet.
                </p>
            {:else}
                <div class="overflow-x-auto">
                    <table class="w-full text-sm">
                        <thead>
                            <tr class="text-left text-xs text-muted-foreground border-b border-border">
                                <th class="px-5 py-3 font-medium">Department</th>
                                <th class="px-5 py-3 font-medium">Members</th>
                                <th class="px-5 py-3 font-medium">Active (30d)</th>
                                <th class="px-5 py-3 font-medium">Completed</th>
                                <th class="px-5 py-3 font-medium">Avg Progress</th>
                            </tr>
                        </thead>
                        <tbody>
                            {#each teamOverview as dept}
                                <tr class="border-b border-border/50 hover:bg-muted/30">
                                    <td class="px-5 py-3 font-medium text-foreground">{dept.department_name}</td>
                                    <td class="px-5 py-3 text-muted-foreground">{dept.total_members}</td>
                                    <td class="px-5 py-3 text-muted-foreground">{dept.active_learners}</td>
                                    <td class="px-5 py-3 text-muted-foreground">{dept.completed_courses}</td>
                                    <td class="px-5 py-3">
                                        <div class="flex items-center gap-2">
                                            <div class="w-20 h-1.5 rounded-full bg-muted overflow-hidden">
                                                <div class="h-full rounded-full bg-info transition-all" style="width: {Math.min(dept.avg_progress, 100)}%"></div>
                                            </div>
                                            <span class="text-xs text-muted-foreground">{dept.avg_progress.toFixed(0)}%</span>
                                        </div>
                                    </td>
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                </div>
            {/if}
        </div>

        <!-- Struggling Learners -->
        <div class="rounded-2xl border border-border bg-card overflow-hidden lift">
            <div class="px-5 py-4 border-b border-border">
                <h3 class="text-sm font-semibold text-foreground flex items-center gap-2">
                    <AlertTriangle class="size-4 text-warning" />
                    Learners Needing Attention
                </h3>
            </div>
            {#if strugglingLearners.length === 0}
                <p class="text-sm text-muted-foreground py-8 text-center">
                    All learners are on track. Great job!
                </p>
            {:else}
                <div class="overflow-x-auto">
                    <table class="w-full text-sm">
                        <thead>
                            <tr class="text-left text-xs text-muted-foreground border-b border-border">
                                <th class="px-5 py-3 font-medium">Learner</th>
                                <th class="px-5 py-3 font-medium">Department</th>
                                <th class="px-5 py-3 font-medium">Avg Score</th>
                                <th class="px-5 py-3 font-medium">Active Courses</th>
                                <th class="px-5 py-3 font-medium">Last Active</th>
                            </tr>
                        </thead>
                        <tbody>
                            {#each strugglingLearners as learner}
                                <tr class="border-b border-border/50 hover:bg-muted/30">
                                    <td class="px-5 py-3">
                                        <div class="flex flex-col">
                                            <span class="font-medium text-foreground">{learner.user_name}</span>
                                            <span class="text-xs text-muted-foreground">{learner.user_email}</span>
                                        </div>
                                    </td>
                                    <td class="px-5 py-3 text-muted-foreground">{learner.department}</td>
                                    <td class="px-5 py-3">
                                        <span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium {learner.avg_score < 30 ? 'bg-destructive/10 text-destructive' : 'bg-warning/10 text-warning'}">
                                            {learner.avg_score.toFixed(0)}%
                                        </span>
                                    </td>
                                    <td class="px-5 py-3 text-muted-foreground">{learner.active_courses}</td>
                                    <td class="px-5 py-3 text-xs text-muted-foreground">{learner.last_active}</td>
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                </div>
            {/if}
        </div>

        <!-- Completion Rates by Department -->
        <div class="rounded-2xl border border-border bg-card overflow-hidden lift">
            <div class="px-5 py-4 border-b border-border">
                <h3 class="text-sm font-semibold text-foreground flex items-center gap-2">
                    <TrendingUp class="size-4 text-muted-foreground" />
                    Completion Rates by Department
                </h3>
            </div>
            {#if completionRates.length === 0}
                <p class="text-sm text-muted-foreground py-8 text-center">
                    No enrollment data available yet.
                </p>
            {:else}
                <div class="overflow-x-auto">
                    <table class="w-full text-sm">
                        <thead>
                            <tr class="text-left text-xs text-muted-foreground border-b border-border">
                                <th class="px-5 py-3 font-medium">Department</th>
                                <th class="px-5 py-3 font-medium">Enrolled</th>
                                <th class="px-5 py-3 font-medium">Completed</th>
                                <th class="px-5 py-3 font-medium">Rate</th>
                            </tr>
                        </thead>
                        <tbody>
                            {#each completionRates as cr}
                                <tr class="border-b border-border/50 hover:bg-muted/30">
                                    <td class="px-5 py-3 font-medium text-foreground">{cr.department_name}</td>
                                    <td class="px-5 py-3 text-muted-foreground">{cr.enrolled}</td>
                                    <td class="px-5 py-3 text-muted-foreground">{cr.completed}</td>
                                    <td class="px-5 py-3">
                                        <div class="flex items-center gap-2">
                                            <div class="w-16 h-2 rounded-full bg-muted overflow-hidden">
                                                <div class="h-full rounded-full transition-all {cr.completion_rate >= 70 ? 'bg-success' : cr.completion_rate >= 40 ? 'bg-warning' : 'bg-destructive'}" style="width: {cr.completion_rate}%"></div>
                                            </div>
                                            <span class="text-xs font-medium {cr.completion_rate >= 70 ? 'text-success' : cr.completion_rate >= 40 ? 'text-warning' : 'text-destructive'}">{cr.completion_rate}%</span>
                                        </div>
                                    </td>
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                </div>
            {/if}
        </div>
    {/if}
</div>
