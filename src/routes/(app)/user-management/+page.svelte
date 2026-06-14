<script lang="ts">
    import { Users, Plus, Trash2, AlertCircle } from "@lucide/svelte";
    import * as Button from "$lib/components/ui/button";
    import * as Input from "$lib/components/ui/input";
    import type { User } from "./+page.server";

    let { data } = $props();
    let users: User[] = $state([]);
    $effect(() => {
        users = data.users ?? [];
    });

    const ROLES = [
        "end user",
        "content creator",
        "admin",
        "approver",
        "manager",
        "auditor",
    ];

    let showForm = $state(false);
    let formEmail = $state("");
    let formPassword = $state("");
    let formName = $state("");
    let formRole = $state("end user");
    let formError = $state("");
    let formSubmitting = $state(false);

    let deleteError = $state("");

    async function handleCreate() {
        formError = "";
        formSubmitting = true;
        try {
            const res = await fetch("/api/admin/users", {
                method: "POST",
                credentials: "include",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    email: formEmail,
                    password: formPassword,
                    name: formName,
                    role: formRole,
                }),
            });
            if (!res.ok) {
                const err = await res.json();
                formError = err.error ?? "Failed to create user";
                return;
            }
            // Refresh the list
            await refreshUsers();
            showForm = false;
            formEmail = "";
            formPassword = "";
            formName = "";
            formRole = "end user";
        } catch {
            formError = "Network error";
        } finally {
            formSubmitting = false;
        }
    }

    async function handleDelete(id: number, name: string) {
        if (!confirm(`Delete user "${name}"? This cannot be undone.`)) return;
        deleteError = "";
        try {
            const res = await fetch(`/api/admin/users/${id}`, {
                method: "DELETE",
                credentials: "include",
            });
            if (!res.ok) {
                const err = await res.json();
                deleteError = err.error ?? "Failed to delete user";
                return;
            }
            users = users.filter((u) => u.id !== id);
        } catch {
            deleteError = "Network error";
        }
    }

    async function refreshUsers() {
        const res = await fetch("/api/admin/users", { credentials: "include" });
        if (res.ok) {
            users = await res.json();
        }
    }

    function roleBadgeClass(role: string): string {
        switch (role) {
            case "admin":
                return "bg-rose-500/10 text-rose-500 border-rose-500/20";
            case "content creator":
                return "bg-blue-500/10 text-blue-500 border-blue-500/20";
            case "approver":
                return "bg-amber-500/10 text-amber-500 border-amber-500/20";
            case "manager":
                return "bg-violet-500/10 text-violet-500 border-violet-500/20";
            case "auditor":
                return "bg-cyan-500/10 text-cyan-500 border-cyan-500/20";
            default:
                return "bg-muted text-muted-foreground border-border";
        }
    }
</script>

