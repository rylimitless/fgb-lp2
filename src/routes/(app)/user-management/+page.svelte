<script lang="ts">
    import {
        Users,
        Plus,
        Trash2,
        AlertCircle,
        Pencil,
        XCircle,
    } from "@lucide/svelte";
    import * as Button from "$lib/components/ui/button";
    import * as Input from "$lib/components/ui/input";
    import type { User } from "./+page.server";

    let { data } = $props();
    let users: User[] = $state([]);
    $effect(() => {
        users = data.users ?? [];
    });

    const ALL_ROLES = [
        { id: "end user", label: "End User" },
        { id: "content creator", label: "Content Creator" },
        { id: "approver", label: "Approver" },
        { id: "admin", label: "Admin" },
        { id: "manager", label: "Manager" },
        { id: "auditor", label: "Auditor" },
    ];

    // ---- Create form state ----
    let showForm = $state(false);
    let formEmail = $state("");
    let formPassword = $state("");
    let formName = $state("");
    let formRoles = $state<string[]>(["end user"]);
    let formError = $state("");
    let formSubmitting = $state(false);

    let deleteError = $state("");

    function toggleRole(roles: string[], role: string): string[] {
        if (roles.includes(role)) return roles.filter((r) => r !== role);
        return [...roles, role];
    }

    // ---- Edit roles state ----
    let editingUser = $state<User | null>(null);
    let editRoles = $state<string[]>([]);
    let editError = $state("");
    let editSaving = $state(false);

    function openEdit(user: User) {
        editingUser = user;
        editRoles = [
            ...(user.roles && user.roles.length > 0 ? user.roles : [user.role]),
        ];
        editError = "";
    }

    function closeEdit() {
        editingUser = null;
        editRoles = [];
        editError = "";
    }

    async function handleSaveRoles() {
        if (!editingUser) return;
        const userId = editingUser.id;
        if (editRoles.length === 0) {
            editError = "At least one role must be selected";
            return;
        }
        editSaving = true;
        editError = "";
        try {
            const res = await fetch(`/api/admin/users/${userId}/roles`, {
                method: "PUT",
                credentials: "include",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ roles: editRoles }),
            });
            if (!res.ok) {
                const err = await res.json();
                editError = err.error ?? "Failed to update roles";
                return;
            }
            // Update local state
            users = users.map((u) =>
                u.id === userId ? { ...u, roles: editRoles } : u,
            );
            closeEdit();
        } catch {
            editError = "Network error";
        } finally {
            editSaving = false;
        }
    }

    // ---- Create ----
    async function handleCreate() {
        formError = "";
        if (formRoles.length === 0) {
            formError = "At least one role must be selected";
            return;
        }
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
                    role: formRoles[0],
                    roles: formRoles,
                }),
            });
            if (!res.ok) {
                const err = await res.json();
                formError = err.error ?? "Failed to create user";
                return;
            }
            await refreshUsers();
            showForm = false;
            formEmail = "";
            formPassword = "";
            formName = "";
            formRoles = ["end user"];
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
                class="flex flex-col gap-4"
                onsubmit={(e) => {
                    e.preventDefault();
                    handleCreate();
                }}
            >
                <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
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
                    <div class="flex flex-col gap-1.5 sm:col-span-2">
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
                </div>

                <!-- Multi-role selection -->
                <div class="flex flex-col gap-2">
                    <label class="text-xs font-medium text-foreground"
                        >Roles</label
                    >
                    <p class="text-xs text-muted-foreground">
                        A user can have multiple roles. Select all that apply.
                    </p>
                    <div class="flex flex-wrap gap-2">
                        {#each ALL_ROLES as r}
                            <label
                                class="flex items-center gap-1.5 rounded-md border px-2.5 py-1.5 text-xs cursor-pointer hover:bg-muted/30 transition-colors {formRoles.includes(
                                    r.id,
                                )
                                    ? 'border-primary bg-primary/5 text-primary'
                                    : 'border-border text-muted-foreground'}"
                            >
                                <input
                                    type="checkbox"
                                    class="sr-only"
                                    checked={formRoles.includes(r.id)}
                                    onchange={() =>
                                        (formRoles = toggleRole(
                                            formRoles,
                                            r.id,
                                        ))}
                                />
                                <span>{r.label}</span>
                            </label>
                        {/each}
                    </div>
                </div>

                <div class="flex justify-end">
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
                                Roles
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
                                    <div class="flex flex-wrap gap-1">
                                        {#each user.roles && user.roles.length > 0 ? user.roles : [user.role] as r}
                                            <span
                                                class="inline-flex items-center rounded-md border px-2 py-0.5 text-xs font-medium {roleBadgeClass(
                                                    r,
                                                )}"
                                            >
                                                {r}
                                            </span>
                                        {/each}
                                    </div>
                                </td>
                                <td
                                    class="px-5 py-3 text-muted-foreground text-xs"
                                >
                                    {user.created_at}
                                </td>
                                <td class="px-5 py-3 text-right">
                                    <div
                                        class="flex items-center justify-end gap-1"
                                    >
                                        <Button.Root
                                            variant="ghost"
                                            size="icon"
                                            class="size-8 text-muted-foreground hover:text-primary"
                                            onclick={() => openEdit(user)}
                                        >
                                            <Pencil class="size-3.5" />
                                        </Button.Root>
                                        <Button.Root
                                            variant="ghost"
                                            size="icon"
                                            class="size-8 text-muted-foreground hover:text-rose-500"
                                            onclick={() =>
                                                handleDelete(
                                                    user.id,
                                                    user.name,
                                                )}
                                        >
                                            <Trash2 class="size-3.5" />
                                        </Button.Root>
                                    </div>
                                </td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
            </div>
        {/if}
    </div>
