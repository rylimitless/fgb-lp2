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
        Ellipsis,
    } from "@lucide/svelte";
    import * as Button from "$lib/components/ui/button";
    import * as Tabs from "$lib/components/ui/tabs";
    import * as DropdownMenu from "$lib/components/ui/dropdown-menu/index.js";

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
    ];

    let currentTab = $state(
        $page.url.pathname.startsWith("/gia-coach") ? "gia-coach" : "dashboard",
    );

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
