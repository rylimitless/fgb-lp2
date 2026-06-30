<script lang="ts">
    import { cn } from "$lib/utils.js";
    import {
        Play,
        RotateCcw,
        CheckCircle,
        Layers,
        ArrowRight,
        Lock,
        Clock,
        Repeat,
        AlertTriangle,
    } from "@lucide/svelte";
    import * as Button from "$lib/components/ui/button";

    type Course = {
        id: number;
        title: string;
        description?: string;
        source_doc_ids?: number[];
        status?: string;
        settings?: any;
        progress?: {
            current_module?: number;
            completed?: boolean;
            score_pct?: string | number;
            is_expired?: boolean;
            days_overdue?: number;
            days_left?: number;
        } | null;
    };

    type Props = {
        course: Course;
        // Image override; when omitted the component auto-resolves from the
        // course title's slug under /brand/pathways/{slug}.png (Higgsfield
        // renders) or /brand/modules/{slug}.png if you add module-cover renders.
        // Pass `image=""` (empty string) to force the programmatic gradient.
        image?: string;
        // Compact density for horizontal scroll rows (Netflix style)
        compact?: boolean;
        onenroll?: (id: number) => void;
        class?: string;
    };

    let {
        course,
        image,
        compact = false,
        onenroll,
        class: className,
    }: Props = $props();

    // Map common course titles to FGB pathway slugs so a generic "Cybersecurity
    // Foundations" course picks up the Higgsfield cybersecurity pathway hero.
    // First exact match wins; otherwise we slugify the title and check
    // /brand/modules/{slug}.png. Empty string skips auto-resolution.
    const PATHWAY_HINTS: Array<[RegExp, string]> = [
        [/cyber|security|infosec/i, "cybersecurity"],
        [/compliance|regulator|policy|aml/i, "compliance"],
        [/risk|credit|portfolio/i, "risk-credit"],
        [/leadership|executive|management/i, "leadership"],
        [/customer|service|relationship|retail/i, "customer-service"],
        [/banking|foundation|introduction|fundamental/i, "banking-foundations"],
    ];

    function slugify(s: string): string {
        return (s ?? "")
            .toLowerCase()
            .replace(/[^a-z0-9]+/g, "-")
            .replace(/^-|-$/g, "");
    }

    function resolveImage(
        c: Course,
        override: string | undefined,
    ): string | undefined {
        if (override !== undefined) return override || undefined;
        const title = c.title ?? "";
        for (const [pattern, slug] of PATHWAY_HINTS) {
            if (pattern.test(title)) return `/brand/pathways/${slug}.png`;
        }
        const slug = slugify(title);
        return slug ? `/brand/modules/${slug}.png` : undefined;
    }

    let resolvedImage = $derived(resolveImage(course, image));

    // State derivation -----------------------------------------------------
    let isExpired = $derived(
        course.progress?.is_expired === true && !course.progress?.completed,
    );
    let state: "completed" | "in_progress" | "not_started" | "expired" =
        $derived(
            course.progress?.completed
                ? "completed"
                : isExpired
                  ? "expired"
                  : course.progress?.current_module !== undefined
                    ? "in_progress"
                    : "not_started",
        );

    let scoreNumber = $derived(() => {
        const raw = course.progress?.score_pct;
        const n = typeof raw === "string" ? parseFloat(raw) : (raw ?? 0);
        return Number.isFinite(n) ? n : 0;
    });

    // Deterministic gradient per course id when no Higgsfield image exists.
    // Same algorithm as Recommendations so a course reads identically wherever
    // it appears in the platform.
    function gradientFor(id: number): string {
        const seed = (id * 7919) % 360;
        const start = `oklch(0.42 0.13 ${(seed + 245) % 360})`;
        const end = `oklch(0.62 0.13 ${(seed + 220) % 360})`;
        return `linear-gradient(135deg, ${start} 0%, ${end} 100%)`;
    }

    function initialFor(title: string): string {
        return title?.[0]?.toUpperCase() ?? "?";
    }

    // Heuristic progress percentage when modules count isn't on the payload —
    // matches the lesson-player's existing approximation.
    let pct = $derived(
        course.progress?.completed
            ? 100
            : course.progress?.current_module !== undefined
              ? Math.min((course.progress.current_module / 5) * 100, 92)
              : 0,
    );

    let ctaLabel = $derived(
        state === "completed"
            ? "Retake"
            : state === "expired"
              ? "Deadline passed"
              : state === "in_progress"
                ? "Continue"
                : "Enroll",
    );

    let CtaIcon = $derived(
        state === "completed" ? RotateCcw : state === "expired" ? Lock : Play,
    );

    // Parse settings for display on card
    let parsedSettings = $derived(() => {
        const s = course.settings;
        if (!s || s === "{}") return {};
        try {
            return typeof s === "string" ? JSON.parse(s) : s;
        } catch {
            return {};
        }
    });
    let maxAttempts = $derived(parsedSettings().max_attempts ?? 0);
    let courseDaysToComplete = $derived(parsedSettings().days_to_complete ?? 0);
    // Per-user remaining days from the API (only set when user is actively enrolled)
    let remainingDays = $derived(
        course.progress?.days_left as number | undefined,
    );
    let hasSettings = $derived(maxAttempts > 0 || courseDaysToComplete > 0);

    function handleClick(e: MouseEvent) {
        // Don't allow enrolling in expired courses
        if (state === "expired") {
            e.preventDefault();
            return;
        }
        // If a click handler is supplied, prevent default link and call it.
        if (onenroll) {
            e.preventDefault();
            onenroll(course.id);
        }
    }
