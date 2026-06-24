<script lang="ts">
    import {
        Building2,
        Plus,
        Trash2,
        AlertCircle,
        LoaderCircle,
        Search,
        UserPlus,
        UserX,
    } from "@lucide/svelte";
    import { PageHeader, StatCard, showToast } from "$lib/components/brand";
    import * as Button from "$lib/components/ui/button";
    import type { Department, DepartmentUser } from "./+page.server";

    let { data } = $props();
    let departments: Department[] = $state([]);
    $effect(() => {
        departments = data.departments ?? [];
    });
    // ---- Create ----
    let showCreate = $state(false);
    let newName = $state("");
    let createError = $state("");
    let creating = $state(false);

    async function handleCreate() {
        createError = "";
        if (!newName.trim()) {
            createError = "Department name is required";
            return;
        }
        creating = true;
        try {
            const res = await fetch("/api/admin/departments", {
                method: "POST",
                credentials: "include",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ name: newName.trim() }),
            });
            if (!res.ok) {
                const err = await res.json();
                createError = err.error ?? "Failed to create department";
                return;
            }
            const dep = await res.json();
            departments = [...departments, dep];
            showToast(`Department "${dep.name}" created.`, {
                title: "Created",
                variant: "success",
            });
            newName = "";
            showCreate = false;
        } catch {
            createError = "Network error";
        } finally {
            creating = false;
        }
    }

    // ---- Delete ----
    let deletingId = $state<number | null>(null);

    async function handleDelete(dep: Department) {
        if (deletingId !== null) return;
        if (
            !confirm(
                `Delete department "${dep.name}"? Users will be unassigned.`,
            )
        )
            return;
        deletingId = dep.id;
        try {
            const res = await fetch(`/api/admin/departments/${dep.id}`, {
                method: "DELETE",
                credentials: "include",
            });
            if (res.ok) {
                departments = departments.filter((d) => d.id !== dep.id);
                if (expandedId === dep.id) closeExpand();
                showToast(`"${dep.name}" deleted.`, {
                    title: "Deleted",
                    variant: "default",
                });
            }
        } catch {
            /* ignore */
        }
        deletingId = null;
    }

    // ---- Expand department ----
    let expandedId = $state<number | null>(null);
    let expandedMembers = $state<DepartmentUser[]>([]);
    let membersLoading = $state(false);

    async function toggleExpand(dep: Department) {
        if (expandedId === dep.id) {
            closeExpand();
            return;
        }
        expandedId = dep.id;
        expandedMembers = [];
        membersLoading = true;
        try {
            const res = await fetch(`/api/admin/departments/${dep.id}/users`, {
                credentials: "include",
            });
            if (res.ok) {
                expandedMembers = await res.json();
            }
        } catch {
            /* ignore */
        }
        membersLoading = false;
    }

    function closeExpand() {
        expandedId = null;
        expandedMembers = [];
        addUserSearch = "";
        userSearchResults = [];
        allUsersCache = [];
    }

    // ---- Add user to department ----
    let addUserSearch = $state("");
    let userSearchResults = $state<any[]>([]);
    let userSearchLoading = $state(false);
    let addUserError = $state("");
    let addingUserId = $state<number | null>(null);

    let searchTimeout: ReturnType<typeof setTimeout>;
    let allUsersCache = $state<any[]>([]);

    function onUserSearchInput() {
        clearTimeout(searchTimeout);
        const q = addUserSearch.trim();
        if (!q) {
            userSearchResults = [];
            return;
        }
        searchTimeout = setTimeout(async () => {
            userSearchLoading = true;
            try {
                // Fetch all users if not cached
                if (allUsersCache.length === 0) {
                    const res = await fetch("/api/admin/users", {
                        credentials: "include",
                    });
                    if (res.ok) allUsersCache = await res.json();
                }
                const memberIds = new Set(expandedMembers.map((m) => m.id));
                const ql = q.toLowerCase();
                userSearchResults = allUsersCache
                    .filter(
                        (u: any) =>
                            !memberIds.has(u.id) &&
                            (u.name.toLowerCase().includes(ql) ||
                                u.email.toLowerCase().includes(ql)),
                    )
                    .slice(0, 8);
            } catch {
                /* ignore */
            }
            userSearchLoading = false;
        }, 250);
    }

    async function handleAddUser(userId: number, userName: string) {
        if (!expandedId || addingUserId !== null) return;
        addUserError = "";
        addingUserId = userId;
        try {
            const res = await fetch(
                `/api/admin/departments/${expandedId}/users`,
                {
                    method: "POST",
                    credentials: "include",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify({ user_id: userId }),
                },
            );
            if (!res.ok) {
                const err = await res.json();
                addUserError = err.error ?? "Failed to add user";
                return;
            }
            // Refresh members
            const mRes = await fetch(
                `/api/admin/departments/${expandedId}/users`,
                { credentials: "include" },
            );
            if (mRes.ok) expandedMembers = await mRes.json();
            // Clear search
            addUserSearch = "";
            userSearchResults = [];
            showToast(`${userName} added to department.`, {
                title: "Member added",
                variant: "success",
            });
        } catch {
            addUserError = "Network error";
        } finally {
            addingUserId = null;
        }
    }

    async function handleRemoveUser(userId: number, userName: string) {
        if (!expandedId) return;
        if (!confirm(`Remove ${userName} from this department?`)) return;
        try {
            const res = await fetch(
                `/api/admin/departments/${expandedId}/users`,
                {
                    method: "DELETE",
                    credentials: "include",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify({ user_id: userId }),
                },
            );
            if (res.ok) {
                expandedMembers = expandedMembers.filter(
                    (m) => m.id !== userId,
                );
                showToast(`${userName} removed.`, {
                    title: "Member removed",
                    variant: "default",
                });
            }
        } catch {
            /* ignore */
        }
    }

    function formatDate(d: string) {
        if (!d) return "";
        return new Date(d).toLocaleDateString("en-US", {
            year: "numeric",
            month: "short",
            day: "numeric",
        });
    }
