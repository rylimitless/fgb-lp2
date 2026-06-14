<script lang="ts">
    import { Brain, LoaderCircle, CheckCircle, XCircle, TrendingUp, TrendingDown, Target, ArrowRight, BookOpen, RotateCcw } from "@lucide/svelte";
    import * as Button from "$lib/components/ui/button";

    let courses = $state<any[]>([]);
    let loadingCourses = $state(true);
    let selectedCourseId = $state<number | null>(null);

    // Session state
    let theta = $state(0);
    let currentItem = $state<any>(null);
    let total = $state(0);
    let answered = $state(0);
    let sessionLoading = $state(false);
    let submitting = $state(false);
    let answer = $state<any>(undefined);
    let lastResult = $state<{ correct: boolean; delta: number; prob: number } | null>(null);
    let sessionDone = $state(false);

    $effect(() => { loadCourses(); });

    async function loadCourses() {
        loadingCourses = true;
        try {
            const res = await fetch("/api/courses", { credentials: "include" });
            if (res.ok) courses = (await res.json()).filter((c: any) => c.status === "published" && c.approved);
        } catch { /* ignore */ }
        loadingCourses = false;
    }

    async function startSession(courseId: number) {
        selectedCourseId = courseId;
        sessionLoading = true;
        sessionDone = false;
        answered = 0;
        lastResult = null;
        answer = undefined;
        try {
            const res = await fetch(`/api/adaptive/start?course_id=${courseId}`, { credentials: "include" });
            if (res.ok) {
                const data = await res.json();
                theta = data.theta;
                currentItem = data.item;
                total = data.total;
            }
        } catch { /* ignore */ }
        sessionLoading = false;
    }

    function selectCourse(courseId: number) {
        selectedCourseId = courseId;
        startSession(courseId);
    }

    function isAnswerCorrect(item: any, ans: any): boolean {
        const d = item.data;
        switch (item.item_type) {
            case "mc": return ans === d.correct;
            case "tf": return ans === d.answer;
            case "ma": {
                if (!Array.isArray(ans) || !d.correct) return false;
                const c = d.correct as number[];
                return ans.length === c.length && c.every((v: number) => ans.includes(v));
            }
            default: return true;
        }
    }

    async function submitAnswer() {
        if (submitting || !currentItem || answer === undefined) return;
        submitting = true;
        const correct = isAnswerCorrect(currentItem, answer);

        try {
            const res = await fetch("/api/adaptive/submit", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                credentials: "include",
                body: JSON.stringify({ course_id: selectedCourseId, item_id: currentItem.id, outcome: correct ? 1 : 0 }),
            });
            if (res.ok) {
                const data = await res.json();
                theta = data.theta;
                lastResult = { correct, delta: data.delta, prob: data.probability };
                answered++;
                currentItem = data.next_item;
                answer = undefined;
                if (!data.next_item) sessionDone = true;
            }
        } catch { /* ignore */ }
        submitting = false;
    }

    function resetSession() {
        if (selectedCourseId) startSession(selectedCourseId);
    }

    function thetaLabel(t: number): string {
        if (t < -1) return "Beginner";
        if (t < 0) return "Developing";
        if (t < 0.5) return "Proficient";
        if (t < 1.5) return "Advanced";
        return "Expert";
    }

    let thetaPct = $derived(((theta + 3) / 6) * 100);
</script>

