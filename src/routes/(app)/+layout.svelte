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
        Wand2,
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
        Award,
        ChevronDown,
        UserRound,
    } from "@lucide/svelte";
    import * as Button from "$lib/components/ui/button";
    import * as DropdownMenu from "$lib/components/ui/dropdown-menu/index.js";
    import { onMount } from "svelte";
    import BrandLogo from "$lib/components/brand/BrandLogo.svelte";
    import ThemeToggle from "$lib/components/brand/ThemeToggle.svelte";
    import CommandPalette from "$lib/components/brand/CommandPalette.svelte";
    import MobileDock from "$lib/components/brand/MobileDock.svelte";
    import ToastViewport from "$lib/components/brand/ToastViewport.svelte";

    let { children, data } = $props();

    let loggingOut = $state(false);
    let mobileNavOpen = $state(false);
    let commandOpen = $state(false);
    // Live header-search state. Typing here defers opening the palette until
    // the first keystroke, at which point the palette takes over with the
    // seeded query. Save the round-trip of "click button, then type again".
    let headerQuery = $state("");

    let userRoles: string[] = $derived(data?.user?.roles ?? []);
    function hasRole(...roles: string[]): boolean {
        if (userRoles.includes("admin")) return true;
        return roles.some((r) => userRoles.includes(r));
    }

    // Sidebar nav — grouped into semantic clusters (Learn / Discover / Create /
    // Operate) so admins with the widest role set never see a flat wall of 15
    // buttons. Groups collapse silently when every child is filtered out by the
    // user's roles. See MCP_REVIEW.md item H1.
    type NavItem = {
        label: string;
        icon: any;
        href: string;
        roles: string[];
    };
    type NavGroup = { label: string; items: NavItem[] };

    const navGroups: NavGroup[] = [
        {
            label: "Learn",
            items: [
                { label: "Dashboard", icon: LayoutDashboard, href: "/", roles: [] },
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
                    label: "Learning Paths",
                    icon: Layers,
                    href: "/learning-paths",
                    roles: [],
                },
                {
                    label: "Certificates",
                    icon: Award,
                    href: "/certificates",
                    roles: [],
                },
            ],
        },
        {
            label: "Discover",
            items: [
                {
                    label: "Catalog",
                    icon: Library,
                    href: "/content-repository",
                    roles: [],
                },
            ],
        },
        {
            label: "Create",
            items: [
                {
                    label: "Content Studio",
                    icon: PenTool,
                    href: "/content-studio",
                    roles: ["content creator"],
                },
                {
                    label: "Course Builder",
                    icon: Wand2,
                    href: "/course-builder",
                    roles: ["content creator"],
                },
                {
                    label: "Path Mgmt",
                    icon: Layers,
                    href: "/learning-path-management",
                    roles: ["admin", "manager", "content creator"],
                },
                {
                    label: "Review Queue",
                    icon: ClipboardCheck,
                    href: "/review-queue",
                    roles: ["approver"],
                },
            ],
        },
        {
            label: "Operate",
            items: [
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
                    roles: ["admin", "auditor"],
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
            ],
        },
    ];

    let visibleGroups = $derived(
        navGroups
            .map((g) => ({
                label: g.label,
                items: g.items.filter(
                    (i) => i.roles.length === 0 || hasRole(...i.roles),
                ),
            }))
            .filter((g) => g.items.length > 0),
    );

    // Flat list retained for the command palette + mobile fallback.
    let flatNav = $derived(visibleGroups.flatMap((g) => g.items));
    let commandItems = $derived([
        ...flatNav.map((item) => ({
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
    async function markAllRead() {
        try {
            const res = await fetch(`/api/notifications/read-all`, {
                method: "PUT",
                credentials: "include",
            });
            if (res.ok) {
                unreadCount = 0;
                notifications = notifications.map((n: any) => ({
                    ...n,
                    is_read: true,
                }));
            }
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
    let userInitials = $derived(
        ((data?.user?.name ?? "FGB User") as string)
            .split(/\s+/)
            .slice(0, 2)
            .map((part) => part[0])
            .join("")
            .toUpperCase(),
    );
</script>

<svelte:head><link rel="icon" href={favicon} /></svelte:head>

<div class="relative min-h-svh bg-background">
    <!-- ===== FLOATING SIDEBAR ===== -->
    <aside
        class="fixed inset-y-0 left-0 z-40 w-[236px] transition-transform duration-300 lg:w-[clamp(148px,14.5vw,236px)] lg:translate-x-0 {mobileNavOpen
            ? 'translate-x-0'
            : '-translate-x-full'}"
    >
        <div
            class="sidebar-shell relative flex h-full flex-col overflow-hidden border-r border-accent/25 brand-gradient text-primary-foreground shadow-lg"
        >
            <div
                class="pointer-events-none absolute -top-16 -left-10 size-48 rounded-full bg-accent/20 blur-3xl"
            ></div>

            <!-- brand -->
            <div
                class="relative z-10 flex min-h-12 items-center justify-between px-4 py-2 xl:min-h-16 xl:px-5 xl:py-3"
            >
                <a href="/" class="flex items-center" aria-label="FGB Academy">
                    <BrandLogo
                        variant="lockup"
                        size={24}
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
                class="sidebar-nav relative z-10 flex flex-1 flex-col gap-1 overflow-y-auto px-2 py-1 xl:px-3 xl:py-2"
                aria-label="Primary"
            >
                {#each visibleGroups as group, gi (group.label)}
                    <div class="flex flex-col gap-0 {gi > 0 ? 'mt-1.5 xl:mt-3' : ''}">
                        <span
                            class="px-2 py-0.5 text-[7px] font-semibold uppercase tracking-[0.18em] text-primary-foreground/45 xl:px-3 xl:py-1 xl:text-[10px]"
                            id={"nav-group-" + group.label.toLowerCase()}
                        >
                            {group.label}
                        </span>
                        <ul
                            class="flex flex-col gap-0.5"
                            role="list"
                            aria-labelledby={"nav-group-" +
                                group.label.toLowerCase()}
                        >
                            {#each group.items as item (item.href)}
                                {@const active = isActive(item.href)}
                                <li>
                                    <button
                                        onclick={() => navTo(item.href)}
                                        aria-current={active
                                            ? "page"
                                            : undefined}
                                        class="focus-premium group relative flex min-h-5 w-full items-center gap-2 rounded-md px-2 py-0.5 text-[8px] font-medium transition-all press xl:min-h-9 xl:gap-3 xl:rounded-lg xl:px-3 xl:py-2 xl:text-sm {active
                                            ? 'bg-accent text-accent-foreground shadow-glow'
                                            : 'text-primary-foreground/75 hover:bg-primary-foreground/10 hover:text-primary-foreground'}"
                                    >
                                        {#if active}
                                            <span
                                                class="absolute left-0 top-1/2 h-5 w-1 -translate-y-1/2 rounded-r-full bg-accent-foreground/40"
                                            ></span>
                                        {/if}
                                        <item.icon class="size-3 shrink-0 xl:size-4" />
                                        <span class="truncate">
                                            {item.label}
                                        </span>
                                    </button>
                                </li>
                            {/each}
                        </ul>
                    </div>
                {/each}
            </nav>

            <!-- footer -->
            <div
                class="relative z-10 flex flex-col gap-0.5 border-t border-primary-foreground/10 px-2 py-2 xl:px-3 xl:py-3"
            >
                <button
                    class="focus-premium flex min-h-5 items-center gap-2 rounded-md px-2 py-0.5 text-[8px] font-medium text-primary-foreground/70 transition-colors hover:bg-primary-foreground/10 hover:text-primary-foreground xl:min-h-9 xl:gap-3 xl:rounded-lg xl:px-3 xl:py-2 xl:text-sm"
                    onclick={() => goto("/help")}
                >
                    <LifeBuoy class="size-4" /> Help &amp; support
                </button>
                <button
                    class="focus-premium flex min-h-5 items-center gap-2 rounded-md px-2 py-0.5 text-[8px] font-medium text-primary-foreground/70 transition-colors hover:bg-primary-foreground/10 hover:text-primary-foreground xl:min-h-9 xl:gap-3 xl:rounded-lg xl:px-3 xl:py-2 xl:text-sm"
                    onclick={() => goto("/settings")}
                >
                    <Settings class="size-4" /> Settings
                </button>
                <button
                    class="focus-premium flex min-h-5 items-center gap-2 rounded-md px-2 py-0.5 text-[8px] font-medium text-primary-foreground/70 transition-colors hover:bg-primary-foreground/10 hover:text-primary-foreground disabled:opacity-50 xl:min-h-9 xl:gap-3 xl:rounded-lg xl:px-3 xl:py-2 xl:text-sm"
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
    <div class="flex min-h-svh flex-col lg:pl-[clamp(148px,14.5vw,236px)]">
        <!-- top bar -->
        <header
            class="sticky top-0 z-20 flex h-12 items-center gap-2 border-b border-border bg-background/88 px-3 backdrop-blur-md supports-[backdrop-filter]:bg-background/76 md:px-4 xl:h-14 xl:gap-3"
        >
            <button
                class="lg:hidden text-muted-foreground hover:text-foreground"
                onclick={() => (mobileNavOpen = true)}
                aria-label="Open menu"
            >
                <Menu class="size-5" />
            </button>

            <!-- search — live input. Typing straight in opens the palette
                 with the query pre-seeded; clicking opens with empty state.
                 Ctrl/Cmd K still works from anywhere via the global listener. -->
            <div class="relative max-w-[420px] flex-1">
                <Search
                    class="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
                />
                <input
                    type="search"
                    inputmode="search"
                    autocomplete="off"
                    spellcheck="false"
                    value={headerQuery}
                    placeholder="Search courses, documents, people…"
                    aria-label="Search FGB Academy"
                    class="h-8 w-full rounded-lg border border-border bg-surface-1 pl-9 pr-14 text-xs text-foreground shadow-inner placeholder:text-muted-foreground outline-none transition-colors hover:border-border-strong focus-visible:border-ring focus-visible:ring-2 focus-visible:ring-ring/30 xl:h-9 xl:text-sm"
                    onfocus={() => (commandOpen = true)}
                    onkeydown={(e) => {
                        if (e.key === "Enter" || /^[a-zA-Z0-9]$/.test(e.key)) {
                            if (e.key.length === 1) headerQuery = e.key;
                            commandOpen = true;
                        }
                    }}
                    oninput={(e) => {
                        headerQuery = (e.currentTarget as HTMLInputElement).value;
                        commandOpen = true;
                    }}
                />
                <kbd
                    class="pointer-events-none absolute right-3 top-1/2 hidden -translate-y-1/2 rounded border border-border-strong bg-muted px-1.5 py-0.5 text-[10px] text-muted-foreground sm:block"
                >
                    Ctrl K
                </kbd>
            </div>

            <div class="ml-auto flex items-center gap-1.5 xl:gap-2">
                <!-- XP (compact on mobile, full pill on ≥sm) -->
                <button
                    type="button"
                    onclick={() => goto("/settings")}
                    aria-label={`${xpPoints.toLocaleString()} XP earned`}
                    class="focus-premium flex h-8 items-center gap-1.5 rounded-lg border border-success/25 bg-success/8 px-2 transition-colors hover:border-success/50 xl:px-2.5"
                >
                    <Trophy class="size-3.5 text-success" />
                    <span
                        class="text-xs font-bold text-success tabular"
                        >{xpPoints.toLocaleString()}</span
                    >
                    <span
                        class="hidden sm:inline text-[10px] uppercase tracking-wider text-success/70"
                        >XP</span
                    >
                </button>
                <!-- streak (compact on mobile, full pill on ≥sm) -->
                <button
                    type="button"
                    onclick={() => goto("/")}
                    aria-label={`${streakDays} day learning streak`}
                    class="focus-premium flex h-8 items-center gap-1.5 rounded-lg border border-streak/25 bg-streak/8 px-2 transition-colors hover:border-streak/50 xl:px-2.5"
                >
                    <Flame class="size-3.5 text-streak fill-streak/40" />
                    <span class="text-xs font-bold text-streak tabular"
                        >{streakDays}</span
                    >
                    <span
                        class="hidden sm:inline text-[10px] uppercase tracking-wider text-streak/80"
                        >day</span
                    >
                </button>

                <ThemeToggle />

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
                                class="relative size-9 rounded-lg"
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
                        <div
                            class="flex items-center justify-between gap-2 px-3 py-2 border-b border-border-strong"
                        >
                            <span class="text-sm font-semibold text-foreground"
                                >Notifications</span
                            >
                            <div class="flex items-center gap-1">
                                {#if unreadCount > 0}
                                    <button
                                        type="button"
                                        class="text-[11px] font-medium text-primary hover:underline"
                                        onclick={markAllRead}
                                    >
                                        Mark all read
                                    </button>
                                {/if}
                                <button
                                    type="button"
                                    class="text-[11px] font-medium text-muted-foreground hover:text-foreground hover:underline"
                                    onclick={() => {
                                        notifOpen = false;
                                        goto("/notifications");
                                    }}
                                >
                                    View all
                                </button>
                            </div>
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
                <div class="hidden sm:block">
                  <DropdownMenu.Root>
                    <DropdownMenu.Trigger>
                        {#snippet child({ props })}
                            <button
                                {...props}
                                class="focus-premium flex min-h-9 items-center gap-2 rounded-lg px-1.5 text-left transition-colors hover:bg-muted"
                                aria-label="Open account menu"
                            >
                                <span class="flex size-8 items-center justify-center rounded-full border border-accent/45 bg-primary text-[10px] font-bold text-primary-foreground">
                                    {userInitials}
                                </span>
                                {#if data?.user?.name}
                                    <span class="hidden max-w-28 flex-col leading-tight md:flex">
                                        <span class="truncate text-[11px] font-semibold text-foreground">{data.user.name}</span>
                                        <span class="truncate text-[9px] uppercase tracking-wider text-muted-foreground">{data.user.role}</span>
                                    </span>
                                {/if}
                                <ChevronDown class="hidden size-3 text-muted-foreground md:block" />
                            </button>
                        {/snippet}
                    </DropdownMenu.Trigger>
                    <DropdownMenu.Content class="w-52" align="end">
                        <DropdownMenu.Label>{data?.user?.name ?? "Account"}</DropdownMenu.Label>
                        <DropdownMenu.Separator />
                        <DropdownMenu.Item onclick={() => goto("/settings")}>
                            <UserRound class="size-4" /> Profile &amp; settings
                        </DropdownMenu.Item>
                        <DropdownMenu.Item onclick={handleLogout} disabled={loggingOut}>
                            <LogOut class="size-4" /> {loggingOut ? "Signing out…" : "Sign out"}
                        </DropdownMenu.Item>
                    </DropdownMenu.Content>
                  </DropdownMenu.Root>
                </div>
            </div>
        </header>

        <!-- content -->
        <main class="flex flex-1 p-3 pb-24 md:p-4 lg:pb-4 xl:p-5">
            {@render children()}
        </main>
    </div>
</div>

<MobileDock />

<CommandPalette
    open={commandOpen}
    items={commandItems}
    initialQuery={headerQuery}
    onOpenChange={(open) => {
        commandOpen = open;
        if (!open) headerQuery = "";
    }}
/>

<ToastViewport />

<style>
    /* Sidebar scrollbar — kept flush inside the dark navy card so the gutter
       never bleeds out into the pale surface-2 background. Thin, brand-tinted,
       auto-appears only when the nav overflows the viewport. */
    :global(.sidebar-nav) {
        scrollbar-width: thin;
        scrollbar-color: rgba(255, 255, 255, 0.18) transparent;
    }
    :global(.sidebar-nav::-webkit-scrollbar) {
        width: 6px;
    }
    :global(.sidebar-nav::-webkit-scrollbar-track) {
        background: transparent;
    }
    :global(.sidebar-nav::-webkit-scrollbar-thumb) {
        background-color: rgba(255, 255, 255, 0.18);
        border-radius: 9999px;
    }
    :global(.sidebar-nav:hover::-webkit-scrollbar-thumb) {
        background-color: rgba(255, 255, 255, 0.28);
    }
    :global(.sidebar-shell) {
        --primary-foreground: oklch(0.94 0.014 225);
    }
</style>
