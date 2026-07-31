<script lang="ts">
    import {
        User,
        Send,
        BookOpen,
        FileText,
    } from "@lucide/svelte";
    import * as Button from "$lib/components/ui/button";
    import { onMount } from "svelte";
    import { page } from "$app/state";
    import {
        GiaAvatar,
        GiaTip,
        LoadingDots,
        Spotlight,
        Markdown,
        OrbitingInsights,
    } from "$lib/components/brand";

    type Message = {
        role: "user" | "assistant";
        content: string;
        sources?: { document_id: number; document_title: string }[];
    };

    let messages = $state<Message[]>([]);
    let input = $state("");
    let loading = $state(false);
    let chatContainer: HTMLDivElement;

    let approvedDocs = $state<any[]>([]);

    onMount(() => {
        loadDocs();
        const query = page.url.searchParams.get("q")?.trim();
        if (query) {
            input = query;
            send();
        }
    });

    async function loadDocs() {
        try {
            const res = await fetch("/api/documents", {
                credentials: "include",
            });
            if (res.ok) {
                const docs = await res.json();
                approvedDocs = docs.filter((d: any) => d.approved);
            }
        } catch {
            /* ignore */
        }
    }

    function askAbout(doc: any) {
        input = `Tell me about the key points in "${doc.title}"`;
        send();
    }

    // Fallback prompt chips used when the learner opens Gia Coach with no
    // prior conversation. They give newcomers a starting shape for questions;
    // one-tap and Gia responds. Copy is sentence-case per gia.md §6.
    const STARTER_PROMPTS = [
        "Summarise the most recent approved document.",
        "Quiz me on my current course.",
        "Explain a tough concept from my learning path.",
        "What should I focus on this week?",
    ];

    function askStarter(prompt: string) {
        input = prompt;
        send();
    }

    function scrollDown() {
        requestAnimationFrame(() => {
            if (chatContainer)
                chatContainer.scrollTop = chatContainer.scrollHeight;
        });
    }

    async function send() {
        const text = input.trim();
        if (!text || loading) return;
        messages = [...messages, { role: "user", content: text }];
        input = "";
        scrollDown();
        loading = true;

        try {
            const res = await fetch("/api/coach/chat", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                credentials: "include",
                body: JSON.stringify({ message: text }),
            });
            if (!res.ok) throw new Error("Failed");
            const data = await res.json();
            messages = [
                ...messages,
                {
                    role: "assistant",
                    content: data.answer,
                    sources: data.sources,
                },
            ];
        } catch {
            messages = [
                ...messages,
                {
                    role: "assistant",
                    content:
                        "Sorry, I couldn't process that. Is the backend running?",
                },
            ];
        } finally {
            loading = false;
            scrollDown();
        }
    }

    function handleKeydown(e: KeyboardEvent) {
        if (e.key === "Enter" && !e.shiftKey) {
            e.preventDefault();
            send();
        }
    }
</script>

