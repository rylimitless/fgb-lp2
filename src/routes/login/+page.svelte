<script lang="ts">
    import "../layout.css";
    import { Mail, Lock, LogIn, Eye, EyeOff, ArrowRight } from "@lucide/svelte";
    import * as Button from "$lib/components/ui/button";
    import { goto } from "$app/navigation";
    import BrandLogo from "$lib/components/brand/BrandLogo.svelte";
    import AnimatedGradient from "$lib/components/brand/AnimatedGradient.svelte";
    import Spotlight from "$lib/components/brand/Spotlight.svelte";

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

    const pillars = [
        {
            label: "Personalised by adaptive AI",
            desc: "Every lesson is tuned to where you are — not where the average learner is.",
        },
        {
            label: "Grounded in your documents",
            desc: "Gia answers from sources your team approves. Nothing fabricated, nothing borrowed.",
        },
        {
            label: "Built for the bank",
            desc: "Roles, audits, and approvals match the way FGB already works.",
        },
    ];
</script>

<svelte:head>
    <title>Sign in · FGB Academy</title>
</svelte:head>

<div class="grid min-h-svh grid-cols-1 lg:grid-cols-2 bg-background">
    <!-- Brand panel — visible on lg+ -->
    <aside
        class="relative hidden lg:flex flex-col justify-between overflow-hidden p-10 xl:p-14 text-primary-foreground brand-gradient"
    >
        <!-- Higgsfield cinematic hero (desaturated into the gradient) -->
        <img
            src="/brand/hero/hero-login.png"
            alt=""
            aria-hidden="true"
            class="absolute inset-0 size-full object-cover opacity-40 mix-blend-luminosity"
            fetchpriority="high"
        />
        <AnimatedGradient variant="primary" />
        <Spotlight class="h-full w-full" opacity={0.55} />

        <div class="relative z-10 flex items-center gap-3">
            <BrandLogo
                variant="lockup"
                size={36}
                class="text-primary-foreground"
            />
        </div>

        <div class="relative z-10 flex max-w-md flex-col gap-6 motion-rise-in">
            <p
                class="text-[11px] font-semibold uppercase tracking-[0.22em] text-accent"
            >
                Welcome to the Academy
            </p>
            <h2
                class="text-display-lg font-semibold leading-tight tracking-tight"
            >
                Learning the bank can <em class="not-italic text-accent"
                    >stake its reputation on.</em
                >
            </h2>
            <p class="text-sm leading-relaxed text-primary-foreground/80">
                FGB Academy is the institutional learning ecosystem for the FGB
                community — premium, intelligent, and grounded in the documents
                your team already trusts.
            </p>

            <ul class="flex flex-col gap-4 mt-2">
                {#each pillars as p, i}
                    <li
                        class="flex items-start gap-3 motion-rise-in"
                        style="animation-delay: {120 + i * 60}ms"
                    >
                        <span
                            class="mt-1 flex size-5 items-center justify-center rounded-full bg-accent text-accent-foreground shrink-0"
                        >
                            <ArrowRight class="size-3" />
                        </span>
                        <div>
                            <p
                                class="text-sm font-semibold text-primary-foreground"
                            >
                                {p.label}
                            </p>
                            <p
                                class="text-xs text-primary-foreground/70 leading-relaxed mt-0.5"
                            >
                                {p.desc}
                            </p>
                        </div>
                    </li>
                {/each}
            </ul>
        </div>

        <div
            class="relative z-10 text-[11px] text-primary-foreground/60 tracking-wide"
        >
            FGB Academy · 25 years of First Global Bank
        </div>
    </aside>

    <!-- Form panel -->
    <main class="relative flex items-center justify-center p-6 sm:p-10">
        <div class="w-full max-w-sm motion-rise-in">
            <!-- Mobile brand mark -->
            <div class="mb-8 flex items-center justify-center lg:hidden">
                <BrandLogo variant="lockup" size={32} class="text-primary" />
            </div>

            <div
                class="mb-8 flex flex-col items-center text-center lg:items-start lg:text-left"
            >
                <span
                    class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground"
                >
                    Sign in
                </span>
                <h1
                    class="mt-1 text-2xl font-semibold tracking-tight text-foreground"
                >
                    Welcome back
                </h1>
                <p class="mt-2 text-sm text-muted-foreground">
                    Enter your credentials to continue learning.
                </p>
            </div>

            <form onsubmit={handleSubmit} class="space-y-5">
                <!-- Email -->
                <div class="space-y-2">
                    <label
                        for="email"
                        class="text-sm font-medium leading-none text-foreground"
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
                            placeholder="you@fgb.com"
                            required
                            autocomplete="email"
                            class="placeholder:text-muted-foreground/60 flex h-10 w-full rounded-md border border-input bg-background pl-9 pr-3 text-sm text-foreground shadow-xs outline-none transition-all focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/30 disabled:cursor-not-allowed disabled:opacity-50"
                        />
                    </div>
                </div>

                <!-- Password -->
                <div class="space-y-2">
                    <div class="flex items-center justify-between">
                        <label
                            for="password"
                            class="text-sm font-medium leading-none text-foreground"
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
                            class="placeholder:text-muted-foreground/60 flex h-10 w-full rounded-md border border-input bg-background pl-9 pr-9 text-sm text-foreground shadow-xs outline-none transition-all focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/30 disabled:cursor-not-allowed disabled:opacity-50"
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
                    <p
                        role="alert"
                        class="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive"
                    >
                        {error}
                    </p>
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
                            class="mr-2 inline-block size-3.5 animate-spin rounded-full border-2 border-current border-r-transparent"
                        ></span>
                    {:else}
                        <LogIn class="size-4" />
                    {/if}
                    {loading ? "Signing in…" : "Sign in"}
                </Button.Root>
            </form>

            <p
                class="mt-8 text-[11px] text-center lg:text-left text-muted-foreground tracking-wide"
            >
                Protected by FGB enterprise authentication.
            </p>
        </div>
    </main>
</div>