</script>

<div class="flex w-full max-w-4xl mx-auto flex-col gap-6">
    <PageHeader
        title="Department management"
        eyebrow="Organisation"
        description="Create departments, assign users, and enrol entire teams in courses from the Content Repository."
    >
        {#snippet icon()}
            <Building2 class="size-6 text-primary" />
        {/snippet}
        {#snippet actions()}
            <Button.Root
                variant={showCreate ? "outline" : "default"}
                size="sm"
                onclick={() => {
                    showCreate = !showCreate;
                    createError = "";
                }}
            >
                {#if showCreate}
                    Cancel
                {:else}
                    <Plus class="size-3.5" />
                    <span>Add department</span>
                {/if}
            </Button.Root>
        {/snippet}
    </PageHeader>

    <!-- Stat row -->
    <div class="grid grid-cols-1 gap-3">
        <div class="motion-rise-in motion-stagger-1">
            <StatCard
                label="Departments"
                value={departments.length}
                tone="primary"
                hint="Click a department to manage members"
            >
                {#snippet icon()}
                    <Building2 class="size-3.5" />
                {/snippet}
            </StatCard>
        </div>
    </div>

    <!-- Create form -->
    {#if showCreate}
        <div class="rounded-xl border border-border bg-card p-5 motion-rise-in">
            <h2 class="text-sm font-semibold text-foreground mb-4">
                Create Department
            </h2>
            {#if createError}
                <div
                    class="mb-4 flex items-center gap-2 rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-xs text-destructive"
                >
                    <AlertCircle class="size-3.5 shrink-0" />
                    <span>{createError}</span>
                </div>
            {/if}
            <form
                class="flex items-end gap-3"
                onsubmit={(e) => {
                    e.preventDefault();
                    handleCreate();
                }}
            >
                <div class="flex-1 flex flex-col gap-1.5">
                    <label
                        class="text-xs font-medium text-foreground"
                        for="dept-name">Name</label
                    >
                    <input
                        id="dept-name"
                        type="text"
                        placeholder="e.g. Fire Safety Division"
                        bind:value={newName}
                        class="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                    />
                </div>
                <Button.Root type="submit" size="sm" disabled={creating}>
                    {creating ? "Creating…" : "Create"}
                </Button.Root>
            </form>
        </div>
    {/if}

    <!-- Department list -->
    <div class="rounded-xl border border-border bg-card overflow-hidden">
        {#if departments.length === 0}
            <div
                class="flex flex-col items-center justify-center py-16 gap-3 text-muted-foreground"
            >
                <Building2 class="size-10" />
                <p class="text-sm">No departments yet.</p>
                <p class="text-xs">
                    Create your first department to start organising users.
                </p>
            </div>
        {:else}
            <div class="divide-y divide-border">
                {#each departments as dep, i (dep.id)}
                    {@const expanded = expandedId === dep.id}
                    <div>
                        <button
                            class="w-full flex items-center gap-3 px-5 py-3 text-left hover:bg-muted/30 transition-colors"
                            onclick={() => toggleExpand(dep)}
                        >
                            <div
                                class="size-8 rounded-lg bg-accent/10 flex items-center justify-center shrink-0"
                            >
                                <Building2 class="size-4 text-accent" />
                            </div>
                            <div class="min-w-0 flex-1">
                                <p
                                    class="text-sm font-medium text-foreground truncate"
                                >
                                    {dep.name}
                                </p>
                                <p class="text-xs text-muted-foreground">
                                    Created {formatDate(dep.created_at)}
                                </p>
                            </div>
                            <Button.Root
                                variant="ghost"
                                size="icon-sm"
                                onclick={(e: MouseEvent) => {
                                    e.stopPropagation();
                                    handleDelete(dep);
                                }}
                                disabled={deletingId === dep.id}
                                class="text-muted-foreground hover:text-destructive"
                                title="Delete department"
                            >
                                {#if deletingId === dep.id}
                                    <LoaderCircle class="size-4 animate-spin" />
                                {:else}
                                    <Trash2 class="size-4" />
                                {/if}
                            </Button.Root>
                        </button>

                        {#if expanded}
                            <div
                                class="border-t border-border bg-muted/20 px-5 py-4 motion-rise-in"
                            >
                                <!-- Add user search -->
                                <div class="mb-4">
                                    <div class="relative">
                                        <Search
                                            class="absolute left-2.5 top-1/2 -translate-y-1/2 size-3.5 text-muted-foreground"
                                        />
                                        <input
                                            type="text"
                                            placeholder="Search users to add..."
                                            bind:value={addUserSearch}
                                            oninput={onUserSearchInput}
                                            class="w-full rounded-lg border border-input bg-background pl-8 pr-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                                        />
                                    </div>
                                    {#if addUserError}
                                        <p
                                            class="mt-1 text-xs text-destructive"
                                        >
                                            {addUserError}
                                        </p>
                                    {/if}
                                    {#if userSearchLoading}
                                        <div class="flex justify-center py-3">
                                            <LoaderCircle
                                                class="size-4 text-muted-foreground animate-spin"
                                            />
                                        </div>
                                    {:else if userSearchResults.length > 0}
                                        <div
                                            class="mt-2 flex flex-col gap-1 max-h-48 overflow-y-auto rounded-lg border border-border bg-card"
                                        >
                                            {#each userSearchResults as u (u.id)}
                                                <button
                                                    class="flex items-center gap-2.5 px-3 py-2 text-left hover:bg-muted/30 transition-colors disabled:opacity-40"
                                                    disabled={addingUserId ===
                                                        u.id}
                                                    onclick={() =>
                                                        handleAddUser(
                                                            u.id,
                                                            u.name,
                                                        )}
                                                >
                                                    {#if addingUserId === u.id}
                                                        <LoaderCircle
                                                            class="size-3.5 animate-spin shrink-0"
                                                        />
                                                    {:else}
                                                        <UserPlus
                                                            class="size-3.5 text-muted-foreground shrink-0"
                                                        />
                                                    {/if}
                                                    <div class="min-w-0">
                                                        <p
                                                            class="text-sm text-foreground truncate"
                                                        >
                                                            {u.name}
                                                        </p>
                                                        <p
                                                            class="text-xs text-muted-foreground truncate"
                                                        >
                                                            {u.email}
                                                        </p>
                                                    </div>
                                                </button>
                                            {/each}
                                        </div>
                                    {/if}
                                </div>

                                <!-- Members list -->
                                <h4
                                    class="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-2"
                                >
                                    Members ({expandedMembers.length})
                                </h4>
                                {#if membersLoading}
                                    <div class="flex justify-center py-6">
                                        <LoaderCircle
                                            class="size-5 text-muted-foreground animate-spin"
                                        />
                                    </div>
                                {:else if expandedMembers.length === 0}
                                    <p
                                        class="text-sm text-muted-foreground text-center py-4"
                                    >
                                        No members yet. Search above to add
                                        users.
                                    </p>
                                {:else}
                                    <div
                                        class="flex flex-col gap-1 max-h-64 overflow-y-auto"
                                    >
                                        {#each expandedMembers as member (member.id)}
                                            <div
                                                class="flex items-center gap-2.5 rounded-lg border border-border bg-card px-3 py-2"
                                            >
                                                <div
                                                    class="size-7 rounded-full bg-primary/10 flex items-center justify-center shrink-0"
                                                >
                                                    <span
                                                        class="text-[10px] font-bold text-primary"
                                                    >
                                                        {member.name?.[0]?.toUpperCase() ??
                                                            "?"}
                                                    </span>
                                                </div>
                                                <div class="min-w-0 flex-1">
                                                    <p
                                                        class="text-sm text-foreground truncate"
                                                    >
                                                        {member.name}
                                                    </p>
                                                    <p
                                                        class="text-xs text-muted-foreground truncate"
                                                    >
                                                        {member.email}
                                                    </p>
                                                </div>
                                                <Button.Root
                                                    variant="ghost"
                                                    size="icon-sm"
                                                    onclick={() =>
                                                        handleRemoveUser(
                                                            member.id,
                                                            member.name,
                                                        )}
                                                    class="text-muted-foreground hover:text-destructive"
                                                    title="Remove from department"
                                                >
                                                    <UserX class="size-3.5" />
                                                </Button.Root>
                                            </div>
                                        {/each}
                                    </div>
                                {/if}
                            </div>
                        {/if}
                    </div>
                {/each}
            </div>
        {/if}
    </div>
</div>
