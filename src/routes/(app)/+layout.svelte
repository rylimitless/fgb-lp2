<script lang="ts">
    import "../layout.css";
    import favicon from "$lib/assets/favicon.svg";
    import { page } from "$app/stores";
    import { goto } from "$app/navigation";
    import {
        LogOut,
        LayoutDashboard,
        Bot,
        Brain,
        GraduationCap,
        PenTool,
        Sparkles,
        ClipboardCheck,
        Library,
        Users,
        Bell,
        LoaderCircle,
        BarChart3,
        ClipboardList,
        Search,
        Flame,
        Trophy,
        Settings,
        LifeBuoy,
        Menu,
        X,
        Building2,
        Layers,
    } from "@lucide/svelte";
    import * as Button from "$lib/components/ui/button";
    import * as DropdownMenu from "$lib/components/ui/dropdown-menu/index.js";
    import { onMount } from "svelte";
    import BrandLogo from "$lib/components/brand/BrandLogo.svelte";
    import GiaAvatar from "$lib/components/brand/GiaAvatar.svelte";
    import CommandPalette from "$lib/components/brand/CommandPalette.svelte";
    import ToastViewport from "$lib/components/brand/ToastViewport.svelte";

    let { children, data } = $props();

    let loggingOut = $state(false);
    let mobileNavOpen = $state(false);
    let commandOpen = $state(false);

    let userRoles: string[] = $derived(data?.user?.roles ?? []);
    function hasRole(...roles: string[]): boolean {
        if (userRoles.includes("admin")) return true;
        return roles.some((r) => userRoles.includes(r));
    }

    // Sidebar nav — real routes + role gating, styled to the reference.
    const allNav = [
        {
            label: "Dashboard",
            icon: LayoutDashboard,
            href: "/",
            roles: [] as string[],
        },
        {
            label: "My Learning",
            icon: GraduationCap,
            href: "/lesson-player",
            roles: ["end user", "content creator", "approver"],
        },
        {
            label: "Adaptive Room",
            icon: Brain,
            href: "/adaptive-room",
            roles: ["end user", "content creator", "approver"],
        },
        {
            label: "Gia Coach",
            icon: Bot,
            href: "/gia-coach",
            roles: ["end user", "content creator", "approver"],
        },
        {
            label: "Catalog",
            icon: Library,
            href: "/content-repository",
            roles: [] as string[],
        },
        {
            label: "Content Studio",
            icon: PenTool,
            href: "/content-studio",
            roles: ["content creator"],
        },
        {
            label: "AI Generator",
            icon: Sparkles,
            href: "/ai-content-generator",
            roles: ["content creator"],
        },
        {
            label: "Review Queue",
            icon: ClipboardCheck,
            href: "/review-queue",
            roles: ["approver"],
        },
        {
            label: "Reports",
            icon: BarChart3,
            href: "/analytics",
            roles: ["admin", "manager", "auditor"],
        },
        {
            label: "Team",
            icon: Users,
            href: "/user-management",
            roles: ["admin"],
        },
        {
            label: "Learning Paths",
            icon: Layers,
            href: "/learning-paths",
            roles: [] as string[],
        },
        {
            label: "Path Mgmt",
            icon: Layers,
            href: "/learning-path-management",
            roles: ["admin", "manager", "content creator"],
        },
        {
            label: "Departments",
            icon: Building2,
            href: "/department-management",
            roles: ["admin", "manager"],
        },
        {
            label: "Audit Log",
            icon: ClipboardList,
            href: "/audit-log",
            roles: ["admin", "auditor"],
        },
    ];
    let nav = $derived(
        allNav.filter((t) => t.roles.length === 0 || hasRole(...t.roles)),
    );
    let commandItems = $derived([
        ...nav.map((item) => ({
            label: item.label,
            href: item.href,
            description: "Navigate",
            icon: item.icon,
            kind: "nav" as const,
        })),
        {
            label: "Settings",
            href: "/settings",
            description: "Profile and preferences",
            icon: Settings,
            kind: "nav" as const,
        },
    ]);

    function isActive(href: string): boolean {
        const path = $page.url.pathname;
        return href === "/" ? path === "/" : path.startsWith(href);
    }

    // ---- Notifications (unchanged logic) ----
    let unreadCount = $state(0);
    let notifications = $state<any[]>([]);
    let notifOpen = $state(false);
    let notifLoading = $state(false);
    let pollInterval: ReturnType<typeof setInterval>;

    async function fetchUnreadCount() {
        try {
            const res = await fetch("/api/notifications/unread-count", {
                credentials: "include",
            });
            if (res.ok) {
                const d = await res.json();
                unreadCount = d.unread ?? 0;
            }
        } catch {
            /* ignore */
        }
    }
    async function fetchNotifications() {
        notifLoading = true;
        try {
            const res = await fetch("/api/notifications?limit=20", {
                credentials: "include",
            });
            if (res.ok) {
                const d = await res.json();
                notifications = d.items ?? [];
                unreadCount = d.unread ?? 0;
            }
        } catch {
            /* ignore */
        }
        notifLoading = false;
    }
    async function markRead(id: number) {
        try {
            await fetch(`/api/notifications/${id}/read`, {
                method: "PUT",
                credentials: "include",
            });
            unreadCount = Math.max(0, unreadCount - 1);
            notifications = notifications.map((n: any) =>
                n.id === id ? { ...n, is_read: true } : n,
            );
        } catch {
            /* ignore */
        }
    }
    function handleNotifClick(n: any) {
        if (!n.is_read) markRead(n.id);
        if (n.link) goto(n.link);
        notifOpen = false;
    }
    function formatNotifDate(d: string) {
        if (!d) return "";
        const dt = new Date(d);
        const now = new Date();
        const m = Math.floor((now.getTime() - dt.getTime()) / 60000);
        if (m < 1) return "Just now";
        if (m < 60) return `${m}m ago`;
        const h = Math.floor(m / 60);
        if (h < 24) return `${h}h ago`;
        return dt.toLocaleDateString("en-US", {
            month: "short",
            day: "numeric",
        });
    }

    onMount(() => {
        fetchUnreadCount();
        pollInterval = setInterval(fetchUnreadCount, 30000);
        const onKey = (event: KeyboardEvent) => {
            if (
                (event.ctrlKey || event.metaKey) &&
                event.key.toLowerCase() === "k"
            ) {
                event.preventDefault();
                commandOpen = true;
            }
        };
        window.addEventListener("keydown", onKey);
        return () => {
            clearInterval(pollInterval);
            window.removeEventListener("keydown", onKey);
        };
    });

    async function handleLogout() {
        loggingOut = true;
        try {
            await fetch("/api/logout", {
                method: "POST",
                credentials: "include",
            });
        } catch {
            /* ignore */
        }
        await goto("/login");
    }

    function navTo(href: string) {
        mobileNavOpen = false;
        goto(href);
    }

    // Gamification values from server
    let xpPoints = $derived((data as any)?.gamification?.totalScore ?? 0);
    let streakDays = $derived((data as any)?.gamification?.streakDays ?? 0);
