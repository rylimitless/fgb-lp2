<script lang="ts">
    import {
        Bot,
        User,
        Send,
        LoaderCircle,
        BookOpen,
        Sparkles,
        FileText,
    } from "@lucide/svelte";
    import * as Button from "$lib/components/ui/button";

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

    $effect(() => {
        loadDocs();
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

<div class="flex w-full max-w-5xl mx-auto gap-6 h-[calc(100vh-8rem)]">
    <!-- Sidebar: Approved Documents -->
    <aside class="w-[240px] shrink-0 flex flex-col gap-2">
        <div class="flex items-center gap-2 mb-1">
            <BookOpen class="size-4 text-primary" />
            <span class="text-sm font-semibold text-foreground">
                Approved Documents</span
            >
        </div>
        {#if approvedDocs.length === 0}
            <p class="text-xs text-muted-foreground">
                No approved documents yet.
            </p>
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
            <div
                class="size-10 rounded-full bg-primary/10 flex items-center justify-center"
            >
                <Bot class="size-5 text-primary" />
            </div>
            <div>
                <h1 class="text-xl font-semibold text-foreground">
                    Gia — AI Learning Coach
                </h1>
                <p class="text-xs text-muted-foreground">
                    Ask questions about your approved documents
                </p>
            </div>
        </div>

        <div
            class="flex-1 overflow-y-auto rounded-xl border border-border bg-card p-4 mb-4"
            bind:this={chatContainer}
        >
            {#if messages.length === 0}
                <div
                    class="flex flex-col items-center justify-center h-full text-center gap-3 py-12"
                >
                    <Sparkles class="size-10 text-muted-foreground/30" />
                    <p class="text-sm text-muted-foreground">
                        Ask me anything about your approved documents.
                    </p>
                    <p class="text-xs text-muted-foreground/60">
                        Or click a document in the sidebar →
                    </p>
                </div>
            {:else}
                <div class="flex flex-col gap-4">
                    {#each messages as msg}
                        <div
                            class="flex gap-3 {msg.role === 'user'
                                ? 'flex-row-reverse'
                                : ''}"
                        >
                            <div
                                class="size-8 rounded-full flex items-center justify-center shrink-0 {msg.role ===
                                'user'
                                    ? 'bg-primary/10'
                                    : 'bg-emerald-500/10'}"
                            >
                                {#if msg.role === "user"}<User
                                        class="size-4 text-primary"
                                    />{:else}<Bot
                                        class="size-4 text-emerald-500"
                                    />{/if}
                            </div>
                            <div class="max-w-[80%]">
                                <div
                                    class="rounded-xl px-4 py-2.5 text-sm leading-relaxed {msg.role ===
                                    'user'
                                        ? 'bg-primary text-primary-foreground'
                                        : 'bg-muted text-foreground'} whitespace-pre-line"
                                >
                                    {msg.content}
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
                        <div class="flex gap-3">
                            <div
                                class="size-8 rounded-full bg-emerald-500/10 flex items-center justify-center shrink-0"
                            >
                                <Bot class="size-4 text-emerald-500" />
                            </div>
                            <div
                                class="rounded-xl px-4 py-2.5 bg-muted flex items-center gap-2"
                            >
                                <LoaderCircle
                                    class="size-3.5 text-muted-foreground animate-spin"
                                />
                                <span class="text-sm text-muted-foreground"
                                    >Thinking...</span
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
            >
                <Send class="size-4" />
            </Button.Root>
        </div>
    </div>
</div>