<div class="flex w-full max-w-5xl mx-auto flex-col gap-6">
    <div class="flex items-center gap-3">
        <Brain class="size-6 text-primary" />
        <h1 class="text-2xl font-semibold text-foreground">Adaptive Learning Room</h1>
    </div>

    <!-- Course Selector -->
    {#if !selectedCourseId}
        <p class="text-sm text-muted-foreground">Select a course to begin an adaptive practice session.</p>
        {#if loadingCourses}
            <div class="flex justify-center py-8"><LoaderCircle class="size-6 text-muted-foreground animate-spin" /></div>
        {:else if courses.length === 0}
            <div class="rounded-xl border border-border bg-card p-8 text-center">
                <BookOpen class="size-8 text-muted-foreground/40 mx-auto mb-2" />
                <p class="text-sm text-muted-foreground">No published courses available.</p>
            </div>
        {:else}
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                {#each courses as course}
                    <button class="rounded-xl border border-border bg-card p-5 text-left hover:border-primary/30 transition-colors" onclick={() => selectCourse(course.id)}>
                        <BookOpen class="size-5 text-primary mb-2" />
                        <h3 class="text-sm font-semibold text-foreground">{course.title}</h3>
                        <p class="text-xs text-muted-foreground mt-1 line-clamp-2">{course.description}</p>
                        <div class="flex items-center gap-1.5 mt-3 text-xs text-primary">
                            <Target class="size-3" /> Start adaptive session
                        </div>
                    </button>
                {/each}
            </div>
        {/if}
    {:else if sessionLoading}
        <div class="flex justify-center py-12"><LoaderCircle class="size-6 text-muted-foreground animate-spin" /></div>
    {:else if sessionDone}
        <div class="rounded-xl border border-border bg-card p-10 text-center flex flex-col items-center gap-4">
            <Brain class="size-12 text-primary" />
            <h2 class="text-xl font-semibold text-foreground">Session Complete</h2>
            <p class="text-sm text-muted-foreground">You answered all {total} questions</p>
            <div class="text-3xl font-bold text-foreground">Theta: {theta}</div>
            <p class="text-xs text-muted-foreground">Level: {thetaLabel(theta)}</p>
            <div class="w-full max-w-xs h-2 rounded-full bg-muted overflow-hidden">
                <div class="h-full rounded-full bg-primary transition-all" style="width: {thetaPct}%"></div>
            </div>
            <div class="flex gap-3 mt-2">
                <Button.Root variant="outline" onclick={() => { selectedCourseId = null; }}>Back to Courses</Button.Root>
                <Button.Root onclick={resetSession}><RotateCcw class="size-4 mr-1.5" /> Retry</Button.Root>
            </div>
        </div>
    {:else}
        <!-- IRT Dashboard -->
        <div class="flex items-center gap-6">
            <div class="flex-1 rounded-xl border border-border bg-card p-4">
                <div class="flex items-center justify-between mb-2">
                    <span class="text-xs text-muted-foreground">Ability (θ)</span>
                    <span class="text-sm font-bold text-foreground">{theta}</span>
                </div>
                <div class="h-2 w-full rounded-full bg-muted overflow-hidden">
                    <div class="h-full rounded-full bg-primary transition-all duration-500" style="width: {thetaPct}%"></div>
                </div>
                <div class="flex justify-between text-[10px] text-muted-foreground mt-1">
                    <span>-3.0</span><span>{thetaLabel(theta)}</span><span>+3.0</span>
                </div>
            </div>
            <div class="flex items-center gap-4 text-sm text-muted-foreground">
                <span>Q {answered + 1} of {total}</span>
                {#if lastResult}
                    <span class="flex items-center gap-1 {lastResult.correct ? 'text-emerald-500' : 'text-red-500'}">
                        {#if lastResult.correct}<CheckCircle class="size-4"/><TrendingUp class="size-4"/> +{lastResult.delta.toFixed(2)}
                        {:else}<XCircle class="size-4"/><TrendingDown class="size-4"/> {lastResult.delta.toFixed(2)}{/if}
                    </span>
                {/if}
            </div>
        </div>

        <!-- Question -->
        {#if currentItem}
            <div class="rounded-xl border border-border bg-card p-6">
                {#if currentItem.item_type === "mc"}
                    <p class="text-sm font-medium text-foreground mb-4">{currentItem.data?.question ?? ""}</p>
                    <div class="flex flex-col gap-2">
                        {#each currentItem.data?.options ?? [] as opt, oi}
                            <label class="flex items-center gap-2.5 rounded-lg border px-3.5 py-2.5 cursor-pointer transition-colors {answer === oi ? 'border-primary bg-primary/5' : 'border-border hover:border-muted-foreground/30'}">
                                <input type="radio" name="mc" checked={answer === oi} onchange={() => answer = oi} class="sr-only" />
                                <div class="size-4 rounded-full border-2 flex items-center justify-center shrink-0 {answer === oi ? 'border-primary' : 'border-muted-foreground/30'}">
                                    {#if answer === oi}<div class="size-2 rounded-full bg-primary"></div>{/if}
                                </div>
                                <span class="text-sm text-foreground">{opt}</span>
                            </label>
                        {/each}
                    </div>
                {:else if currentItem.item_type === "tf"}
                    <p class="text-sm font-medium text-foreground mb-4">{currentItem.data?.statement ?? ""}</p>
                    <div class="flex gap-3">
                        {#each [true, false] as val}
                            <button class="flex-1 rounded-lg border px-4 py-2.5 text-sm font-medium transition-colors {answer === val ? 'border-primary bg-primary/5 text-primary' : 'border-border text-muted-foreground hover:border-muted-foreground/30'}" onclick={() => answer = val}>{val ? "True" : "False"}</button>
                        {/each}
                    </div>
                {:else if currentItem.item_type === "ma"}
                    <p class="text-sm font-medium text-foreground mb-4">{currentItem.data?.question ?? ""} <span class="text-xs text-muted-foreground">(select all that apply)</span></p>
                    <div class="flex flex-col gap-2">
                        {#each currentItem.data?.options ?? [] as opt, oi}
                            <label class="flex items-center gap-2.5 rounded-lg border px-3.5 py-2.5 cursor-pointer transition-colors {(answer ?? []).includes(oi) ? 'border-primary bg-primary/5' : 'border-border hover:border-muted-foreground/30'}">
                                <input type="checkbox" checked={(answer ?? []).includes(oi)} onchange={(e) => {
                                    const cur = answer ?? [];
                                    const t = e.target as HTMLInputElement;
                                    answer = t.checked ? [...cur, oi] : cur.filter((v: number) => v !== oi);
                                }} class="sr-only" />
                                <div class="size-4 rounded border-2 flex items-center justify-center shrink-0 {(answer ?? []).includes(oi) ? 'bg-primary border-primary text-primary-foreground' : 'border-muted-foreground/30'}">
                                    {#if (answer ?? []).includes(oi)}<CheckCircle class="size-3" />{/if}
                                </div>
                                <span class="text-sm text-foreground">{opt}</span>
                            </label>
                        {/each}
                    </div>
                {:else if currentItem.item_type === "fb"}
                    <p class="text-sm font-medium text-foreground mb-4">{currentItem.data?.text ?? "Fill in the blanks:"}</p>
                    <div class="flex flex-col gap-2">
                        {#each currentItem.data?.blanks ?? [] as blank, bi}
                            <input type="text" placeholder="Answer {bi + 1}" value={answer?.[bi] ?? ""} oninput={(e) => {
                                const cur = [...(answer ?? [])];
                                cur[bi] = (e.target as HTMLInputElement).value;
                                answer = cur;
                            }} class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring" />
                        {/each}
                    </div>
                {:else if currentItem.item_type === "sa"}
                    <p class="text-sm font-medium text-foreground mb-4">{currentItem.data?.question ?? ""}</p>
                    <textarea rows={3} value={answer ?? ""} oninput={(e) => answer = (e.target as HTMLTextAreaElement).value} placeholder="Type your answer..." class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring resize-none"></textarea>
                {:else}
                    <p class="text-sm text-muted-foreground">{currentItem.item_type} — adaptively selected</p>
                    <textarea rows={3} value={answer ?? ""} oninput={(e) => answer = (e.target as HTMLTextAreaElement).value} placeholder="Your response..." class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm mt-2 focus:outline-none focus:ring-2 focus:ring-ring resize-none"></textarea>
                {/if}

                <div class="flex justify-end mt-6">
                    <Button.Root size="default" disabled={submitting || answer === undefined} onclick={submitAnswer}>
                        {#if submitting}<LoaderCircle class="size-4 mr-1.5 animate-spin" /> Submitting...{:else}Submit <ArrowRight class="size-4 ml-1.5" />{/if}
                    </Button.Root>
                </div>
            </div>
        {/if}

        {#if lastResult}
            <div class="rounded-xl border p-4 {lastResult.correct ? 'border-emerald-500/30 bg-emerald-500/5' : 'border-red-500/30 bg-red-500/5'}">
                <div class="flex items-center gap-2 text-sm font-medium {lastResult.correct ? 'text-emerald-500' : 'text-red-500'}">
                    {#if lastResult.correct}<CheckCircle class="size-4"/> Correct!{:else}<XCircle class="size-4"/> Incorrect{/if}
                </div>
                <p class="text-xs text-muted-foreground mt-1">
                    P(correct) was {lastResult.prob} · Theta {'→'} {theta} ({(lastResult.delta >= 0 ? '+' : '')}{lastResult.delta.toFixed(2)})
                </p>
            </div>
        {/if}
    {/if}
</div>
