<script lang="ts">
    import {
        ClipboardList,
        Search,
        ChevronLeft,
        ChevronRight,
    } from "@lucide/svelte";
    import { goto } from "$app/navigation";

    let { data } = $props();
    let auditLog = $derived(data.auditLog ?? []);
    let filters = $derived(data.filters ?? { limit: "50", offset: "0", action: "" });

    const ACTION_OPTIONS = [
        { value: "", label: "All actions" },
        { value: "login", label: "Login" },
        { value: "login_failed", label: "Login Failed" },
        { value: "logout", label: "Logout" },
        { value: "admin_setup", label: "Admin Setup" },
        { value: "gia_query", label: "GIA Query" },
        { value: "document_uploaded", label: "Document Uploaded" },
        { value: "document_deleted", label: "Document Deleted" },
        { value: "document_approved", label: "Document Approved" },
        { value: "document_reviewed", label: "Document Reviewed" },
        { value: "document_resubmitted", label: "Document Resubmitted" },
        { value: "course_created", label: "Course Created" },
        { value: "course_deleted", label: "Course Deleted" },
        { value: "course_reviewed", label: "Course Reviewed" },
        { value: "course_resubmitted", label: "Course Resubmitted" },
        { value: "user_created", label: "User Created" },
        { value: "user_roles_updated", label: "User Roles Updated" },
        { value: "user_deleted", label: "User Deleted" },
    ];

    let selectedAction = $state(filters.action);
    let searchQuery = $state("");

    function gotoPage(offset: number) {
        const params = new URLSearchParams();
        params.set("limit", "50");
        params.set("offset", String(offset));
        if (selectedAction) params.set("action", selectedAction);
        goto(`/audit-log?${params.toString()}`);
    }

    function applyFilter() {
        const params = new URLSearchParams();
        params.set("limit", "50");
        params.set("offset", "0");
        if (selectedAction) params.set("action", selectedAction);
        goto(`/audit-log?${params.toString()}`);
    }

    let currentOffset = $derived(parseInt(filters.offset) || 0);

    function actionBadge(action: string): string {
        if (action.startsWith("login")) return "bg-blue-500/10 text-blue-500";
        if (action.includes("failed")) return "bg-red-500/10 text-red-500";
        if (action.includes("created") || action.includes("uploaded"))
            return "bg-emerald-500/10 text-emerald-500";
        if (action.includes("deleted")) return "bg-red-500/10 text-red-500";
        if (action.includes("reviewed"))
            return "bg-amber-500/10 text-amber-500";
        if (action.includes("approved"))
            return "bg-emerald-500/10 text-emerald-500";
        if (action.startsWith("gia")) return "bg-violet-500/10 text-violet-500";
        if (action.includes("user_")) return "bg-cyan-500/10 text-cyan-500";
        return "bg-muted text-muted-foreground";
    }

    function formatDetails(entry: any): string {
        const d = entry.details;
        if (!d || typeof d !== "object") return "—";
        const parts: string[] = [];
        if (d.msg) parts.push(d.msg);
        for (const [k, v] of Object.entries(d)) {
            if (k === "msg" || k === "user_id" || k === "reason") continue;
            if (typeof v === "string") parts.push(v);
            else if (typeof v === "number") parts.push(`${k}: ${v}`);
        }
        if (d.reason && d.email) return `Failed login for ${d.email}`;
        return parts.join(" · ") || "—";
    }

    let filtered = $derived(
        searchQuery
            ? auditLog.filter((e: any) => {
                  const q = searchQuery.toLowerCase();
                  return (
                      e.action.toLowerCase().includes(q) ||
                      (e.user_name && e.user_name.toLowerCase().includes(q)) ||
                      (e.user_email &&
                          e.user_email.toLowerCase().includes(q)) ||
                      formatDetails(e).toLowerCase().includes(q)
                  );
              })
            : auditLog,
    );
</script>

<svelte:head><title>Audit Log - FGB</title></svelte:head>

