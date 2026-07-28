<script lang="ts">
    import {
        ClipboardList,
        Search,
        ChevronLeft,
        ChevronRight,
        ChevronDown,
        Filter,
    } from "@lucide/svelte";
    import { goto } from "$app/navigation";
    import { PageHeader, PremiumTable, DotPattern } from "$lib/components/brand";
    import * as Button from "$lib/components/ui/button";
    import * as DropdownMenu from "$lib/components/ui/dropdown-menu";

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
    let selectedActionLabel = $derived(
        ACTION_OPTIONS.find((o) => o.value === selectedAction)?.label ??
            "All actions",
    );

    function gotoPage(offset: number) {
        const params = new URLSearchParams();
        params.set("limit", "50");
        params.set("offset", String(offset));
        if (selectedAction) params.set("action", selectedAction);
        goto(`/audit-log?${params.toString()}`);
    }

    function applyFilter(value?: string) {
        if (value !== undefined) selectedAction = value;
        const params = new URLSearchParams();
        params.set("limit", "50");
        params.set("offset", "0");
        if (selectedAction) params.set("action", selectedAction);
        goto(`/audit-log?${params.toString()}`);
    }

    let currentOffset = $derived(parseInt(filters.offset) || 0);

    // Action -> semantic token mapping. Collapses the previous 7-color rainbow
    // into 4 brand-semantic categories: info (sign-in / coach activity),
    // success (creation / approval), warning (review / pending state),
    // destructive (failure / deletion). See BRAND.md §4.3.
    function actionBadge(action: string): string {
        if (action.includes("failed") || action.includes("deleted"))
            return "bg-destructive/10 text-destructive";
        if (action.includes("created") || action.includes("uploaded") || action.includes("approved"))
            return "bg-success/10 text-success";
        if (action.includes("reviewed") || action.includes("resubmitted"))
            return "bg-warning/10 text-warning";
        if (action.startsWith("login") || action.startsWith("logout") || action.startsWith("gia") || action.includes("user_") || action.includes("admin"))
            return "bg-info/10 text-info";
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
    <PageHeader
        title="Audit log"
        eyebrow="Governance"
        description="Every action across the Academy, timestamped and attributed. Filter, search, and paginate. Read-only by design."
    >
        {#snippet icon()}
            <ClipboardList class="size-6 text-primary" />
        {/snippet}
    </PageHeader>

    <!-- Filters -->
    <div
        class="relative flex flex-wrap items-center gap-3 rounded-2xl border border-border-strong bg-card p-3 overflow-hidden"
    >
        <DotPattern gap={22} radius={1} class="opacity-60" />
        <div class="relative flex-1 min-w-[200px] max-w-xs">
            <Search
                class="absolute left-2.5 top-1/2 -translate-y-1/2 size-3.5 text-muted-foreground"
            />
            <input
                type="text"
                placeholder="Search…"
                aria-label="Search audit log"
                bind:value={searchQuery}
                class="w-full pl-8 pr-3 py-1.5 text-sm border border-border-strong rounded-lg bg-background text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring/30"
            />
        </div>
        <DropdownMenu.Root>
            <DropdownMenu.Trigger>
                {#snippet child({ props })}
                    <button
                        {...props}
                        aria-label="Filter by action"
                        class="inline-flex items-center gap-2 text-sm border border-border-strong rounded-lg bg-background text-foreground px-3 py-1.5 outline-none focus-visible:ring-2 focus-visible:ring-ring/30"
                    >
                        <Filter class="size-3.5 text-muted-foreground" />
                        <span>{selectedActionLabel}</span>
                        <ChevronDown class="size-3.5 text-muted-foreground" />
                    </button>
                {/snippet}
            </DropdownMenu.Trigger>
            <DropdownMenu.Content class="w-56 max-h-[60vh] overflow-y-auto" align="start">
                <DropdownMenu.Label>Filter by action</DropdownMenu.Label>
                <DropdownMenu.Separator />
                <DropdownMenu.RadioGroup
                    value={selectedAction}
                    onValueChange={(v) => applyFilter(v)}
                >
                    {#each ACTION_OPTIONS as opt (opt.value)}
                        <DropdownMenu.RadioItem value={opt.value}>
                            {opt.label}
                        </DropdownMenu.RadioItem>
                    {/each}
                </DropdownMenu.RadioGroup>
            </DropdownMenu.Content>
        </DropdownMenu.Root>
    </div>

    <!-- Table -->
    <div class="rounded-2xl border border-border bg-card overflow-hidden lift">
        <PremiumTable
            items={filtered}
            columns={["Time", "User", "Action", "Details"]}
            title="Audit events"
            description="Searchable operational record"
            emptyTitle="No audit events found"
            emptyDescription="Try a different action filter or clear the search to see the full log."
        >
            {#snippet row(entry)}
                <td class="px-5 py-3 text-xs text-muted-foreground whitespace-nowrap">
                    {new Date(entry.created_at).toLocaleString()}
                </td>
                <td class="px-5 py-3 text-xs whitespace-nowrap">
                    {#if entry.user_name}
                        <span class="text-foreground">{entry.user_name}</span>
                        <span class="text-muted-foreground ml-1">({entry.user_role})</span>
                    {:else}
                        <span class="text-muted-foreground">System</span>
                    {/if}
                </td>
                <td class="px-5 py-3 whitespace-nowrap">
                    <span class="inline-block px-1.5 py-0.5 rounded text-xs font-medium {actionBadge(entry.action)}">
                        {entry.action}
                    </span>
                </td>
                <td class="px-5 py-3 text-xs text-muted-foreground max-w-md truncate">
                    {formatDetails(entry)}
                </td>
            {/snippet}
        </PremiumTable>

        {#if filtered.length > 0}
            <!-- Pagination -->
            <div
                class="flex items-center justify-between px-4 py-3 border-t border-border"
            >
                <span class="text-xs text-muted-foreground">
                    Showing {currentOffset + 1}–{currentOffset +
                        filtered.length}
                </span>
                <div class="flex items-center gap-2">
                    <Button.Root
                        variant="outline"
                        size="sm"
                        disabled={currentOffset === 0}
                        onclick={() => gotoPage(Math.max(0, currentOffset - 50))}
                    >
                        <ChevronLeft class="size-3.5" />
                        Previous
                    </Button.Root>
                    <Button.Root
                        variant="outline"
                        size="sm"
                        onclick={() => gotoPage(currentOffset + 50)}
                    >
                        Next
                        <ChevronRight class="size-3.5" />
                    </Button.Root>
                </div>
            </div>
        {/if}
    </div>
</div>