</script>

<svelte:head><link rel="icon" href={favicon} /></svelte:head>

<div class="relative min-h-svh bg-surface-2">
    <!-- ===== FLOATING SIDEBAR ===== -->
    <aside
        class="fixed inset-y-0 left-0 z-40 w-64 p-3 transition-transform duration-300 lg:translate-x-0 {mobileNavOpen
            ? 'translate-x-0'
            : '-translate-x-full'}"
    >
        <div
            class="relative flex h-full flex-col overflow-hidden rounded-3xl brand-gradient text-primary-foreground shadow-lg"
        >
            <div
                class="pointer-events-none absolute -top-16 -left-10 size-48 rounded-full bg-accent/20 blur-3xl"
            ></div>

            <!-- brand -->
            <div
                class="relative z-10 flex items-center justify-between px-5 pt-5 pb-3"
            >
                <a href="/" class="flex items-center" aria-label="FGB Academy">
                    <BrandLogo
                        variant="lockup"
                        size={26}
                        class="text-primary-foreground"
                    />
                </a>
                <button
                    class="lg:hidden text-primary-foreground/70 hover:text-primary-foreground"
                    onclick={() => (mobileNavOpen = false)}
                    aria-label="Close menu"
                >
                    <X class="size-5" />
                </button>
            </div>

            <!-- nav -->
            <nav
                class="relative z-10 flex-1 overflow-y-auto px-3 py-2 flex flex-col gap-0.5"
            >
                {#each nav as item}
                    {@const active = isActive(item.href)}
                    <button
                        onclick={() => navTo(item.href)}
                        class="group relative flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium transition-all press {active
                            ? 'bg-accent text-accent-foreground shadow-glow'
                            : 'text-primary-foreground/70 hover:bg-primary-foreground/10 hover:text-primary-foreground'}"
                    >
                        {#if active}
                            <span
                                class="absolute left-0 top-1/2 h-5 w-1 -translate-y-1/2 rounded-r-full bg-accent-foreground/40"
                            ></span>
                        {/if}
                        <item.icon class="size-4 shrink-0" />
                        <span class="truncate">{item.label}</span>
                    </button>
                {/each}
            </nav>

            <!-- footer -->
            <div
                class="relative z-10 border-t border-primary-foreground/10 px-3 py-3 flex flex-col gap-0.5"
            >
                <button
                    class="flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium text-primary-foreground/70 hover:bg-primary-foreground/10 hover:text-primary-foreground transition-colors press"
                    onclick={() => goto("/content-repository")}
                >
                    <LifeBuoy class="size-4" /> Help Center
                </button>
                <button
                    class="flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium text-primary-foreground/70 hover:bg-primary-foreground/10 hover:text-primary-foreground transition-colors press"
                    onclick={() => goto("/settings")}
                >
                    <Settings class="size-4" /> Settings
                </button>
                <button
                    class="flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium text-primary-foreground/70 hover:bg-primary-foreground/10 hover:text-primary-foreground transition-colors press"
                    onclick={handleLogout}
                    disabled={loggingOut}
                >
                    <LogOut class="size-4" />
                    {loggingOut ? "Signing out…" : "Sign out"}
                </button>
            </div>
        </div>
    </aside>

    <!-- mobile scrim -->
    {#if mobileNavOpen}
        <button
            class="fixed inset-0 z-30 bg-black/40 lg:hidden"
            onclick={() => (mobileNavOpen = false)}
            aria-label="Close menu"
        ></button>
    {/if}

    <!-- ===== MAIN COLUMN ===== -->
    <div class="lg:pl-64 flex min-h-svh flex-col">
        <!-- top bar -->
        <header
            class="sticky top-0 z-20 flex h-16 items-center gap-3 px-4 md:px-6 bg-surface-2/80 backdrop-blur supports-[backdrop-filter]:bg-surface-2/60"
        >
            <button
                class="lg:hidden text-muted-foreground hover:text-foreground"
                onclick={() => (mobileNavOpen = true)}
                aria-label="Open menu"
            >
                <Menu class="size-5" />
            </button>

            <!-- search -->
            <div class="relative flex-1 max-w-md">
                <Search
                    class="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
                />
                <button
                    type="button"
                    class="h-9 w-full rounded-full border border-border bg-card pl-9 pr-14 text-left text-sm text-muted-foreground outline-none transition-colors hover:text-foreground focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/30"
                    aria-label="Open search"
                    onclick={() => (commandOpen = true)}
                >
                    Search courses, skills, topics…
                </button>
                <kbd
                    class="pointer-events-none absolute right-3 top-1/2 hidden -translate-y-1/2 rounded border border-border bg-muted px-1.5 py-0.5 text-[10px] text-muted-foreground sm:block"
                >
                    Ctrl K
                </kbd>
            </div>

            <div class="ml-auto flex items-center gap-2 md:gap-3">
                <!-- XP -->
                <div
                    class="hidden sm:flex items-center gap-1.5 rounded-full border border-accent/30 bg-accent-soft px-3 py-1.5"
                >
                    <Trophy class="size-3.5 text-accent-foreground" />
                    <span
                        class="text-xs font-bold text-accent-foreground tabular"
                        >{xpPoints.toLocaleString()}</span
                    >
                    <span
                        class="text-[10px] uppercase tracking-wider text-accent-foreground/70"
                        >XP</span
                    >
                </div>
                <!-- streak -->
                <div
                    class="hidden sm:flex items-center gap-1.5 rounded-full border border-streak/30 bg-streak/10 px-3 py-1.5"
                >
                    <Flame class="size-3.5 text-streak fill-streak/40" />
                    <span class="text-xs font-bold text-streak tabular"
                        >{streakDays}</span
                    >
                    <span
                        class="text-[10px] uppercase tracking-wider text-streak/80"
                        >day</span
                    >
                </div>

                <!-- notifications -->
                <DropdownMenu.Root
                    open={notifOpen}
                    onOpenChange={(o) => {
                        notifOpen = o;
                        if (o) fetchNotifications();
                    }}
                >
                    <DropdownMenu.Trigger>
                        {#snippet child({ props })}
                            <Button.Root
                                {...props}
                                variant="ghost"
                                size="icon"
                                class="relative size-9 rounded-full"
                            >
                                <Bell class="size-4" />
                                {#if unreadCount > 0}
                                    <span
                                        class="absolute -top-0.5 -right-0.5 flex items-center justify-center min-w-[18px] h-[18px] rounded-full bg-destructive text-destructive-foreground text-[10px] font-bold px-1"
                                    >
                                        {unreadCount > 99 ? "99+" : unreadCount}
                                    </span>
                                {/if}
                            </Button.Root>
                        {/snippet}
                    </DropdownMenu.Trigger>
                    <DropdownMenu.Content class="w-80" align="end">
                        <div class="px-3 py-2 border-b border-border">
                            <span class="text-sm font-semibold text-foreground"
                                >Notifications</span
                            >
                        </div>
                        <div class="max-h-72 overflow-y-auto">
                            {#if notifLoading}
                                <div
                                    class="flex items-center justify-center py-8"
                                >
                                    <LoaderCircle
                                        class="size-5 text-muted-foreground animate-spin"
                                    />
                                </div>
                            {:else if notifications.length === 0}
                                <div class="px-4 py-8 text-center">
                                    <p class="text-sm text-muted-foreground">
                                        No notifications
                                    </p>
                                </div>
                            {:else}
                                {#each notifications as n}
                                    <button
                                        class="w-full text-left px-4 py-3 hover:bg-muted/50 transition-colors border-b border-border last:border-0 {!n.is_read
                                            ? 'bg-muted/20'
                                            : ''}"
                                        onclick={() => handleNotifClick(n)}
                                    >
                                        <div class="flex items-start gap-2">
                                            {#if !n.is_read}<span
                                                    class="mt-1.5 size-2 rounded-full bg-info shrink-0"
                                                ></span>{:else}<span
                                                    class="mt-1.5 size-2 shrink-0"
                                                ></span>{/if}
                                            <div class="min-w-0 flex-1">
                                                <p
                                                    class="text-sm font-medium text-foreground truncate"
                                                >
                                                    {n.title}
                                                </p>
                                                <p
                                                    class="text-xs text-muted-foreground mt-0.5 line-clamp-2"
                                                >
                                                    {n.message}
                                                </p>
                                                <p
                                                    class="text-[10px] text-muted-foreground/60 mt-1"
                                                >
                                                    {formatNotifDate(
                                                        n.created_at,
                                                    )}
                                                </p>
                                            </div>
                                        </div>
                                    </button>
                                {/each}
                            {/if}
                        </div>
                    </DropdownMenu.Content>
                </DropdownMenu.Root>

                <!-- profile -->
                <div class="flex items-center gap-2 pl-1">
                    <GiaAvatar size={34} />
                    {#if data?.user?.name}
                        <div class="hidden md:flex flex-col leading-tight">
                            <span class="text-xs font-semibold text-foreground"
                                >{data.user.name}</span
                            >
                            <span
                                class="text-[10px] uppercase tracking-wider text-muted-foreground"
                                >{data.user.role}</span
                            >
                        </div>
                    {/if}
                </div>
            </div>
        </header>

        <!-- content -->
        <main class="flex flex-1 p-4 md:p-6">
            {@render children()}
        </main>
    </div>
</div>

<CommandPalette
    open={commandOpen}
    items={commandItems}
    onOpenChange={(open) => (commandOpen = open)}
/>

<ToastViewport />
