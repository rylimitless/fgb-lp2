<script lang="ts">
    import { page } from "$app/stores";
    import { onMount } from "svelte";
    import {
        Award,
        Trophy,
        Calendar,
        CheckCircle,
        Shield,
        ArrowRight,
        Sparkles,
    } from "@lucide/svelte";

    let { data } = $props();

    let certificate = $derived(data.certificate);

    function formatDate(dateStr: string | null): string {
        if (!dateStr) return "—";
        const d = new Date(dateStr);
        return d.toLocaleDateString("en-US", {
            month: "long",
            day: "numeric",
            year: "numeric",
        });
    }

    function tierColor(tier: string): string {
        switch (tier) {
            case "gold":
                return "text-accent bg-accent/10 border-accent/30";
            case "silver":
                return "text-sky-400 bg-sky-400/10 border-sky-400/30";
            case "bronze":
                return "text-amber-600 bg-amber-600/10 border-amber-600/30";
            default:
                return "text-muted-foreground bg-muted border-border";
        }
    }

    function tierLabel(tier: string): string {
        return tier.charAt(0).toUpperCase() + tier.slice(1);
    }
</script>

<svelte:head>
    <title>Certificate Verification — FGB Academy</title>
</svelte:head>

<div
    class="flex min-h-screen items-center justify-center bg-gradient-to-br from-surface-1 via-card to-surface-2 p-4"
>
    {#if !certificate}
        <!-- Error state -->
        <div
            class="flex flex-col items-center gap-4 rounded-2xl border border-border bg-card p-10 text-center max-w-md"
        >
            <div
                class="flex size-16 items-center justify-center rounded-full bg-destructive/10"
            >
                <Shield class="size-8 text-destructive" />
            </div>
            <h2 class="text-xl font-semibold text-foreground">
                Certificate Not Found
            </h2>
            <p class="text-sm text-muted-foreground">
                This certificate code is invalid or the certificate has been
                revoked. Please check the URL and try again.
            </p>
            <a
                href="/welcome"
                class="inline-flex items-center gap-1.5 text-sm font-medium text-primary hover:underline"
            >
                Back to FGB Academy <ArrowRight class="size-3" />
            </a>
        </div>
    {:else}
        <!-- Certificate display -->
        <div
            class="relative w-full max-w-2xl overflow-hidden rounded-3xl border-2 border-accent/40 bg-card shadow-2xl motion-rise-in"
        >
            <!-- Decorative top bar -->
            <div
                class="absolute inset-x-0 top-0 h-2 bg-gradient-to-r from-accent via-primary to-accent"
            ></div>

            <!-- Watermark -->
            <div
                class="pointer-events-none absolute inset-0 flex items-center justify-center opacity-[0.03]"
                aria-hidden="true"
            >
                <Award class="size-96" />
            </div>

            <div class="relative z-10 flex flex-col gap-8 p-8 md:p-12">
                <!-- Header -->
                <div class="flex items-center justify-between">
                    <div class="flex items-center gap-3">
                        <div
                            class="flex size-12 items-center justify-center rounded-xl bg-accent/10"
                        >
                            <Award class="size-6 text-accent" />
                        </div>
                        <div>
                            <p
                                class="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground"
                            >
                                FGB Academy
                            </p>
                            <h1 class="text-2xl font-bold text-foreground">
                                Certificate of Completion
                            </h1>
                        </div>
                    </div>
                    <div class="flex items-center gap-2">
                        <span
                            class={tierColor(certificate.tier) +
                                " inline-flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs font-semibold"}
                        >
                            <Trophy class="size-3.5" />
                            {tierLabel(certificate.tier)}
                        </span>
                    </div>
                </div>

                <!-- Main content -->
                <div class="flex flex-col gap-6">
                    <div class="flex flex-col gap-2 text-center">
                        <p class="text-sm text-muted-foreground">
                            This certifies that
                        </p>
                        <h2 class="text-3xl font-bold text-foreground">
                            {certificate.user_name}
                        </h2>
                        <p class="text-sm text-muted-foreground">
                            has successfully completed the course
                        </p>
                        <h3 class="text-xl font-semibold text-accent">
                            {certificate.course_title}
                        </h3>
                    </div>

                    <!-- Stats grid -->
                    <div
                        class="grid grid-cols-3 gap-4 rounded-2xl border border-border bg-muted/30 p-5"
                    >
                        <div
                            class="flex flex-col items-center gap-1 text-center"
                        >
                            <span class="text-xs text-muted-foreground"
                                >Score</span
                            >
                            <span
                                class="text-2xl font-bold tabular text-foreground"
                            >
                                {Math.round(certificate.score_pct)}%
                            </span>
                        </div>
                        <div
                            class="flex flex-col items-center gap-1 text-center"
                        >
                            <span class="text-xs text-muted-foreground"
                                >Tier</span
                            >
                            <span
                                class="text-lg font-bold capitalize text-accent"
                            >
                                {certificate.tier}
                            </span>
                        </div>
                        <div
                            class="flex flex-col items-center gap-1 text-center"
                        >
                            <span class="text-xs text-muted-foreground"
                                >Issued</span
                            >
                            <span class="text-xs font-semibold text-foreground">
                                {formatDate(certificate.issued_at)}
                            </span>
                        </div>
                    </div>

                    <!-- Verification -->
                    <div
                        class="flex flex-col items-center gap-2 rounded-xl border border-dashed border-accent/30 bg-accent/5 p-4"
                    >
                        <div
                            class="flex items-center gap-2 text-xs text-accent"
                        >
                            <CheckCircle class="size-4" />
                            <span class="font-semibold"
                                >Verified Certificate</span
                            >
                        </div>
                        <code
                            class="rounded-md bg-muted px-3 py-1 font-mono text-xs text-muted-foreground"
                        >
                            {certificate.certificate_code}
                        </code>
                        <p class="text-[11px] text-muted-foreground">
                            This credential was issued by FGB Academy and can be
                            verified at any time.
                        </p>
                    </div>
                </div>

                <!-- Footer -->
                <div
                    class="flex items-center justify-between border-t border-border pt-4"
                >
                    <div
                        class="flex items-center gap-2 text-xs text-muted-foreground"
                    >
                        <Calendar class="size-3" />
                        <span>Issued {formatDate(certificate.issued_at)}</span>
                    </div>
                    <div
                        class="flex items-center gap-1 text-xs text-muted-foreground"
                    >
                        <Sparkles class="size-3 text-accent" />
                        <span>FGB Academy</span>
                    </div>
                </div>
            </div>
        </div>
    {/if}
</div>