<div class="flex w-full max-w-4xl mx-auto flex-col gap-6">
    <div class="flex items-center justify-between">
        <div class="flex items-center gap-3">
            <Users class="size-6 text-primary" />
            <h1 class="text-2xl font-semibold text-foreground">
                User Management
            </h1>
        </div>
        <Button.Root
            variant={showForm ? "outline" : "default"}
            size="sm"
            onclick={() => (showForm = !showForm)}
        >
            {#if showForm}
                Cancel
            {:else}
                <Plus class="size-3.5" />
                <span>Add User</span>
            {/if}
        </Button.Root>
    </div>

    <!-- Create User Form -->
    {#if showForm}
        <div class="rounded-xl border border-border bg-card p-5">
            <h2 class="text-sm font-semibold text-foreground mb-4">
                Create New User
            </h2>
            {#if formError}
                <div
                    class="mb-4 flex items-center gap-2 rounded-lg border border-rose-500/20 bg-rose-500/10 p-3 text-xs text-rose-500"
                >
                    <AlertCircle class="size-3.5 shrink-0" />
                    <span>{formError}</span>
                </div>
            {/if}
            <form
                class="grid grid-cols-1 sm:grid-cols-2 gap-4"
                onsubmit={(e) => {
                    e.preventDefault();
                    handleCreate();
                }}
            >
                <div class="flex flex-col gap-1.5">
                    <label
                        class="text-xs font-medium text-foreground"
                        for="name">Name</label
                    >
                    <Input.Root
                        id="name"
                        type="text"
                        placeholder="Full name"
                        bind:value={formName}
                        required
                    />
                </div>
                <div class="flex flex-col gap-1.5">
                    <label
                        class="text-xs font-medium text-foreground"
                        for="email">Email</label
                    >
                    <Input.Root
                        id="email"
                        type="email"
                        placeholder="user@example.com"
                        bind:value={formEmail}
                        required
                    />
                </div>
                <div class="flex flex-col gap-1.5">
                    <label
                        class="text-xs font-medium text-foreground"
                        for="password">Password</label
                    >
                    <Input.Root
                        id="password"
                        type="password"
                        placeholder="Min. 8 characters"
                        bind:value={formPassword}
                        required
                        minlength={8}
                    />
                </div>
                <div class="flex flex-col gap-1.5">
                    <label
                        class="text-xs font-medium text-foreground"
                        for="role">Role</label
                    >
                    <select
                        id="role"
                        bind:value={formRole}
                        class="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm text-foreground shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
                    >
                        {#each ROLES as role}
                            <option
                                class="text-foreground bg-background capitalize"
                                value={role}
                            >
                                {role}
                            </option>
                        {/each}
                    </select>
                </div>
                <div class="sm:col-span-2 flex justify-end">
                    <Button.Root
                        type="submit"
                        size="sm"
                        disabled={formSubmitting}
                    >
                        {formSubmitting ? "Creating…" : "Create User"}
                    </Button.Root>
                </div>
            </form>
        </div>
    {/if}

    <!-- Error message -->
    {#if deleteError}
        <div
            class="flex items-center gap-2 rounded-lg border border-rose-500/20 bg-rose-500/10 p-3 text-xs text-rose-500"
        >
            <AlertCircle class="size-3.5 shrink-0" />
            <span>{deleteError}</span>
        </div>
    {/if}

    <!-- Users Table -->
    <div class="rounded-xl border border-border bg-card overflow-hidden">
        {#if users.length === 0}
            <div class="p-8 text-center">
                <Users class="size-8 text-muted-foreground/40 mx-auto mb-3" />
                <p class="text-sm text-muted-foreground">No users found.</p>
            </div>
        {:else}
            <div class="overflow-x-auto">
                <table class="w-full text-sm">
                    <thead>
                        <tr class="border-b border-border bg-muted/50">
                            <th
                                class="text-left px-5 py-3 text-xs font-medium text-muted-foreground"
                            >
                                Name
                            </th>
                            <th
                                class="text-left px-5 py-3 text-xs font-medium text-muted-foreground"
                            >
                                Email
                            </th>
                            <th
                                class="text-left px-5 py-3 text-xs font-medium text-muted-foreground"
                            >
                                Role
                            </th>
                            <th
                                class="text-left px-5 py-3 text-xs font-medium text-muted-foreground"
                            >
                                Created
                            </th>
                            <th
                                class="text-right px-5 py-3 text-xs font-medium text-muted-foreground"
                            >
                                Actions
                            </th>
                        </tr>
                    </thead>
                    <tbody>
                        {#each users as user}
                            <tr
                                class="border-b border-border last:border-b-0 hover:bg-muted/30 transition-colors"
                            >
                                <td
                                    class="px-5 py-3 font-medium text-foreground"
                                >
                                    {user.name}
                                </td>
                                <td class="px-5 py-3 text-muted-foreground"
                                    >{user.email}</td
                                >
                                <td class="px-5 py-3">
                                    <span
                                        class="inline-flex items-center rounded-md border px-2 py-0.5 text-xs font-medium {roleBadgeClass(
                                            user.role,
                                        )}"
                                    >
                                        {user.role}
                                    </span>
                                </td>
                                <td
                                    class="px-5 py-3 text-muted-foreground text-xs"
                                >
                                    {user.created_at}
                                </td>
                                <td class="px-5 py-3 text-right">
                                    <Button.Root
                                        variant="ghost"
                                        size="icon"
                                        class="size-8 text-muted-foreground hover:text-rose-500"
                                        onclick={() =>
                                            handleDelete(user.id, user.name)}
                                    >
                                        <Trash2 class="size-3.5" />
                                    </Button.Root>
                                </td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
            </div>
        {/if}
    </div>
</div>
