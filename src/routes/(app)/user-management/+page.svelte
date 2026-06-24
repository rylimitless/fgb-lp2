<script lang="ts">
    import {
        PageHeader,
        StatCard,
        PremiumTable,
        showToast,
    } from "$lib/components/brand";
    import {
        Users,
        Plus,
        Trash2,
        AlertCircle,
        Pencil,
        XCircle,
        BookOpen,
        GraduationCap,
        LoaderCircle,
        Search,
        Check,
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
            const savedName = editingUser?.name ?? "User";
            closeEdit();
            showToast(`Roles updated for ${savedName}.`, {
                title: "Saved",
                variant: "success",
            });
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
            showToast(`${formName || formEmail} added to the Academy.`, {
                title: "User created",
                variant: "success",
            });
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
            showToast(`${name} removed.`, {
                title: "User deleted",
                variant: "default",
            });
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
        // Role -> brand semantic token. 4-token hierarchy per BRAND.md §4.3:
        // admin -> primary (strongest), manager -> accent (leadership gold),
        // approver -> warning (gatekeeping), auditor -> info (oversight),
        // content creator -> info (productive), default -> neutral.
        switch (role) {
            case "admin":
                return "bg-primary/10 text-primary border-primary/30";
            case "manager":
                return "bg-accent-soft text-accent-foreground border-accent/40";
            case "approver":
                return "bg-warning/10 text-warning border-warning/30";
            case "auditor":
            case "content creator":
                return "bg-info/10 text-info border-info/30";
            default:
                return "bg-muted text-muted-foreground border-border";
        }
    }

    // ---- Enroll in Course ----
    let enrollingUser = $state<User | null>(null);
    let enrollDialogOpen = $state(false);
    let availableCourses = $state<any[]>([]);
    let enrollCoursesLoading = $state(false);
    let selectedCourseId = $state<number | null>(null);
    let enrollError = $state("");
    let enrollSaving = $state(false);
    let enrollSearch = $state("");

    async function openEnroll(user: User) {
        enrollingUser = user;
        enrollDialogOpen = true;
        selectedCourseId = null;
        enrollError = "";
        enrollSearch = "";
        availableCourses = [];
        enrollCoursesLoading = true;
        try {
            const res = await fetch("/api/courses/published", {
                credentials: "include",
            });
            if (res.ok) {
                availableCourses = await res.json();
            }
        } catch {
            /* ignore */
        }
        enrollCoursesLoading = false;
    }

    function closeEnroll() {
        enrollingUser = null;
        enrollDialogOpen = false;
        selectedCourseId = null;
        enrollError = "";
    }

    async function handleEnroll() {
        if (!enrollingUser || !selectedCourseId) return;
        enrollSaving = true;
        enrollError = "";
        try {
            const res = await fetch("/api/admin/enrollments", {
                method: "POST",
                credentials: "include",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    user_id: enrollingUser.id,
                    course_id: selectedCourseId,
                }),
            });
            if (!res.ok) {
                const err = await res.json();
                enrollError = err.error ?? "Failed to enroll user";
                return;
            }
            const result = await res.json();
            showToast(
                `${result.user_name} enrolled in "${result.course_title}".`,
                { title: "Enrolled", variant: "success" },
            );
            closeEnroll();
        } catch {
            enrollError = "Network error";
        } finally {
            enrollSaving = false;
        }
    }

    let filteredCourses = $derived(
        enrollSearch
            ? availableCourses.filter(
                  (c: any) =>
                      c.title
                          .toLowerCase()
                          .includes(enrollSearch.toLowerCase()) ||
                      (c.description ?? "")
                          .toLowerCase()
                          .includes(enrollSearch.toLowerCase()),
              )
            : availableCourses,
    );
</script>

