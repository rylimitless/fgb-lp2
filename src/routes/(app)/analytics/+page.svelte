<script lang="ts">
    import {
        BarChart3,
        Users,
        MessageSquare,
        Target,
        Zap,
    } from "@lucide/svelte";
    import { onMount } from "svelte";
    import { Chart, registerables } from "chart.js";

    Chart.register(...registerables);

    let { data } = $props();

    let overview = $derived(data.overview);
    let mostFailed = $derived(data.mostFailed ?? []);
    let courseEffectiveness = $derived(data.courseEffectiveness ?? []);
    let coachUsage = $derived(data.coachUsage ?? []);

    // Chart.js instances
    let failedCanvas = $state<HTMLCanvasElement>();
    let effectivenessCanvas = $state<HTMLCanvasElement>();
    let radarCanvas = $state<HTMLCanvasElement>();
    let usageCanvas = $state<HTMLCanvasElement>();

    const accentColors = [
        "#ef4444", // red
        "#f97316", // orange
        "#eab308", // yellow
        "#22c55e", // green
        "#06b6d4", // cyan
        "#3b82f6", // blue
        "#8b5cf6", // violet
        "#ec4899", // pink
        "#64748b", // slate
        "#14b8a6", // teal
    ];

    function truncate(str: string, len: number): string {
        return str.length > len ? str.slice(0, len) + "…" : str;
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
                            backgroundColor: "#ef444480",
                            borderColor: "#ef4444",
                            borderWidth: 1,
                        },
                        {
                            label: "Correct",
                            data: correctData,
                            backgroundColor: "#22c55e80",
                            borderColor: "#22c55e",
                            borderWidth: 1,
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
                            borderColor: "#f59e0b",
                            backgroundColor: "#f59e0b33",
                            borderWidth: 2,
                            pointRadius: 3,
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
                            backgroundColor: "#3b82f633",
                            borderColor: "#3b82f6",
                            borderWidth: 2,
                            pointBackgroundColor: "#3b82f6",
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
                            borderColor: "#8b5cf6",
                            backgroundColor: "#8b5cf633",
                            fill: true,
                            tension: 0.3,
                            pointRadius: 2,
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
    <div class="flex items-center gap-3">
        <BarChart3 class="size-6 text-primary" />
        <h1 class="text-2xl font-semibold text-foreground">Analytics</h1>
    </div>

    <!-- Stat Cards -->
    <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
        <div class="rounded-xl border border-border bg-card p-4">
            <div class="flex items-center gap-2 mb-1">
                <Users class="size-4 text-blue-500" />
                <span class="text-xs text-muted-foreground"
                    >Active Learners</span
                >
            </div>
            <p class="text-2xl font-bold text-foreground">
                {overview?.active_learners ?? 0}
            </p>
            <p class="text-xs text-muted-foreground mt-0.5">
                of {overview?.total_users ?? 0} total users
            </p>
        </div>
        <div class="rounded-xl border border-border bg-card p-4">
            <div class="flex items-center gap-2 mb-1">
                <MessageSquare class="size-4 text-emerald-500" />
                <span class="text-xs text-muted-foreground">GIA Questions</span>
            </div>
            <p class="text-2xl font-bold text-foreground">
                {overview?.coach_queries ?? 0}
            </p>
            <p class="text-xs text-muted-foreground mt-0.5">total asked</p>
        </div>
        <div class="rounded-xl border border-border bg-card p-4">
            <div class="flex items-center gap-2 mb-1">
                <Target class="size-4 text-amber-500" />
                <span class="text-xs text-muted-foreground"
                    >Avg Proficiency</span
                >
            </div>
            <p class="text-2xl font-bold text-foreground">
                {overview?.adaptive?.avg_theta?.toFixed(2) ?? "—"}
            </p>
            <p class="text-xs text-muted-foreground mt-0.5">theta score</p>
        </div>
        <div class="rounded-xl border border-border bg-card p-4">
            <div class="flex items-center gap-2 mb-1">
                <Zap class="size-4 text-violet-500" />
                <span class="text-xs text-muted-foreground"
                    >Knowledge Spread</span
                >
            </div>
            <p class="text-2xl font-bold text-foreground">
                {overview?.adaptive?.stddev_theta?.toFixed(2) ?? "—"}
            </p>
            <p class="text-xs text-muted-foreground mt-0.5">std deviation</p>
        </div>
    </div>

    <!-- Chart Row 1 -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div class="rounded-xl border border-border bg-card p-5">
            {#if mostFailed.length > 0}
                <div class="h-80">
                    <canvas bind:this={failedCanvas}></canvas>
                </div>
            {:else}
                <div class="flex items-center justify-center h-80">
                    <p class="text-sm text-muted-foreground">
                        No practice data available yet.
                    </p>
                </div>
            {/if}
        </div>

        <div class="rounded-xl border border-border bg-card p-5">
            {#if courseEffectiveness.length > 0}
                <div class="h-80">
                    <canvas bind:this={effectivenessCanvas}></canvas>
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
        <div class="rounded-xl border border-border bg-card p-5">
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

        <div class="rounded-xl border border-border bg-card p-5">
            {#if coachUsage.length > 0}
                <div class="h-80">
                    <canvas bind:this={usageCanvas}></canvas>
                </div>
            {:else}
                <div class="flex items-center justify-center h-80">
                    <p class="text-sm text-muted-foreground">
                        No GIA usage data yet. Ask GIA a question to populate
                        this chart.
                    </p>
                </div>
            {/if}
        </div>
    </div>
</div>
