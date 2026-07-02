<script lang="ts">
    import { goto } from "$app/navigation";
    import {
        Award,
        Trophy,
        Medal,
        Star,
        Calendar,
        ArrowRight,
        ShieldCheck,
        Zap,
        Flame,
        BookOpen,
        Brain,
        Target,
        Sparkles,
        Crown,
        GraduationCap,
        type Icon as LucideIcon,
    } from "@lucide/svelte";
    import { cn } from "$lib/utils.js";
    import type { Component } from "svelte";
    import PageHeader from "$lib/components/brand/PageHeader.svelte";
    import BadgeMedal from "$lib/components/brand/BadgeMedal.svelte";

    let { data } = $props();

    let certificates: Array<{
        id: number;
        certificate_code: string;
        score_pct: number;
        tier: string;
        issued_at: string;
        course_title: string;
        user_name: string;
    }> = $derived(data.certificates ?? []);

    let earnedBadges: Array<{
        id: number;
        code: string;
        name: string;
        description: string;
        icon: string;
        tier: "bronze" | "silver" | "gold";
        category: string;
        earned_at: string | null;
    }> = $derived(data.earnedBadges ?? []);

    let earnedCount: number = $derived(data.earnedCount ?? 0);

    // Icon map for lucide icons driven by the badge's icon string
    const iconMap: Record<string, Component> = {
        award: Award as unknown as Component,
        trophy: Trophy as unknown as Component,
        medal: Medal as unknown as Component,
        star: Star as unknown as Component,
        "shield-check": ShieldCheck as unknown as Component,
        zap: Zap as unknown as Component,
        flame: Flame as unknown as Component,
        "book-open": BookOpen as unknown as Component,
        brain: Brain as unknown as Component,
        target: Target as unknown as Component,
        sparkles: Sparkles as unknown as Component,
        crown: Crown as unknown as Component,
        graduation: GraduationCap as unknown as Component,
    };

    function badgeIcon(iconName: string): Component | null {
        return iconMap[iconName] ?? null;
    }

    function formatDate(dateStr: string | null): string {
        if (!dateStr) return "—";
        const d = new Date(dateStr);
        return d.toLocaleDateString("en-US", {
            month: "short",
            day: "numeric",
            year: "numeric",
        });
    }

    function tierColor(tier: string): string {
        switch (tier) {
            case "gold":
                return "text-accent bg-accent/10 border-accent/30";
            case "silver":
                return "text-silver-foreground bg-silver/10 border-silver/30";
            case "bronze":
                return "text-bronze-foreground bg-bronze/10 border-bronze/30";
            default:
                return "text-muted-foreground bg-muted border-border";
        }
    }

    function tierLabel(tier: string): string {
        return tier.charAt(0).toUpperCase() + tier.slice(1);
    }

    function openCertificate(code: string) {
        goto(`/certificates/${code}`);
    }

    // Group badges by category for organized display
    let badgeCategories = $derived(
        earnedBadges.reduce(
            (acc, badge) => {
                const cat = badge.category || "General";
                if (!acc[cat]) acc[cat] = [];
                acc[cat].push(badge);
                return acc;
            },
            {} as Record<string, typeof earnedBadges>,
        ),
    );
</script>

<svelte:head>
    <title>Certificates & Badges — FGB Academy</title>
</svelte:head>