<div class="flex w-full max-w-4xl mx-auto flex-col gap-6">
    <PageHeader
        title="User management"
        eyebrow="Governance"
        description="Manage Academy accounts, assign roles, and remove access. Every change is written to the audit log."
    >
        {#snippet icon()}
            <Users class="size-6 text-primary" />
        {/snippet}
        {#snippet actions()}
            <Button.Root
                variant={showForm ? "outline" : "default"}
                size="sm"
                onclick={() => (showForm = !showForm)}
            >
                {#if showForm}
                    Cancel
                {:else}
                    <Plus class="size-3.5" />
                    <span>Add user</span>
                {/if}
            </Button.Root>
        {/snippet}
    </PageHeader>

    <!-- User count stat row -->
    <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
        <div class="motion-rise-in motion-stagger-1">
            <StatCard
                label="Total users"
                value={data.users?.length ?? 0}
                tone="neutral"
                hint="all roles"
            >
                {#snippet icon()}
                    <Users class="size-3.5" />
                {/snippet}
            </StatCard>
        </div>
        <div class="motion-rise-in motion-stagger-2">
            <StatCard
                label="Administrators"
                value={data.users?.filter((u: any) =>
                    (u.roles ?? [u.role]).includes("admin"),
                ).length ?? 0}
                tone="primary"
                hint="root access"
            />
        </div>
        <div class="motion-rise-in motion-stagger-3">
            <StatCard
                label="Content creators"
                value={data.users?.filter((u: any) =>
                    (u.roles ?? [u.role]).includes("content creator"),
                ).length ?? 0}
                tone="info"
                hint="can author courses"
            />
        </div>
        <div class="motion-rise-in motion-stagger-4">
            <StatCard
                label="Approvers"
                value={data.users?.filter((u: any) =>
                    (u.roles ?? [u.role]).includes("approver"),
                ).length ?? 0}
                tone="warning"
                hint="can publish content"
            />
        </div>
    </div>

    <!-- Create User Form -->
    {#if showForm}
        <div class="rounded-xl border border-border bg-card p-5">
            <h2 class="text-sm font-semibold text-foreground mb-4">
                Create New User
            </h2>
            {#if formError}
                <div
                    class="mb-4 flex items-center gap-2 rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-xs text-destructive"
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
            class="flex items-center gap-2 rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-xs text-destructive"
        >
            <AlertCircle class="size-3.5 shrink-0" />
            <span>{deleteError}</span>
        </div>
    {/if}

    <!-- Users Table -->
    <PremiumTable
        items={users}
        columns={["Name", "Email", "Roles", "Created", "Actions"]}
        title="Academy accounts"
        description="Roles, access, and provisioning history"
        emptyTitle="No users found"
        emptyDescription="Create the first account to start provisioning Academy access."
    >
        {#snippet row(user)}
            <td class="px-5 py-3 font-medium text-foreground">
                {user.name}
            </td>
            <td class="px-5 py-3 text-muted-foreground">{user.email}</td>
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
            <td class="px-5 py-3 text-muted-foreground text-xs">
                {user.created_at}
            </td>
            <td class="px-5 py-3 text-right">
                <div class="flex items-center justify-end gap-1">
                    <Button.Root
                        variant="ghost"
                        size="icon"
                        class="size-8 text-muted-foreground hover:text-primary"
                        onclick={() => openEnroll(user)}
                        title="Enroll in course"
                    >
                        <BookOpen class="size-3.5" />
                    </Button.Root>
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
                        class="size-8 text-muted-foreground hover:text-destructive"
                        onclick={() => handleDelete(user.id, user.name)}
                    >
                        <Trash2 class="size-3.5" />
                    </Button.Root>
                </div>
            </td>
        {/snippet}
    </PremiumTable>
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
                        class="flex items-center gap-2 rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-xs text-destructive"
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

<!-- Enroll in Course Dialog -->
{#if enrollingUser && enrollDialogOpen}
    {@const u = enrollingUser!}
    <div
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
        onclick={closeEnroll}
        role="dialog"
        aria-modal="true"
    >
        <div
            class="bg-card border border-border rounded-xl shadow-xl w-full max-w-lg mx-4 max-h-[80vh] flex flex-col"
            onclick={(e: MouseEvent) => e.stopPropagation()}
        >
            <!-- Header -->
            <div
                class="flex items-center justify-between px-6 py-4 border-b border-border shrink-0"
            >
                <div class="min-w-0">
                    <h3 class="text-base font-semibold text-foreground">
                        Enroll in Course
                    </h3>
                    <p class="text-sm text-muted-foreground mt-0.5 truncate">
                        {u.name} ({u.email})
                    </p>
                </div>
                <Button.Root
                    variant="ghost"
                    size="icon-sm"
                    onclick={closeEnroll}
                    class="shrink-0 ml-3"
                >
                    <XCircle class="size-5" />
                </Button.Root>
            </div>

            <!-- Body -->
            <div class="px-6 py-4 flex flex-col gap-4 overflow-y-auto">
                {#if enrollError}
                    <div
                        class="flex items-center gap-2 rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-xs text-destructive"
                    >
                        <AlertCircle class="size-3.5 shrink-0" />
                        <span>{enrollError}</span>
                    </div>
                {/if}

                <!-- Search -->
                <div class="relative">
                    <Search
                        class="absolute left-2.5 top-1/2 -translate-y-1/2 size-3.5 text-muted-foreground"
                    />
                    <input
                        type="text"
                        placeholder="Search courses..."
                        bind:value={enrollSearch}
                        class="w-full rounded-lg border border-input bg-background pl-8 pr-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                    />
                </div>

                <!-- Course list -->
                {#if enrollCoursesLoading}
                    <div class="flex justify-center py-8">
                        <LoaderCircle
                            class="size-5 text-muted-foreground animate-spin"
                        />
                    </div>
                {:else if filteredCourses.length === 0}
                    <p class="text-sm text-muted-foreground text-center py-6">
                        {enrollSearch
                            ? "No matching courses"
                            : "No published courses available"}
                    </p>
                {:else}
                    <div class="flex flex-col gap-1 max-h-64 overflow-y-auto">
                        {#each filteredCourses as c (c.id)}
                            <button
                                class="flex items-start gap-3 rounded-lg border px-3.5 py-2.5 text-left transition-colors {selectedCourseId ===
                                c.id
                                    ? 'border-primary bg-primary/5'
                                    : 'border-border hover:border-muted-foreground/30'}"
                                onclick={() => (selectedCourseId = c.id)}
                            >
                                <div
                                    class="size-8 rounded-lg bg-primary/10 flex items-center justify-center shrink-0 mt-0.5"
                                >
                                    <GraduationCap
                                        class="size-4 text-primary"
                                    />
                                </div>
                                <div class="min-w-0">
                                    <p
                                        class="text-sm font-medium text-foreground truncate"
                                    >
                                        {c.title}
                                    </p>
                                    {#if c.description}
                                        <p
                                            class="text-xs text-muted-foreground mt-0.5 line-clamp-2"
                                        >
                                            {c.description}
                                        </p>
                                    {/if}
                                </div>
                                {#if selectedCourseId === c.id}
                                    <Check
                                        class="size-4 text-primary shrink-0 mt-1"
                                    />
                                {/if}
                            </button>
                        {/each}
                    </div>
                {/if}
            </div>

            <!-- Footer -->
            <div
                class="flex items-center justify-end gap-2 px-6 py-4 border-t border-border shrink-0"
            >
                <Button.Root
                    variant="ghost"
                    size="sm"
                    onclick={closeEnroll}
                    disabled={enrollSaving}
                >
                    Cancel
                </Button.Root>
                <Button.Root
                    size="sm"
                    onclick={handleEnroll}
                    disabled={!selectedCourseId || enrollSaving}
                >
                    {enrollSaving ? "Enrolling…" : "Enroll"}
                </Button.Root>
            </div>
        </div>
    </div>
{/if}
