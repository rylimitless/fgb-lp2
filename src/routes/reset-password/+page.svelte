<script lang="ts">
    import "../layout.css";
    import { Lock, ArrowLeft, CheckCircle } from "@lucide/svelte";
    import * as Button from "$lib/components/ui/button";
    import { goto } from "$app/navigation";
    import { page } from "$app/stores";
    import BrandLogo from "$lib/components/brand/BrandLogo.svelte";

    let token = $state("");
    let password = $state("");
    let confirmPassword = $state("");
    let loading = $state(false);
    let success = $state(false);
    let error = $state("");

    $effect(() => {
        const params = new URLSearchParams($page.url.search);
        token = params.get("token") ?? "";
    });

    async function handleSubmit(e: SubmitEvent) {
        e.preventDefault();
        error = "";

        if (password !== confirmPassword) {
            error = "Passwords do not match.";
            return;
        }

        if (password.length < 8) {
            error = "Password must be at least 8 characters.";
            return;
        }

        if (!token) {
            error = "Missing reset token. Please use the link from your email.";
            return;
        }

        loading = true;

        try {
            const res = await fetch("/api/reset-password", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ token, password }),
            });

            const data = await res.json();

            if (!res.ok) {
                error = data.error || "Failed to reset password.";
                loading = false;
                return;
            }

            success = true;
        } catch {
            error = "Network error — is the server running?";
        }

        loading = false;
    }
</script>

<svelte:head>
    <title>Reset password · FGB Academy</title>
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
                {success ? "Password updated" : "Set new password"}
            </h1>
            <p class="mt-2 text-sm text-muted-foreground">
                {success
                    ? "Your password has been reset successfully."
                    : "Choose a new password for your account."}
            </p>
        </div>

        {#if success}
            <div class="rounded-lg border border-success/30 bg-success/10 px-4 py-3 text-sm text-success text-center flex items-center justify-center gap-2">
                <CheckCircle class="size-4" />
                Password reset complete. You can now sign in.
            </div>
            <Button.Root
                size="lg"
                class="w-full mt-6"
                onclick={() => goto("/login")}
            >
                Sign in
            </Button.Root>
        {:else if !token}
            <div class="rounded-lg border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive text-center">
                Invalid or missing reset token. Please request a new reset link.
            </div>
            <Button.Root
                variant="outline"
                size="lg"
                class="w-full mt-6"
                onclick={() => goto("/forgot-password")}
            >
                Request new link
            </Button.Root>
        {:else}
            <form onsubmit={handleSubmit} class="space-y-5">
                <div class="space-y-2">
                    <label for="password" class="text-sm font-medium leading-none text-foreground">
                        New password
                    </label>
                    <div class="relative">
                        <Lock class="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                        <input
                            id="password"
                            type="password"
                            bind:value={password}
                            placeholder="At least 8 characters"
                            required
                            minlength="8"
                            autocomplete="new-password"
                            class="placeholder:text-muted-foreground/60 flex h-10 w-full rounded-md border border-input bg-background pl-9 pr-3 text-sm text-foreground shadow-xs outline-none transition-all focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/30"
                        />
                    </div>
                </div>

                <div class="space-y-2">
                    <label for="confirm" class="text-sm font-medium leading-none text-foreground">
                        Confirm password
                    </label>
                    <div class="relative">
                        <Lock class="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                        <input
                            id="confirm"
                            type="password"
                            bind:value={confirmPassword}
                            placeholder="Re-enter your password"
                            required
                            minlength="8"
                            autocomplete="new-password"
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
                    {/if}
                    {loading ? "Resetting…" : "Reset password"}
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
