<script lang="ts">
    import "../layout.css";
    import { Mail, Lock, LogIn, Eye, EyeOff } from "@lucide/svelte";
    import * as Button from "$lib/components/ui/button";
    import { goto } from "$app/navigation";

    let email = $state("");
    let password = $state("");
    let showPassword = $state(false);
    let loading = $state(false);
    let error = $state("");

    async function handleSubmit(e: SubmitEvent) {
        e.preventDefault();
        loading = true;
        error = "";

        try {
            const res = await fetch("/api/login", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                credentials: "include",
                body: JSON.stringify({ email, password }),
            });

            if (!res.ok) {
                const data = await res.json();
                error = data.error || "Login failed";
                loading = false;
                return;
            }

            await goto("/");
        } catch {
            error = "Network error — is the server running?";
        }

        loading = false;
    }
</script>

<div class="flex min-h-svh items-center justify-center bg-background p-4">
    <div
        class="w-full max-w-sm animate-in fade-in slide-in-from-bottom-6 duration-500"
    >
        <!-- Header -->
        <div class="mb-8 text-center">
            <h1 class="text-2xl font-semibold tracking-tight text-foreground">
                Welcome back
            </h1>
            <p class="mt-2 text-sm text-muted-foreground">
                Enter your credentials to access your account
            </p>
        </div>

        <!-- Form -->
        <form onsubmit={handleSubmit} class="space-y-5">
            <!-- Email -->
            <div class="space-y-2">
                <label
                    for="email"
                    class="text-sm font-medium leading-none text-foreground peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
                >
                    Email
                </label>
                <div class="relative">
                    <Mail
                        class="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
                    />
                    <input
                        id="email"
                        type="email"
                        bind:value={email}
                        placeholder="you@example.com"
                        required
                        autocomplete="email"
                        class="placeholder:text-muted-foreground/60 flex h-9 w-full rounded-md border border-input bg-background pl-9 pr-3 text-sm text-foreground shadow-xs outline-none transition-colors file:border-0 file:bg-transparent file:text-sm file:font-medium focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/30 disabled:cursor-not-allowed disabled:opacity-50"
                    />
                </div>
            </div>

            <!-- Password -->
            <div class="space-y-2">
                <div class="flex items-center justify-between">
                    <label
                        for="password"
                        class="text-sm font-medium leading-none text-foreground peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
                    >
                        Password
                    </label>
                    <a
                        href="/forgot-password"
                        class="text-xs font-medium text-muted-foreground underline-offset-4 hover:text-foreground hover:underline transition-colors"
                    >
                        Forgot?
                    </a>
                </div>
                <div class="relative">
                    <Lock
                        class="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
                    />
                    <input
                        id="password"
                        type={showPassword ? "text" : "password"}
                        bind:value={password}
                        placeholder="••••••••"
                        required
                        autocomplete="current-password"
                        class="placeholder:text-muted-foreground/60 flex h-9 w-full rounded-md border border-input bg-background pl-9 pr-9 text-sm text-foreground shadow-xs outline-none transition-colors file:border-0 file:bg-transparent file:text-sm file:font-medium focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/30 disabled:cursor-not-allowed disabled:opacity-50"
                    />
                    <button
                        type="button"
                        onclick={() => (showPassword = !showPassword)}
                        class="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
                        aria-label={showPassword
                            ? "Hide password"
                            : "Show password"}
                    >
                        {#if showPassword}
                            <EyeOff class="size-4" />
                        {:else}
                            <Eye class="size-4" />
                        {/if}
                    </button>
                </div>
            </div>

            {#if error}
                <p class="text-sm text-destructive">{error}</p>
            {/if}

            <!-- Submit -->
            <Button.Root
                type="submit"
                size="lg"
                class="w-full"
                disabled={loading}
            >
                {#if loading}
                    <span
                        class="mr-2 i-lucide-loader-circle inline-block size-3.5 animate-spin rounded-full border-2 border-current border-r-transparent"
                    ></span>
                {:else}
                    <LogIn class="size-4" />
                {/if}
                {loading ? "Signing in…" : "Sign in"}
            </Button.Root>
        </form>
    </div>
</div>