</script>

<article
    class={cn(
        "group relative flex flex-col overflow-hidden rounded-2xl border border-border bg-card lift press motion-rise-in",
        compact ? "w-[280px] sm:w-[320px] shrink-0" : "",
        className,
    )}
>
    <a
        href={`/lesson-player`}
        onclick={handleClick}
        class="flex flex-col h-full focus:outline-none focus-visible:ring-2 focus-visible:ring-ring/40 rounded-2xl"
        aria-label={`${ctaLabel} course: ${course.title}`}
    >
        <!-- Thumbnail -->
        <div
            class="relative aspect-video w-full overflow-hidden"
            style:background={gradientFor(course.id)}
        >
            <!-- Programmatic gradient + initial sits underneath; image (if any)
			     overlays it. If the image 404s we hide it so the gradient shows. -->
            <span
                class="absolute inset-0 flex items-center justify-center text-primary-foreground/85 text-5xl font-bold tracking-tight"
                aria-hidden="true"
            >
                {initialFor(course.title)}
            </span>
            {#if resolvedImage}
                <img
                    src={resolvedImage}
                    alt=""
                    class="absolute inset-0 size-full object-cover transition-transform duration-500 group-hover:scale-[1.04]"
                    loading="lazy"
                    decoding="async"
                    onerror={(e) =>
                        ((e.currentTarget as HTMLImageElement).style.display =
                            "none")}
                />
                <!-- Bottom gradient scrim for title legibility when image is present -->
                <div
                    class="absolute inset-0 bg-gradient-to-t from-black/65 via-transparent to-transparent"
                ></div>
            {/if}

            <!-- Status chip top-right -->
            <span
                class={cn(
                    "absolute top-2 right-2 inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider backdrop-blur",
                    state === "completed"
                        ? "bg-success/80 text-success-foreground"
                        : state === "expired"
                          ? "bg-destructive/85 text-destructive-foreground"
                          : state === "in_progress"
                            ? "bg-accent/85 text-accent-foreground"
                            : "bg-background/30 text-primary-foreground",
                )}
            >
                {#if state === "completed"}
                    <CheckCircle class="size-2.5" />
                    Complete
                {:else if state === "expired"}
                    <AlertTriangle class="size-2.5" />
                    Expired
                {:else if state === "in_progress"}
                    In progress
                {:else}
                    New
                {/if}
            </span>

            <!-- Source count chip bottom-right -->
            {#if course.source_doc_ids?.length}
                <span
                    class="absolute bottom-2 right-2 inline-flex items-center gap-1 rounded-full bg-background/30 backdrop-blur px-2 py-0.5 text-[10px] font-medium text-primary-foreground tabular"
                >
                    <Layers class="size-3" />
                    {course.source_doc_ids.length}
                </span>
            {/if}

            <!-- Progress bar overlay at bottom when in-progress / complete -->
            {#if pct > 0}
                <div
                    class="absolute inset-x-0 bottom-0 h-1 bg-background/20"
                    role="presentation"
                >
                    <div
                        class="h-full bg-accent transition-all duration-500"
                        style:width="{pct}%"
                    ></div>
                </div>
            {/if}
        </div>

        <!-- Body -->
        <div class="flex flex-1 flex-col gap-2 p-4">
            <h3
                class="text-base font-semibold text-foreground leading-snug line-clamp-2"
            >
                {course.title}
            </h3>
            {#if course.description}
                <p
                    class="text-xs text-muted-foreground leading-relaxed line-clamp-2"
                >
                    {course.description}
                </p>
            {/if}

            {#if hasSettings || remainingDays !== undefined}
                <div class="flex items-center gap-2 flex-wrap">
                    {#if maxAttempts > 0}
                        <span
                            class="inline-flex items-center gap-1 rounded-md bg-muted/60 px-1.5 py-0.5 text-[10px] text-muted-foreground tabular"
                        >
                            <Repeat class="size-2.5" />
                            {maxAttempts} attempt{maxAttempts !== 1 ? "s" : ""}
                        </span>
                    {/if}
                    {#if remainingDays !== undefined && remainingDays >= 0}
                        <!-- Per-user countdown: prominent, live from enrollment -->
                        <span
                            class="inline-flex items-center gap-1.5 rounded-md px-2 py-1 text-xs font-semibold tabular {remainingDays <=
                            1
                                ? 'bg-destructive/15 text-destructive'
                                : remainingDays <= 3
                                  ? 'bg-warning/15 text-warning'
                                  : 'bg-accent/15 text-accent'}"
                        >
                            <Clock class="size-3" />
                            {remainingDays === 0
                                ? "Due today"
                                : `${remainingDays} day${remainingDays !== 1 ? "s" : ""} left`}
                        </span>
                    {:else if courseDaysToComplete > 0}
                        <!-- Course-level total (user not enrolled yet) -->
                        <span
                            class="inline-flex items-center gap-1 rounded-md bg-muted/60 px-1.5 py-0.5 text-[10px] text-muted-foreground tabular"
                        >
                            <Clock class="size-2.5" />
                            {courseDaysToComplete} day{courseDaysToComplete !==
                            1
                                ? "s"
                                : ""} to complete
                        </span>
                    {/if}
                </div>
            {/if}

            {#if state === "completed"}
                <p class="text-[11px] text-success font-medium tabular mt-auto">
                    Scored {Math.round(scoreNumber())}%
                </p>
            {:else if state === "expired"}
                <p
                    class="text-[11px] text-destructive font-medium tabular mt-auto"
                >
                    Overdue by {course.progress?.days_overdue ?? "?"} day(s)
                </p>
            {:else if state === "in_progress"}
                <p class="text-[11px] text-muted-foreground tabular mt-auto">
                    Module {(course.progress?.current_module ?? 0) + 1} · {Math.round(
                        pct,
                    )}%
                </p>
            {:else}
                <p class="text-[11px] text-muted-foreground tabular mt-auto">
                    Not started
                </p>
            {/if}

            <div class="mt-2 flex items-center justify-between">
                <Button.Root
                    size="default"
                    variant={state === "completed"
                        ? "outline"
                        : state === "expired"
                          ? "ghost"
                          : "default"}
                    disabled={state === "expired"}
                >
                    <CtaIcon class="size-3.5" />
                    {ctaLabel}
                </Button.Root>
                <span
                    class="text-muted-foreground/50 group-hover:text-primary group-hover:translate-x-0.5 transition-all duration-200"
                    aria-hidden="true"
                >
                    <ArrowRight class="size-4" />
                </span>
            </div>
        </div>
    </a>
</article>
