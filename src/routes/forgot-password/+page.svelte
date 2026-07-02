<script lang="ts">
    import "../layout.css";
    import { Mail, ArrowLeft, Send } from "@lucide/svelte";
    import * as Button from "$lib/components/ui/button";
    import { goto } from "$app/navigation";
    import BrandLogo from "$lib/components/brand/BrandLogo.svelte";

    let email = $state("");
    let loading = $state(false);
    let sent = $state(false);
    let error = $state("");

    async function handleSubmit(e: SubmitEvent) {
        e.preventDefault();
        loading = true;
        error = "";

        try {
            const res = await fetch("/api/forgot-password", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ email }),
            });

            const data = await res.json();

            if (!res.ok) {
                error = data.error || "Something went wrong.";
                loading = false;
                return;
            }

            sent = true;
        } catch {
            error = "Network error — is the server running?";
        }

        loading = false;
    }
</script>

<svelte:head>
    <title>Forgot password · FGB Academy</title>
</svelte:head>

<div class="flex min-h-svh items-center justify-center bg-background p-6">
    <div class="w-full max-w-sm motion-rise-in">
        <div class="mb-8 flex items-center justify-center">
            <BrandLogo variant="lockup" size={32} class="text-primary" />
        </div>

        <div class="mb-8 flex flex-col items-center text-center">
            <span class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">
                Password reset
            </span>
            <h1 class="mt-1 text-2xl font-semibold tracking-tight text-foreground">
                Forgot your password?
            </h1>
            <p class="mt-2 text-sm text-muted-foreground">
                {sent
                    ? "Check your email for a reset link."
                    : "Enter your email and we'll send you a reset link."}
            </p>
        </div>

        {#if sent}
            <div class="rounded-lg border border-success/30 bg-success/10 px-4 py-3 text-sm text-success text-center">
                If an account with that email exists, a reset link has been sent. It expires in 1 hour.
            </div>
            <Button.Root
                variant="outline"
                size="lg"
                class="w-full mt-6"
                onclick={() => goto("/login")}
            >
                <ArrowLeft class="size-4" />
                Back to sign in
            </Button.Root>
        {:else}
            <form onsubmit={handleSubmit} class="space-y-5">
                <div class="space-y-2">
                    <label for="email" class="text-sm font-medium leading-none text-foreground">
                        Email
                    </label>
                    <div class="relative">
                        <Mail class="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                        <input
                            id="email"
                            type="email"
                            bind:value={email}
                            placeholder="you@fgb.com"
                            required
                            autocomplete="email"
                            class="placeholder:text-muted-foreground/60 flex h-10 w-full rounded-md border border-input bg-background pl-9 pr-3 text-sm text-foreground shadow-xs outline-none transition-all focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/30"
                        />
                    </div>
                </div>

                {#if error}
                    <p role="alert" class="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
                        {error}
                    </p>
                {/if}

                <Button.Root type="submit" size="lg" class="w-full" disabled={loading}>
                    {#if loading}
                        <span class="mr-2 inline-block size-3.5 animate-spin rounded-full border-2 border-current border-r-transparent"></span>
                    {:else}
                        <Send class="size-4" />
                    {/if}
                    {loading ? "Sending…" : "Send reset link"}
                </Button.Root>
            </form>

            <Button.Root
                variant="ghost"
                size="sm"
                class="w-full mt-4"
                onclick={() => goto("/login")}
            >
                <ArrowLeft class="size-4" />
                Back to sign in
            </Button.Root>
        {/if}
    </div>
</div>
