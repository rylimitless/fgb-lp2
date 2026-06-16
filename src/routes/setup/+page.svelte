<script lang="ts">
    import "../layout.css";
    import { Mail, Lock, Eye, EyeOff, User, ShieldCheck, ArrowRight } from "@lucide/svelte";
    import * as Button from "$lib/components/ui/button";
    import { goto } from "$app/navigation";
    import BrandLogo from "$lib/components/brand/BrandLogo.svelte";
    import AnimatedGradient from "$lib/components/brand/AnimatedGradient.svelte";
    import Spotlight from "$lib/components/brand/Spotlight.svelte";
    import LoadingDots from "$lib/components/brand/LoadingDots.svelte";

    let name = $state("");
    let email = $state("");
    let password = $state("");
    let showPassword = $state(false);
    let loading = $state(false);
    let error = $state("");

    async function handleSubmit(e: SubmitEvent) {
        e.preventDefault();
        loading = true;
        error = "";

        if (password.length < 8) {
            error = "Password must be at least 8 characters";
            loading = false;
            return;
        }

        try {
            const res = await fetch("/api/setup", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                credentials: "include",
                body: JSON.stringify({ name, email, password }),
            });

            if (!res.ok) {
                const data = await res.json();
                error = data.error || "Setup failed";
                loading = false;
                return;
            }

            await goto("/");
        } catch {
            error = "Network error — is the server running?";
        }

        loading = false;
    }

    const setupPillars = [
        {
            label: "You're the first administrator",
            desc: "This account becomes the root admin — full control of users, roles, audits, and content approval.",
        },
        {
            label: "Brand-locked from the start",
            desc: "The Academy is pre-configured with the FGB design system. No theming required.",
        },
        {
            label: "Audit-ready",
            desc: "Every action you and other users take from here is logged. The audit log is your evidence locker.",
        },
    ];
</script>

<svelte:head>
    <title>First-time setup · FGB Academy</title>
</svelte:head>

<div class="grid min-h-svh grid-cols-1 lg:grid-cols-2 bg-background">
    <!-- Brand panel — mirrors /login for parity -->
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
            <BrandLogo variant="lockup" size={36} class="text-primary-foreground" />
        </div>

        <div class="relative z-10 flex max-w-md flex-col gap-6 motion-rise-in">
            <p class="text-[11px] font-semibold uppercase tracking-[0.22em] text-accent">
                Initial setup
            </p>
            <h2 class="text-display-lg font-semibold leading-tight tracking-tight">
                Set up the Academy. <em class="not-italic text-accent">Once.</em>
            </h2>
            <p class="text-sm leading-relaxed text-primary-foreground/80">
                You only see this screen once — on first launch. The account you
                create becomes the root administrator for this instance of FGB Academy.
            </p>

            <ul class="flex flex-col gap-4 mt-2">
                {#each setupPillars as p, i}
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
                            <p class="text-sm font-semibold text-primary-foreground">
                                {p.label}
                            </p>
                            <p class="text-xs text-primary-foreground/70 leading-relaxed mt-0.5">
                                {p.desc}
                            </p>
                        </div>
                    </li>
                {/each}
            </ul>
        </div>

        <div class="relative z-10 text-[11px] text-primary-foreground/60 tracking-wide">
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

            <div class="mb-8 flex flex-col items-center text-center lg:items-start lg:text-left">
                <span class="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">
                    First-time setup
                </span>
                <h1 class="mt-1 text-2xl font-semibold tracking-tight text-foreground">
                    Create admin account
                </h1>
                <p class="mt-2 text-sm text-muted-foreground">
                    This account becomes the root administrator.
                </p>
            </div>

            <form onsubmit={handleSubmit} class="space-y-5">
                <!-- Name -->
                <div class="space-y-2">
                    <label
                        for="name"
                        class="text-sm font-medium leading-none text-foreground"
                    >
                        Full name
                    </label>
                    <div class="relative">
                        <User class="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                        <input
                            id="name"
                            type="text"
                            bind:value={name}
                            placeholder="Jane Smith"
                            required
                            autocomplete="name"
                            class="placeholder:text-muted-foreground/60 flex h-10 w-full rounded-md border border-input bg-background pl-9 pr-3 text-sm text-foreground shadow-xs outline-none transition-all focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/30"
                        />
                    </div>
                </div>

                <!-- Email -->
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
                            placeholder="admin@fgb.com"
                            required
                            autocomplete="email"
                            class="placeholder:text-muted-foreground/60 flex h-10 w-full rounded-md border border-input bg-background pl-9 pr-3 text-sm text-foreground shadow-xs outline-none transition-all focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/30"
                        />
                    </div>
                </div>

                <!-- Password -->
                <div class="space-y-2">
                    <label for="password" class="text-sm font-medium leading-none text-foreground">
                        Password
                    </label>
                    <div class="relative">
                        <Lock class="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                        <input
                            id="password"
                            type={showPassword ? "text" : "password"}
                            bind:value={password}
                            placeholder="Min. 8 characters"
                            required
                            minlength="8"
                            autocomplete="new-password"
                            class="placeholder:text-muted-foreground/60 flex h-10 w-full rounded-md border border-input bg-background pl-9 pr-9 text-sm text-foreground shadow-xs outline-none transition-all focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/30"
                        />
                        <button
                            type="button"
                            onclick={() => (showPassword = !showPassword)}
                            class="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
                            aria-label={showPassword ? "Hide password" : "Show password"}
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

                <Button.Root type="submit" size="lg" class="w-full" disabled={loading}>
                    {#if loading}
                        <span class="mr-1.5 inline-flex">
                            <LoadingDots label="Creating admin account" />
                        </span>
                        Creating admin…
                    {:else}
                        <ShieldCheck class="size-4" />
                        Create admin account
                    {/if}
                </Button.Root>
            </form>

            <p class="mt-8 text-[11px] text-center lg:text-left text-muted-foreground tracking-wide">
                You can change this later from User Management.
            </p>
        </div>
    </main>
</div>
