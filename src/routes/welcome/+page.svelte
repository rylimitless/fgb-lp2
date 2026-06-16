<script lang="ts">
    import "../layout.css";
    import { goto } from "$app/navigation";
    import {
        ArrowRight, Play, Sparkles, ShieldCheck, Lightbulb, TrendingUp,
        BrainCircuit, Layers, Globe, BarChart3,
    } from "@lucide/svelte";
    import * as Button from "$lib/components/ui/button";
    import {
        BrandLogo, Particles, AnimatedGrid, Spotlight, ScrollReveal,
    } from "$lib/components/brand";

    const features = [
        { icon: Sparkles, title: "Personalised learning", desc: "AI recommends exactly what you need, when you need it." },
        { icon: ShieldCheck, title: "Trusted by FGB", desc: "Built on verified content, approvals, and real expertise." },
        { icon: Lightbulb, title: "Learning that sticks", desc: "Interactive, engaging, and built for real-world impact." },
        { icon: TrendingUp, title: "Grow your career", desc: "Track skills, earn badges, and showcase your progress." },
    ];

    const learners = [
        { title: "Employees", blurb: "Build skills, stay compliant and grow your career.", image: "/brand/pathways/banking-foundations.png" },
        { title: "Leaders", blurb: "Strengthen leadership capabilities and drive performance.", image: "/brand/pathways/leadership.png" },
        { title: "Teams", blurb: "Upskill together and achieve more as one.", image: "/brand/pathways/customer-service.png" },
        { title: "Partners", blurb: "Access resources designed for our valued partners.", image: "/brand/programs/executive.png" },
    ];

    const pillars = [
        { icon: BrainCircuit, label: "Adaptive AI", sub: "Recommendations" },
        { icon: Layers, label: "Microlearning", sub: "Bite-sized impact" },
        { icon: Globe, label: "Real-world", sub: "Application" },
        { icon: BarChart3, label: "Measurable", sub: "Progress" },
    ];
</script>

<svelte:head>
    <title>FGB Academy — Learn today. Lead tomorrow.</title>
    <meta name="description" content="FGB Academy — the intelligent learning ecosystem for the First Global Bank community. 25 years of excellence." />
</svelte:head>