</div>

<!-- Edit Roles Dialog -->
{#if editingUser}
    {@const u = editingUser!}
    <div
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
        onclick={closeEdit}
        role="dialog"
        aria-modal="true"
    >
        <div
            class="bg-card border border-border rounded-xl shadow-xl w-full max-w-md mx-4"
            onclick={(e: MouseEvent) => e.stopPropagation()}
        >
            <!-- Header -->
            <div
                class="flex items-center justify-between px-6 py-4 border-b border-border"
            >
                <div class="min-w-0">
                    <h3 class="text-base font-semibold text-foreground">
                        Edit Roles
                    </h3>
                    <p class="text-sm text-muted-foreground mt-0.5 truncate">
                        {u.name} ({u.email})
                    </p>
                </div>
                <Button.Root
                    variant="ghost"
                    size="icon-sm"
                    onclick={closeEdit}
                    class="shrink-0 ml-3"
                >
                    <XCircle class="size-5" />
                </Button.Root>
            </div>

            <!-- Body -->
            <div class="px-6 py-4 flex flex-col gap-4">
                {#if editError}
                    <div
                        class="flex items-center gap-2 rounded-lg border border-rose-500/20 bg-rose-500/10 p-3 text-xs text-rose-500"
                    >
                        <AlertCircle class="size-3.5 shrink-0" />
                        <span>{editError}</span>
                    </div>
                {/if}

                <div class="flex flex-col gap-2">
                    <label class="text-xs font-medium text-foreground"
                        >Select roles</label
                    >
                    <div class="flex flex-wrap gap-2">
                        {#each ALL_ROLES as r}
                            <label
                                class="flex items-center gap-1.5 rounded-md border px-2.5 py-1.5 text-xs cursor-pointer hover:bg-muted/30 transition-colors {editRoles.includes(
                                    r.id,
                                )
                                    ? 'border-primary bg-primary/5 text-primary'
                                    : 'border-border text-muted-foreground'}"
                            >
                                <input
                                    type="checkbox"
                                    class="sr-only"
                                    checked={editRoles.includes(r.id)}
                                    onchange={() =>
                                        (editRoles = toggleRole(
                                            editRoles,
                                            r.id,
                                        ))}
                                />
                                <span>{r.label}</span>
                            </label>
                        {/each}
                    </div>
                </div>
            </div>

            <!-- Footer -->
            <div
                class="flex items-center justify-end gap-2 px-6 py-4 border-t border-border"
            >
                <Button.Root
                    variant="ghost"
                    size="sm"
                    onclick={closeEdit}
                    disabled={editSaving}
                >
                    Cancel
                </Button.Root>
                <Button.Root
                    size="sm"
                    onclick={handleSaveRoles}
                    disabled={editSaving}
                >
                    {editSaving ? "Saving…" : "Save Roles"}
                </Button.Root>
            </div>
        </div>
    </div>
{/if}
