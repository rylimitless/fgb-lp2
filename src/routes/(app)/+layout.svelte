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
        Ellipsis,
        Bell,
        LoaderCircle,
        BarChart3,
    } from "@lucide/svelte";
    import * as Button from "$lib/components/ui/button";
    import * as Tabs from "$lib/components/ui/tabs";
    import * as DropdownMenu from "$lib/components/ui/dropdown-menu/index.js";
    import { onMount } from "svelte";

    let { children } = $props();

    let loggingOut = $state(false);

    const navTabs = [
        {
            value: "dashboard",
            label: "Dashboard",
            icon: LayoutDashboard,
            href: "/",
        },
        {
            value: "gia-coach",
            label: "Gia Coach",
            icon: Bot,
            href: "/gia-coach",
        },
    ];

    const moreTabs = [
        {
            value: "adaptive-room",
            label: "Adaptive Room",
            icon: Brain,
            href: "/adaptive-room",
        },
        {
            value: "lesson-player",
            label: "Guided Lesson Player",
            icon: GraduationCap,
            href: "/lesson-player",
        },
        {
            value: "content-studio",
            label: "Content Studio",
            icon: PenTool,
            href: "/content-studio",
        },
        {
            value: "ai-content-generator",
            label: "AI Generator",
            icon: Sparkles,
            href: "/ai-content-generator",
        },
        {
            value: "review-queue",
            label: "Review Queue",
            icon: ClipboardCheck,
            href: "/review-queue",
        },
        {
            value: "content-repository",
            label: "Repository",
            icon: Library,
            href: "/content-repository",
        },
        {
            value: "user-management",
            label: "User Management",
            icon: Users,
            href: "/user-management",
        },
        {
            value: "analytics",
            label: "Analytics",
            icon: BarChart3,
            href: "/analytics",
        },
    ];

    let currentTab = $state(
        $page.url.pathname.startsWith("/gia-coach") ? "gia-coach" : "dashboard",
    );

    // Notifications state
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
                const data = await res.json();
                unreadCount = data.unread ?? 0;
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
                const data = await res.json();
                notifications = data.items ?? [];
                unreadCount = data.unread ?? 0;
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
        const diffMs = now.getTime() - dt.getTime();
        const diffMins = Math.floor(diffMs / 60000);
        if (diffMins < 1) return "Just now";
        if (diffMins < 60) return `${diffMins}m ago`;
        const diffHours = Math.floor(diffMins / 60);
        if (diffHours < 24) return `${diffHours}h ago`;
        return dt.toLocaleDateString("en-US", {
            month: "short",
            day: "numeric",
        });
    }

    onMount(() => {
        fetchUnreadCount();
        pollInterval = setInterval(fetchUnreadCount, 30000);
        return () => clearInterval(pollInterval);
    });

    async function handleLogout() {
        loggingOut = true;
        try {
            await fetch("/api/logout", {
                method: "POST",
                credentials: "include",
            });
        } catch {
            // proceed even if server call fails
        }
        await goto("/login");
    }

    function handleTabClick(value: string) {
        const tab = navTabs.find((t) => t.value === value);
        if (tab) goto(tab.href);
    }
</script>

<svelte:head><link rel="icon" href={favicon} /></svelte:head>

<div class="flex min-h-svh flex-col bg-background">
    <!-- Top bar -->
    <header
        class="sticky top-0 z-10 flex h-14 items-center justify-between border-b border-border bg-background px-6"
    >
        <div class="flex items-center gap-4">
            <LayoutDashboard class="size-5 text-primary" />
            <span class="text-sm font-semibold tracking-tight text-foreground"
                >FGB</span
            >
            <Tabs.Root value={currentTab}>
                <Tabs.List variant="line" class="h-full">
                    {#each navTabs as tab}
                        <Tabs.Trigger
                            value={tab.value}
                            onclick={() => handleTabClick(tab.value)}
                            class="h-full data-active:text-foreground"
                        >
                            <tab.icon class="size-3.5" />
                            <span>{tab.label}</span>
                        </Tabs.Trigger>
                    {/each}
                </Tabs.List>
            </Tabs.Root>

            <!-- Notifications bell -->
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
                            size="sm"
                            class="relative"
                        >
                            <Bell class="size-4" />
                            <span>Notifications</span>
                            {#if unreadCount > 0}
                                <span
                                    class="absolute -top-0.5 -right-0.5 flex items-center justify-center min-w-[18px] h-[18px] rounded-full bg-red-500 text-white text-[10px] font-bold px-1"
                                >
                                    {unreadCount > 99 ? "99+" : unreadCount}
                                </span>
                            {/if}
                        </Button.Root>
                    {/snippet}
                </DropdownMenu.Trigger>
                <DropdownMenu.Content class="w-80" align="start">
                    <div class="px-3 py-2 border-b border-border">
                        <span class="text-sm font-semibold text-foreground"
                            >Notifications</span
                        >
                    </div>
                    <div class="max-h-72 overflow-y-auto">
                        {#if notifLoading}
                            <div class="flex items-center justify-center py-8">
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
                                        {#if !n.is_read}
                                            <span
                                                class="mt-1.5 size-2 rounded-full bg-blue-500 shrink-0"
                                            ></span>
                                        {:else}
                                            <span class="mt-1.5 size-2 shrink-0"
                                            ></span>
                                        {/if}
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
                                                {formatNotifDate(n.created_at)}
                                            </p>
                                        </div>
                                    </div>
                                </button>
                            {/each}
                        {/if}
                    </div>
                </DropdownMenu.Content>
            </DropdownMenu.Root>

            <DropdownMenu.Root>
                <DropdownMenu.Trigger>
                    {#snippet child({ props })}
                        <Button.Root {...props} variant="ghost" size="sm">
                            <Ellipsis class="size-4" />
                            <span>Menu</span>
                        </Button.Root>
                    {/snippet}
                </DropdownMenu.Trigger>
                <DropdownMenu.Content class="w-56" align="start">
                    {#each moreTabs as tab}
                        <DropdownMenu.Item
                            onclick={() => goto(tab.href)}
                            class="py-2.5"
                        >
                            <tab.icon class="size-4" />
                            <span class="text-sm">{tab.label}</span>
                        </DropdownMenu.Item>
                    {/each}
                </DropdownMenu.Content>
            </DropdownMenu.Root>
        </div>
        <Button.Root
            variant="ghost"
            size="sm"
            onclick={handleLogout}
            disabled={loggingOut}
        >
            <LogOut class="size-4" />
            {loggingOut ? "Signing out…" : "Sign out"}
        </Button.Root>
    </header>

    <!-- Main content -->
    <main class="flex flex-1 p-6">
        {@render children()}
    </main>
</div>
