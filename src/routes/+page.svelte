<script lang="ts">
    import { LogOut, LayoutDashboard } from "@lucide/svelte";
    import * as Button from "$lib/components/ui/button";
    import { goto } from "$app/navigation";

    let loggingOut = $state(false);

    async function handleLogout() {
        loggingOut = true;
        try {
            await fetch("http://localhost:5555/api/logout", {
                method: "POST",
                credentials: "include",
            });
        } catch {
            // proceed even if server call fails
        }
        await goto("/login");
    }
</script>

<div class="flex min-h-svh flex-col bg-background">
    <!-- Top bar -->
    <header
        class="sticky top-0 z-10 flex h-14 items-center justify-between border-b border-border bg-background px-6"
    >
        <div class="flex items-center gap-2">
            <LayoutDashboard class="size-5 text-primary" />
            <span class="text-sm font-semibold tracking-tight text-foreground"
                >Dashboard</span
            >
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
    <main class="flex flex-1 items-center justify-center p-6">
        <div class="text-center">
            <h1 class="text-3xl font-semibold tracking-tight text-foreground">
                Welcome to FGB
            </h1>
            <p class="mt-3 text-muted-foreground">
                You are signed in. This is the main page.
            </p>
        </div>
    </main>
</div>
