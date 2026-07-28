<script lang="ts">
    import {
        BarChart3,
        Users,
        MessageSquare,
        Target,
        Zap,
        ClipboardList,
        ArrowRight,
    } from "@lucide/svelte";
    import { onMount } from "svelte";
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
    let auditLog = $derived(data.auditLog ?? []);

    // Chart.js instances
    let failedCanvas = $state<HTMLCanvasElement>();
    let effectivenessCanvas = $state<HTMLCanvasElement>();
    let radarCanvas = $state<HTMLCanvasElement>();
    let usageCanvas = $state<HTMLCanvasElement>();

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

    onMount(() => {
        // 1. Most Failed Questions - horizontal bar
        if (failedCanvas && mostFailed.length > 0) {
            const labels = mostFailed.map((f: any) =>
                truncate(f.topic || "Untitled", 30),
            );
            const wrongData = mostFailed.map((f: any) => f.total_wrong ?? 0);
            const correctData = mostFailed.map(
                (f: any) => f.total_correct ?? 0,
            );

            new Chart(failedCanvas, {
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

            new Chart(effectivenessCanvas, {
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

            new Chart(radarCanvas, {
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

            new Chart(usageCanvas, {
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
                <!-- Screen-reader twin: exposes the chart data as a semantic
                     table. Same numbers Chart.js renders on the canvas. -->
                <table class="sr-only" aria-label="Most challenging topics">
                    <caption>
                        Wrong and correct response counts by topic
                    </caption>
                    <thead>
                        <tr>
                            <th scope="col">Topic</th>
                            <th scope="col">Wrong</th>
                            <th scope="col">Correct</th>
                        </tr>
                    </thead>
                    <tbody>
                        {#each mostFailed as f}
                            <tr>
                                <th scope="row">{f.topic ?? "Untitled"}</th>
                                <td>{f.total_wrong ?? 0}</td>
                                <td>{f.total_correct ?? 0}</td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
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
                <table
                    class="sr-only"
                    aria-label="Content effectiveness by course"
                >
                    <caption>
                        Average score and learner count per course
                    </caption>
                    <thead>
                        <tr>
                            <th scope="col">Course</th>
                            <th scope="col">Average score %</th>
                            <th scope="col">Learners</th>
                        </tr>
                    </thead>
                    <tbody>
                        {#each courseEffectiveness as c}
                            <tr>
                                <th scope="row"
                                    >{c.course_title ?? "Untitled"}</th
                                >
                                <td
                                    >{typeof c.avg_score === "string"
                                        ? parseFloat(c.avg_score)
                                        : (c.avg_score ?? 0)}</td
                                >
                                <td>{c.learner_count ?? 0}</td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
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
                <table
                    class="sr-only"
                    aria-label="Gia coach usage over time"
                >
                    <caption>Daily Gia coach query volume</caption>
                    <thead>
                        <tr>
                            <th scope="col">Date</th>
                            <th scope="col">Queries</th>
                        </tr>
                    </thead>
                    <tbody>
                        {#each coachUsage as u}
                            <tr>
                                <th scope="row"
                                    >{u.day ?? u.date ?? "—"}</th
                                >
                                <td>{u.count ?? u.queries ?? 0}</td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
            {:else}
                <div class="flex items-center justify-center h-80">
                    <p class="text-sm text-muted-foreground">
                        No Gia usage data yet. Ask Gia a question to populate
                        this chart.
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
</div>