<div class="min-h-svh brand-gradient text-primary-foreground overflow-x-hidden">
    <!-- ===== NAV ===== -->
    <header class="fixed top-0 inset-x-0 z-50 flex items-center justify-between px-5 md:px-12 h-16 backdrop-blur-md bg-[oklch(0.16_0.04_245/0.55)] border-b border-primary-foreground/10">
        <BrandLogo variant="lockup" size={26} class="text-primary-foreground" />
        <nav class="hidden md:flex items-center gap-8 text-sm text-primary-foreground/70">
            <a href="#why" class="hover:text-primary-foreground transition-colors">Why FGB Academy</a>
            <a href="#learners" class="hover:text-primary-foreground transition-colors">Learning Paths</a>
            <a href="#learners" class="hover:text-primary-foreground transition-colors">For Teams</a>
            <a href="#banner" class="hover:text-primary-foreground transition-colors">Resources</a>
        </nav>
        <div class="flex items-center gap-2">
            <Button.Root variant="outline" size="default" class="border-primary-foreground/30 bg-transparent text-primary-foreground hover:bg-primary-foreground/10" onclick={() => goto("/login")}>Sign in</Button.Root>
            <Button.Root size="default" class="bg-accent text-accent-foreground hover:bg-accent/85" onclick={() => goto("/login")}>Get Started</Button.Root>
        </div>
    </header>

    <!-- ===== SECTION 1 — HERO ===== -->
    <section class="relative grid lg:grid-cols-2 items-center gap-8 min-h-svh px-5 md:px-12 pt-24 pb-16 overflow-hidden">
        <AnimatedGrid size={64} />
        <Particles quantity={90} color="#d6c47e" />
        <Spotlight class="h-[120%] w-[80%]" opacity={0.45} />

        <!-- Left: copy -->
        <div class="relative z-10 flex flex-col gap-6 max-w-xl">
            <span class="inline-flex w-fit items-center gap-2 rounded-full border border-accent/40 bg-accent/10 px-4 py-1.5 text-[11px] font-semibold uppercase tracking-[0.2em] text-accent motion-rise-in">
                <span class="flex size-4 items-center justify-center rounded-full bg-accent text-accent-foreground text-[8px] font-bold">25</span>
                25 Years of Excellence
            </span>
            <span class="inline-flex w-fit items-center gap-2 text-xs font-medium text-primary-foreground/70 motion-rise-in">
                <span class="size-2 rounded-full bg-success motion-glow"></span>
                4,200+ FGB learners ready for the next capability sprint
            </span>
            <h1 class="text-display-2xl font-bold tracking-tight leading-[0.95] motion-rise-in motion-stagger-1">
                Learn today.<br />
                <span class="text-aurora">Lead tomorrow.</span>
            </h1>
            <p class="max-w-md text-lg text-primary-foreground/80 leading-relaxed motion-rise-in motion-stagger-2">
                FGB Academy is your intelligent learning companion — designed to
                build capability, confidence and career growth across First
                Global Bank.
            </p>
            <div class="flex flex-wrap items-center gap-3 motion-rise-in motion-stagger-3">
                <Button.Root size="lg" class="group h-12 px-7 text-sm bg-accent text-accent-foreground hover:bg-accent/85 shadow-glow" onclick={() => goto("/login")}>
                    Get Started <ArrowRight class="size-4 transition-transform group-hover:translate-x-0.5" />
                </Button.Root>
                <Button.Root variant="outline" size="lg" class="h-12 px-6 text-sm border-primary-foreground/30 bg-primary-foreground/5 text-primary-foreground hover:bg-primary-foreground/10" onclick={() => goto("/login")}>
                    <Play class="size-4 fill-current" /> Watch overview
                </Button.Root>
            </div>
        </div>

        <!-- Right: cinematic stairway -->
        <div class="relative z-10 flex items-center justify-center motion-rise-in motion-stagger-2">
            <div class="relative">
                <div class="absolute inset-0 -m-10 rounded-full bg-accent/15 blur-3xl"></div>
                <img src="/brand/hero/stairway.png" alt="A learner ascending toward the FGB Academy mark" class="relative w-full max-w-md rounded-3xl shadow-lg motion-float" fetchpriority="high" />
            </div>
        </div>
    </section>

    <!-- ===== SECTION 2 — FEATURE ROW ===== -->
    <section id="why" class="relative mx-auto max-w-6xl px-5 md:px-12 pb-8">
        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
            {#each features as f, i}
                <ScrollReveal delay={i * 70}>
                    <div class="group h-full rounded-2xl border border-primary-foreground/10 bg-primary-foreground/[0.04] backdrop-blur p-5 lift">
                        <span class="flex size-11 items-center justify-center rounded-xl bg-accent/15 text-accent group-hover:bg-accent group-hover:text-accent-foreground transition-colors">
                            <f.icon class="size-5" />
                        </span>
                        <h3 class="mt-4 text-base font-semibold text-primary-foreground">{f.title}</h3>
                        <p class="mt-1.5 text-sm text-primary-foreground/65 leading-relaxed">{f.desc}</p>
                    </div>
                </ScrollReveal>
            {/each}
        </div>
    </section>

    <!-- ===== SECTION 3 — DESIGNED FOR EVERY LEARNER ===== -->
    <section id="learners" class="relative mx-auto max-w-6xl px-5 md:px-12 py-20 md:py-28">
        <ScrollReveal class="text-center mb-12">
            <h2 class="text-display-lg font-bold tracking-tight">
                Designed for <span class="text-aurora">every learner</span>
            </h2>
        </ScrollReveal>
        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
            {#each learners as l, i}
                <ScrollReveal delay={i * 80}>
                    <button class="group relative block h-80 w-full overflow-hidden rounded-3xl border border-primary-foreground/10 text-left press" onclick={() => goto("/login")}>
                        <img src={l.image} alt="" class="absolute inset-0 size-full object-cover transition-transform duration-500 group-hover:scale-105" loading="lazy" onerror={(e) => ((e.currentTarget as HTMLImageElement).style.display='none')} />
                        <div class="absolute inset-0 bg-gradient-to-t from-[oklch(0.14_0.04_245/0.95)] via-[oklch(0.14_0.04_245/0.45)] to-transparent"></div>
                        <div class="absolute inset-x-0 bottom-0 p-5">
                            <h3 class="text-xl font-bold text-primary-foreground">{l.title}</h3>
                            <p class="mt-1 text-xs text-primary-foreground/75 leading-relaxed max-h-0 overflow-hidden opacity-0 transition-all duration-300 group-hover:max-h-20 group-hover:opacity-100">{l.blurb}</p>
                            <span class="mt-3 inline-flex size-8 items-center justify-center rounded-full bg-accent text-accent-foreground translate-y-0 group-hover:translate-x-1 transition-transform">
                                <ArrowRight class="size-4" />
                            </span>
                        </div>
                    </button>
                </ScrollReveal>
            {/each}
        </div>
    </section>

    <!-- ===== SECTION 4 — PREMIUM 25-YEAR BANNER ===== -->
    <section id="banner" class="relative mx-auto max-w-6xl px-5 md:px-12 pb-24">
        <ScrollReveal>
            <div class="relative overflow-hidden rounded-3xl border border-accent/20 bg-[oklch(0.13_0.03_245)] p-8 md:p-12">
                <AnimatedGrid size={44} />
                <Particles quantity={50} color="#d6c47e" />
                <Spotlight class="h-full w-1/2" opacity={0.3} />
                <div class="relative z-10">
                    <div class="text-center mb-10">
                        <h2 class="text-display-md font-bold tracking-tight">A learning experience that feels different</h2>
                    </div>
                    <div class="grid grid-cols-2 md:grid-cols-4 gap-6 mb-12">
                        {#each pillars as p, i}
                            <ScrollReveal delay={i * 70} class="flex flex-col items-center text-center gap-2">
                                <span class="flex size-12 items-center justify-center rounded-2xl bg-accent/15 text-accent"><p.icon class="size-6" /></span>
                                <p class="text-sm font-semibold text-primary-foreground">{p.label}</p>
                                <p class="text-[11px] text-primary-foreground/60">{p.sub}</p>
                            </ScrollReveal>
                        {/each}
                    </div>
                    <div class="flex flex-col md:flex-row items-center justify-center gap-6 pt-6 border-t border-primary-foreground/10">
                        <!-- 25 medallion -->
                        <div class="relative flex size-24 items-center justify-center rounded-full bg-gradient-to-br from-accent to-[oklch(0.62_0.1_86)] text-accent-foreground shadow-glow motion-float shrink-0">
                            <div class="absolute inset-1.5 rounded-full border-2 border-accent-foreground/30"></div>
                            <div class="flex flex-col items-center leading-none">
                                <span class="text-2xl font-bold tabular">25</span>
                                <span class="text-[7px] font-semibold uppercase tracking-[0.18em]">Years</span>
                            </div>
                        </div>
                        <p class="text-center md:text-left text-lg md:text-xl font-semibold leading-snug">
                            Empowering people. <span class="text-aurora">Strengthening the bank.</span><br class="hidden md:block" />
                            Building the future — together.
                        </p>
                    </div>
                    <div class="mt-10 text-center">
                        <Button.Root size="lg" class="h-12 px-8 text-sm bg-accent text-accent-foreground hover:bg-accent/85 shadow-glow" onclick={() => goto("/login")}>
                            Start your journey <ArrowRight class="size-4" />
                        </Button.Root>
                    </div>
                </div>
            </div>
        </ScrollReveal>
    </section>

    <footer class="border-t border-primary-foreground/10 px-5 md:px-12 py-6 text-center text-[11px] text-primary-foreground/50 tracking-wide">
        FGB Academy · 25 years of First Global Bank
    </footer>
</div>