<div class="flex w-full max-w-5xl mx-auto flex-col gap-6">
    <!-- Page Header -->
    <PageHeader
        title="Certificates & Badges"
        eyebrow="Achievements"
        description="View your earned certificates and badges. Share your accomplishments with the world."
        variant="default"
    >
        {#snippet icon()}
            <div
                class="flex size-9 items-center justify-center rounded-xl bg-accent/10"
            >
                <Award class="size-5 text-accent" />
            </div>
        {/snippet}
    </PageHeader>

    <!-- Certificates Section -->
    <section class="flex flex-col gap-4">
        <div class="flex items-center gap-2">
            <div
                class="flex size-8 items-center justify-center rounded-lg bg-primary/10"
            >
                <Trophy class="size-4 text-primary" />
            </div>
            <h2 class="text-lg font-semibold text-foreground">Certificates</h2>
            <span class="text-xs text-muted-foreground tabular">
                ({certificates.length} earned)
            </span>
        </div>

        {#if certificates.length === 0}
            <div
                class="flex flex-col items-center gap-3 rounded-xl border border-dashed border-border p-10 text-center"
            >
                <Award class="size-10 text-muted-foreground/40" />
                <div>
                    <p class="text-sm font-medium text-muted-foreground">
                        No certificates yet
                    </p>
                    <p class="text-xs text-muted-foreground/60 mt-1">
                        Complete a course with a passing score to earn a
                        certificate.
                    </p>
                </div>
            </div>
        {:else}
            <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
                {#each certificates as cert}
                    <button
                        class="group flex flex-col gap-3 rounded-2xl border border-border bg-card p-5 text-left transition hover:border-primary/30 hover:shadow-md motion-rise-in"
                        onclick={() => openCertificate(cert.certificate_code)}
                    >
                        <!-- Tier badge + course name -->
                        <div class="flex items-start justify-between gap-2">
                            <div class="min-w-0 flex-1">
                                <p
                                    class="truncate text-sm font-semibold text-foreground"
                                >
                                    {cert.course_title}
                                </p>
                                <p class="mt-0.5 text-xs text-muted-foreground">
                                    {cert.user_name}
                                </p>
                            </div>
                            <span
                                class={cn(
                                    "inline-flex shrink-0 items-center gap-1 rounded-full border px-2 py-0.5 text-[11px] font-medium",
                                    tierColor(cert.tier),
                                )}
                            >
                                <Trophy class="size-3" />
                                {tierLabel(cert.tier)}
                            </span>
                        </div>

                        <!-- Score bar -->
                        <div class="flex items-center gap-2">
                            <div
                                class="h-1.5 flex-1 rounded-full bg-muted overflow-hidden"
                            >
                                <div
                                    class="h-full rounded-full bg-primary transition-all duration-500"
                                    style="width: {cert.score_pct}%"
                                ></div>
                            </div>
                            <span
                                class="text-xs font-semibold tabular text-foreground"
                            >
                                {Math.round(cert.score_pct)}%
                            </span>
                        </div>

                        <!-- Footer: code + date + action -->
                        <div class="flex items-center justify-between gap-2">
                            <div
                                class="flex items-center gap-2 text-[11px] text-muted-foreground"
                            >
                                <Calendar class="size-3" />
                                <span>{formatDate(cert.issued_at)}</span>
                            </div>
                            <span
                                class="inline-flex items-center gap-1 rounded-md bg-muted px-2 py-0.5 text-[10px] font-mono text-muted-foreground"
                            >
                                {cert.certificate_code}
                            </span>
                        </div>

                        <div
                            class="flex items-center justify-end gap-1 text-xs font-medium text-primary opacity-0 transition-opacity group-hover:opacity-100"
                        >
                            View certificate
                            <ArrowRight class="size-3" />
                        </div>
                    </button>
                {/each}
            </div>
        {/if}
    </section>

    <!-- Badges Section -->
    <section class="flex flex-col gap-4">
        <div class="flex items-center gap-2">
            <div
                class="flex size-8 items-center justify-center rounded-lg bg-accent/10"
            >
                <Medal class="size-4 text-accent" />
            </div>
            <h2 class="text-lg font-semibold text-foreground">Badges</h2>
            <span class="text-xs text-muted-foreground tabular">
                ({earnedCount} earned)
            </span>
        </div>

        {#if earnedBadges.length === 0}
            <div
                class="flex flex-col items-center gap-3 rounded-xl border border-dashed border-border p-10 text-center"
            >
                <Medal class="size-10 text-muted-foreground/40" />
                <div>
                    <p class="text-sm font-medium text-muted-foreground">
                        No badges earned yet
                    </p>
                    <p class="text-xs text-muted-foreground/60 mt-1">
                        Keep learning and completing courses to earn badges.
                    </p>
                </div>
            </div>
        {:else}
            {#each Object.entries(badgeCategories) as [category, badges]}
                <div class="flex flex-col gap-2">
                    <h3
                        class="text-xs font-semibold uppercase tracking-wider text-muted-foreground"
                    >
                        {category}
                    </h3>
                    <div
                        class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-4"
                    >
                        {#each badges as badge}
                            {@const IconComp = badgeIcon(badge.icon)}
                            <div
                                class={cn(
                                    "flex flex-col items-center gap-2 rounded-2xl border border-border bg-surface-1 p-4 text-center transition hover:border-accent/20",
                                    badge.earned_at
                                        ? "hover:shadow-sm motion-rise-in"
                                        : "opacity-50",
                                )}
                            >
                                {#if IconComp}
                                    {#snippet icon()}
                                        <IconComp class="size-5" />
                                    {/snippet}
                                    <BadgeMedal
                                        tier={badge.tier}
                                        size={56}
                                        locked={!badge.earned_at}
                                        {icon}
                                    />
                                {:else}
                                    <BadgeMedal
                                        tier={badge.tier}
                                        size={56}
                                        locked={!badge.earned_at}
                                    />
                                {/if}
                                <div class="flex flex-col gap-0.5">
                                    <span
                                        class="text-xs font-semibold text-foreground leading-tight"
                                    >
                                        {badge.name}
                                    </span>
                                    <span
                                        class="text-[10px] text-muted-foreground leading-tight"
                                    >
                                        {badge.description}
                                    </span>
                                    {#if badge.earned_at}
                                        <span
                                            class="text-[10px] text-accent mt-0.5"
                                        >
                                            {formatDate(badge.earned_at)}
                                        </span>
                                    {/if}
                                </div>
                            </div>
                        {/each}
                    </div>
                </div>
            {/each}
        {/if}
    </section>
</div>
