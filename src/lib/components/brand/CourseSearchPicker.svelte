<script lang="ts">
    import { cn } from "$lib/utils.js";
    import { Search, CheckCircle, X } from "@lucide/svelte";

    /**
     * CourseSearchPicker — fuzzy-search selector for picking from a list of
     * courses. Replaces a plain <select> when the catalog is large.
     *
     * Usage:
     *   <CourseSearchPicker
     *     items={publishedCourses}
     *     placeholder="Search courses..."
     *     onselect={(course) => selected = course}
     *   />
     */

    export type PickerItem = {
        id: number;
        title: string;
        description?: string;
    };

    type Props = {
        items: PickerItem[];
        placeholder?: string;
        emptyText?: string;
        /** ids to hide from the list (e.g. already-added courses) */
        excludeIds?: Set<number>;
        onselect: (item: PickerItem) => void;
        class?: string;
    };

    let {
        items,
        placeholder = "Search...",
        emptyText = "No matches.",
        excludeIds = new Set<number>(),
        onselect,
        class: className,
    }: Props = $props();

    let query = $state("");
    let highlighted = $state(0);
    let open = $state(false);

    // fzf-style fuzzy subsequence match with scoring.
    // Returns null if no match, else a score (lower = better).
    function fuzzyScore(haystack: string, needle: string): number | null {
        if (needle === "") return 0;
        const h = haystack.toLowerCase();
        const n = needle.toLowerCase();
        let score = 0;
        let hi = 0; // haystack index
        let ni = 0; // needle index
        let prevMatched = false;
        let bestRun = 0;
        let run = 0;
        for (; hi < h.length && ni < n.length; hi++) {
            if (h[hi] === n[ni]) {
                ni++;
                // bonus for contiguous runs
                run++;
                if (run > bestRun) bestRun = run;
                // bonus for matching at word boundaries
                if (hi === 0 || /[\s\-_/.]/.test(h[hi - 1])) score -= 4;
                // bonus for contiguous
                if (prevMatched) score -= 2;
                prevMatched = true;
            } else {
                run = 0;
                prevMatched = false;
                score += 1; // gap penalty
            }
        }
        if (ni < n.length) return null; // didn't match all needle chars
        // prefer earlier matches and longer contiguous runs
        return score - bestRun * 3;
    }

    let filtered = $derived.by(() => {
        const q = query.trim();
        const out: { item: PickerItem; score: number }[] = [];
        for (const it of items) {
            if (excludeIds.has(it.id)) continue;
            if (q === "") {
                out.push({ item: it, score: 0 });
                continue;
            }
            const titleScore = fuzzyScore(it.title, q);
            const descScore = it.description
                ? fuzzyScore(it.description, q)
                : null;
            // take the best (lowest) score between title and description,
            // but prefer title matches.
            let score: number | null = titleScore;
            if (descScore !== null && (score === null || descScore < score)) {
                score = descScore + 5; // slight penalty for desc-only matches
            }
            if (score !== null) out.push({ item: it, score });
        }
        out.sort((a, b) => a.score - b.score);
        return out.slice(0, 30).map((o) => o.item);
    });

    $effect(() => {
        // reset highlight when the list/filter changes
        highlighted = 0;
    });

    function handleKeydown(e: KeyboardEvent) {
        if (e.key === "ArrowDown") {
            e.preventDefault();
            highlighted = Math.min(highlighted + 1, filtered.length - 1);
        } else if (e.key === "ArrowUp") {
            e.preventDefault();
            highlighted = Math.max(highlighted - 1, 0);
        } else if (e.key === "Enter") {
            e.preventDefault();
            if (filtered[highlighted]) {
                pick(filtered[highlighted]);
            }
        } else if (e.key === "Escape") {
            open = false;
        }
    }

    function pick(item: PickerItem) {
        onselect(item);
        query = "";
        open = false;
    }
</script>

<div class={cn("relative", className)}>
    <div class="relative">
        <Search
            class="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
        />
        <input
            type="text"
            bind:value={query}
            onfocus={() => (open = true)}
            onkeydown={handleKeydown}
            {placeholder}
            class="w-full rounded-lg border border-border bg-muted py-2 pl-9 pr-9 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
        />
        {#if query}
            <button
                type="button"
                class="absolute right-2 top-1/2 -translate-y-1/2 rounded p-1 text-muted-foreground hover:bg-muted hover:text-foreground"
                onclick={() => (query = "")}
                aria-label="Clear search"
            >
                <X class="size-3.5" />
            </button>
        {/if}
    </div>

    {#if open && (query !== "" || filtered.length > 0)}
        <div
            class="absolute z-30 mt-1 max-h-64 w-full overflow-y-auto rounded-lg border border-border bg-card shadow-lg"
        >
            {#if filtered.length === 0}
                <p class="px-3 py-4 text-center text-sm text-muted-foreground">
                    {emptyText}
                </p>
            {:else}
                {#each filtered as item, i (item.id)}
                    <button
                        type="button"
                        class="flex w-full items-start gap-2 px-3 py-2 text-left transition-colors hover:bg-muted {i ===
                        highlighted
                            ? "bg-muted"
                            : ""}"
                        onmouseenter={() => (highlighted = i)}
                        onclick={() => pick(item)}
                    >
                        <CheckCircle
                            class="mt-0.5 size-4 shrink-0 text-primary opacity-0"
                        />
                        <div class="min-w-0 flex-1">
                            <p
                                class="truncate text-sm font-medium text-foreground"
                            >
                                {item.title}
                            </p>
                            {#if item.description}
                                <p
                                    class="truncate text-xs text-muted-foreground"
                                >
                                    {item.description}
                                </p>
                            {/if}
                        </div>
                    </button>
                {/each}
            {/if}
        </div>
    {/if}
</div>
