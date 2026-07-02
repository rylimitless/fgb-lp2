<script lang="ts">
    import { cn } from "$lib/utils.js";
    import { CheckCircle, XCircle, AlertCircle } from "@lucide/svelte";
    import type { Snippet } from "svelte";

    type Status = "correct" | "incorrect" | "unanswered";

    type Props = {
        status: Status;
        // One-line plain answer explanation, sentence-case per gia.md voice rules.
        correctAnswer?: string;
        // Optional extra explanation snippet for richer markup.
        explanation?: Snippet;
        // When true, suppresses the entire surface — useful for non-graded states.
        hidden?: boolean;
        // In learning mode: when true shows the correct answer; when false hides it after a check.
        revealAnswer?: boolean;
        class?: string;
    };

    let {
        status,
        correctAnswer,
        explanation,
        hidden = false,
        revealAnswer = true,
        class: className,
    }: Props = $props();

    const styleByStatus: Record<
        Status,
        {
            wrapper: string;
            label: string;
            icon: typeof CheckCircle;
            heading: string;
        }
    > = {
        correct: {
            wrapper: "border-success/30 bg-success/5 text-success",
            label: "text-success",
            icon: CheckCircle,
            heading: "Correct.",
        },
        incorrect: {
            wrapper: "border-destructive/30 bg-destructive/5 text-destructive",
            label: "text-destructive",
            icon: XCircle,
            heading: "Not quite.",
        },
        unanswered: {
            wrapper: "border-warning/30 bg-warning/5 text-warning",
            label: "text-warning",
            icon: AlertCircle,
            heading: "No answer recorded.",
        },
    };

    let s = $derived(styleByStatus[status]);
</script>

{#if !hidden}
    <div
        role="status"
        aria-live="polite"
        class={cn(
            "mt-3 flex items-start gap-2 rounded-xl border px-3 py-2 motion-rise-in",
            s.wrapper,
            className,
        )}
    >
        <s.icon class="size-4 mt-0.5 shrink-0" aria-hidden="true" />
        <div class="min-w-0 text-xs leading-relaxed">
            <span class={cn("font-semibold", s.label)}>{s.heading}</span>
            {#if status === "incorrect" && correctAnswer && revealAnswer}
                <span class="text-foreground/80 ml-1">
                    The right answer was <span
                        class="font-medium text-foreground"
                        >{correctAnswer}</span
                    >.
                </span>
            {:else if status === "incorrect" && !revealAnswer}
                <span class="text-foreground/80 ml-1">
                    Try again, or reveal the answer.
                </span>
            {:else if status === "unanswered"}
                <span class="text-foreground/80 ml-1">
                    This question was skipped.
                </span>
            {/if}
            {#if explanation}
                <div class="mt-1 text-foreground/75">
                    {@render explanation()}
                </div>
            {/if}
        </div>
    </div>
{/if}