<div class="flex w-full max-w-7xl mx-auto flex-col gap-6">
    <div class="flex items-center gap-3">
        <ClipboardList class="size-6 text-primary" />
        <h1 class="text-2xl font-semibold text-foreground">Audit Log</h1>
    </div>

    <!-- Filters -->
    <div class="flex flex-wrap items-center gap-3">
        <div class="relative flex-1 min-w-[200px] max-w-xs">
            <Search
                class="absolute left-2.5 top-1/2 -translate-y-1/2 size-3.5 text-muted-foreground"
            />
            <input
                type="text"
                placeholder="Search…"
                bind:value={searchQuery}
                class="w-full pl-8 pr-3 py-1.5 text-sm border border-border rounded-lg bg-background text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-primary"
            />
        </div>
        <select
            bind:value={selectedAction}
            onchange={applyFilter}
            class="text-sm border border-border rounded-lg bg-background text-foreground px-3 py-1.5 focus:outline-none focus:ring-1 focus:ring-primary"
        >
            {#each ACTION_OPTIONS as opt}
                <option value={opt.value}>{opt.label}</option>
            {/each}
        </select>
    </div>

    <!-- Table -->
    <div class="rounded-xl border border-border bg-card overflow-hidden">
        {#if filtered.length === 0}
            <p class="text-sm text-muted-foreground py-12 text-center">
                No audit events found.
            </p>
        {:else}
            <div class="overflow-x-auto">
                <table class="w-full text-sm">
                    <thead>
                        <tr
                            class="text-left text-xs text-muted-foreground border-b border-border bg-muted/30"
                        >
                            <th class="py-3 pl-4 pr-3 font-medium">Time</th>
                            <th class="py-3 pr-3 font-medium">User</th>
                            <th class="py-3 pr-3 font-medium">Action</th>
                            <th class="py-3 pr-4 font-medium">Details</th>
                        </tr>
                    </thead>
                    <tbody>
                        {#each filtered as entry}
                            <tr
                                class="border-b border-border/50 hover:bg-muted/20 transition-colors"
                            >
                                <td
                                    class="py-2.5 pl-4 pr-3 text-xs text-muted-foreground whitespace-nowrap"
                                >
                                    {new Date(
                                        entry.created_at,
                                    ).toLocaleString()}
                                </td>
                                <td
                                    class="py-2.5 pr-3 text-xs whitespace-nowrap"
                                >
                                    {#if entry.user_name}
                                        <span class="text-foreground"
                                            >{entry.user_name}</span
                                        >
                                        <span
                                            class="text-muted-foreground ml-1"
                                            >({entry.user_role})</span
                                        >
                                    {:else}
                                        <span class="text-muted-foreground"
                                            >System</span
                                        >
                                    {/if}
                                </td>
                                <td class="py-2.5 pr-3 whitespace-nowrap">
                                    <span
                                        class="inline-block px-1.5 py-0.5 rounded text-xs font-medium {actionBadge(
                                            entry.action,
                                        )}"
                                    >
                                        {entry.action}
                                    </span>
                                </td>
                                <td
                                    class="py-2.5 pr-4 text-xs text-muted-foreground max-w-md truncate"
                                >
                                    {formatDetails(entry)}
                                </td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
            </div>

            <!-- Pagination -->
            <div
                class="flex items-center justify-between px-4 py-3 border-t border-border"
            >
                <span class="text-xs text-muted-foreground">
                    Showing {currentOffset + 1}–{currentOffset +
                        filtered.length}
                </span>
                <div class="flex items-center gap-2">
                    <button
                        class="flex items-center gap-1 text-xs px-2 py-1 rounded border border-border hover:bg-muted transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
                        disabled={currentOffset === 0}
                        onclick={() => gotoPage(Math.max(0, currentOffset - 50))}
                    >
                        <ChevronLeft class="size-3.5" />
                        Previous
                    </button>
                    <button
                        class="flex items-center gap-1 text-xs px-2 py-1 rounded border border-border hover:bg-muted transition-colors"
                        onclick={() => gotoPage(currentOffset + 50)}
                    >
                        Next
                        <ChevronRight class="size-3.5" />
                    </button>
                </div>
            </div>
        {/if}
    </div>
</div>