<div class="flex flex-col md:flex-row w-full max-w-5xl mx-auto gap-4 md:gap-6 md:h-[calc(100vh-8rem)]">
    <!-- Sidebar: Approved Documents -->
    <aside class="md:w-[240px] shrink-0 flex flex-col gap-2 max-h-48 md:max-h-none overflow-hidden">
        <div class="flex items-center gap-2 mb-1">
            <BookOpen class="size-4 text-primary" />
            <span class="text-sm font-semibold text-foreground">
                Approved Documents</span
            >
        </div>
        {#if approvedDocs.length === 0}
            <GiaTip
                size="sm"
                message="No approved documents yet. Once they're reviewed, I'll answer from them."
                class="self-start"
            />
        {:else}
            <div class="flex flex-col gap-1 overflow-y-auto flex-1">
                {#each approvedDocs as doc}
                    <button
                        class="flex items-center gap-2 rounded-lg px-2.5 py-2 text-left text-xs transition-colors text-muted-foreground hover:bg-muted hover:text-foreground"
                        onclick={() => askAbout(doc)}
                    >
                        <FileText class="size-3.5 shrink-0" />
                        <span class="truncate">{doc.title}</span>
                    </button>
                {/each}
            </div>
        {/if}
    </aside>

    <!-- Chat -->
    <div class="flex-1 flex flex-col min-w-0">
        <div class="flex items-center gap-3 mb-4 shrink-0">
            <GiaAvatar size={44} state={loading ? "thinking" : "idle"} pulse={loading} />
            <div>
                <h1 class="text-xl font-semibold text-foreground">
                    Gia — Growth Intelligence Assistant
                </h1>
                <p class="text-xs text-muted-foreground">
                    Ask Gia about your approved documents. She cites every source.
                </p>
            </div>
        </div>

        <div
            class="relative flex-1 overflow-y-auto rounded-2xl border border-border bg-card p-4 mb-4"
            bind:this={chatContainer}
            aria-live="polite"
            aria-busy={loading}
            aria-label="Conversation with Gia"
        >
            <Spotlight class="h-full w-full" opacity={0.12} />
            {#if messages.length === 0}
                <div
                    class="relative z-10 flex flex-col items-center justify-center h-full text-center gap-4 py-12"
                >
                    <OrbitingInsights size={220} />
                    <div class="flex flex-col gap-1.5 max-w-sm">
                        <p class="text-base font-semibold text-foreground">
                            What would you like to learn about?
                        </p>
                        <p class="text-sm text-muted-foreground leading-relaxed">
                            Ask anything about your approved documents — Gia will
                            answer with citations.
                        </p>
                    </div>

                    <!-- Starter prompt chips — give newcomers a first push -->
                    <div
                        class="flex flex-wrap justify-center gap-2 max-w-lg mt-2"
                    >
                        {#each STARTER_PROMPTS as prompt}
                            <button
                                type="button"
                                onclick={() => askStarter(prompt)}
                                class="rounded-full border border-border-strong bg-card px-3 py-1.5 text-xs text-foreground hover:border-primary hover:bg-primary-soft transition-colors press"
                            >
                                {prompt}
                            </button>
                        {/each}
                    </div>

                    {#if approvedDocs.length > 0}
                        <p class="text-[11px] text-muted-foreground/70 uppercase tracking-wider mt-2">
                            Or pick a document from the sidebar
                        </p>
                    {/if}
                </div>
            {:else}
                <div class="relative z-10 flex flex-col gap-4">
                    {#each messages as msg, i}
                        <div
                            class="flex gap-3 motion-rise-in {msg.role === 'user'
                                ? 'flex-row-reverse'
                                : ''}"
                            style="animation-delay: {Math.min(i * 30, 240)}ms"
                        >
                            {#if msg.role === "user"}
                                <div
                                    class="size-8 rounded-full bg-primary/10 flex items-center justify-center shrink-0"
                                    aria-hidden="true"
                                >
                                    <User class="size-4 text-primary" />
                                </div>
                            {:else}
                                <GiaAvatar size={32} state="idle" />
                            {/if}
                            <div class="max-w-[80%]">
                                <div
                                    class="rounded-2xl px-4 py-2.5 text-sm leading-relaxed shadow-sm {msg.role ===
                                    'user'
                                        ? 'bg-primary text-primary-foreground whitespace-pre-line'
                                        : 'bg-muted text-foreground border border-border/60'}"
                                >
                                    {#if msg.role === "assistant"}
                                        <Markdown source={msg.content} />
                                    {:else}
                                        {msg.content}
                                    {/if}
                                </div>
                                {#if msg.sources && msg.sources.length > 0}
                                    <div
                                        class="flex items-center gap-1.5 mt-1.5 flex-wrap"
                                    >
                                        <BookOpen
                                            class="size-3 text-muted-foreground"
                                        />
                                        {#each msg.sources as src}
                                            <span
                                                class="text-[10px] text-muted-foreground bg-muted/50 rounded px-1.5 py-0.5"
                                                >{src.document_title}</span
                                            >
                                        {/each}
                                    </div>
                                {/if}
                            </div>
                        </div>
                    {/each}

                    {#if loading}
                        <div class="flex gap-3 motion-rise-in">
                            <GiaAvatar size={32} state="thinking" pulse />
                            <div
                                class="rounded-2xl px-4 py-3 bg-muted border border-border/60 flex items-center gap-2.5 text-info"
                            >
                                <LoadingDots label="Gia is thinking" />
                                <span class="text-sm text-muted-foreground"
                                    >Reading your sources…</span
                                >
                            </div>
                        </div>
                    {/if}
                </div>
            {/if}
        </div>

        <div class="flex gap-2 shrink-0">
            <textarea
                bind:value={input}
                aria-label="Message Gia"
                rows={2}
                placeholder="Ask Gia about your documents..."
                onkeydown={handleKeydown}
                disabled={loading}
                class="flex-1 rounded-xl border border-input bg-background px-4 py-2.5 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring resize-none"
            ></textarea>
            <Button.Root
                size="icon"
                class="size-10 shrink-0 self-end"
                disabled={loading || !input.trim()}
                onclick={send}
                aria-label="Send message to Gia"
            >
                <Send class="size-4" />
            </Button.Root>
        </div>
    </div>
</div>
