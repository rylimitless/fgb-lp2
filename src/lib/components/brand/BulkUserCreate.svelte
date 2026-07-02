<script lang="ts">
    import {
        XCircle,
        Upload,
        Trash2,
        AlertCircle,
        LoaderCircle,
        Check,
        Copy,
        Users,
        FileUp,
        MailCheck,
        MailX,
    } from "@lucide/svelte";
    import * as Button from "$lib/components/ui/button";
    import * as Input from "$lib/components/ui/input";
    import { showToast } from "$lib/components/brand";

    type InputMode = "paste" | "manual" | "csv";

    type ParsedUser = {
        name: string;
        email: string;
        roles: string[];
        error?: string;
    };

    type CreatedUser = {
        id: number;
        name: string;
        email: string;
        roles: string[];
        email_sent: boolean;
    };

    const ALL_ROLES = [
        { id: "end user", label: "End User" },
        { id: "content creator", label: "Content Creator" },
        { id: "approver", label: "Approver" },
        { id: "admin", label: "Admin" },
        { id: "manager", label: "Manager" },
        { id: "auditor", label: "Auditor" },
    ];

    let {
        open = $bindable(false),
        onCreated,
    }: {
        open: boolean;
        onCreated?: () => void;
    } = $props();

    // ---- Paste mode ----
    let mode = $state<InputMode>("paste");
    let pasteText = $state("");
    let parsedUsers = $state<ParsedUser[]>([]);
    let parseError = $state("");

    // ---- Manual mode ----
    let manualRows = $state<{ name: string; email: string; roles: string[] }[]>(
        [],
    );

    // ---- CSV upload mode ----
    let csvFile = $state<File | null>(null);
    let csvPreview = $state<ParsedUser[]>([]);
    let csvParsing = $state(false);
    let csvParseError = $state("");

    // ---- Submission ----
    let submitting = $state(false);
    let submitError = $state("");

    // ---- Results ----
    let resultsOpen = $state(false);
    let createdUsers = $state<CreatedUser[]>([]);
    let resultsCopied = $state(false);

    function reset() {
        mode = "paste";
        pasteText = "";
        parsedUsers = [];
        parseError = "";
        manualRows = [];
        csvFile = null;
        csvPreview = [];
        csvParsing = false;
        csvParseError = "";
        submitting = false;
        submitError = "";
        resultsOpen = false;
        createdUsers = [];
        resultsCopied = false;
    }

    function close() {
        open = false;
        reset();
    }

    function toggleRole(roles: string[], role: string): string[] {
        if (roles.includes(role)) return roles.filter((r) => r !== role);
        return [...roles, role];
    }

    // ---- Parse pasted text ----
    function parsePasted() {
        parseError = "";
        parsedUsers = [];

        if (!pasteText.trim()) {
            parseError = "Paste user data to continue.";
            return;
        }

        const lines = pasteText
            .trim()
            .split("\n")
            .map((l) => l.trim())
            .filter((l) => l.length > 0);

        const results: ParsedUser[] = [];

        for (let i = 0; i < lines.length; i++) {
            const line = lines[i];
            // Split by comma or tab
            const parts = line
                .split(/[,\t]+/)
                .map((p) => p.trim())
                .filter((p) => p.length > 0);

            if (parts.length < 2) {
                results.push({
                    name: "",
                    email: "",
                    roles: ["end user"],
                    error: `Line ${i + 1}: Expected "Name, Email" format (got: "${line}")`,
                });
                continue;
            }

            const name = parts[0];
            const email = parts[1];
            let roles: string[] = ["end user"];

            if (parts.length >= 3) {
                // Roles can be semicolon-separated: "Content Creator;Approver"
                const rawRoles = parts.slice(2).join(",");
                roles = rawRoles
                    .split(/[;]+/)
                    .map((r) => r.trim().toLowerCase())
                    .filter((r) => ALL_ROLES.some((ar) => ar.id === r));
                if (roles.length === 0) roles = ["end user"];
            }

            // Validate email
            if (!email.includes("@") || !email.includes(".")) {
                results.push({
                    name,
                    email,
                    roles,
                    error: `Line ${i + 1}: Invalid email "${email}"`,
                });
                continue;
            }

            results.push({ name, email, roles });
        }

        parsedUsers = results;

        if (results.length > 200) {
            parseError =
                "Maximum 200 users per batch. Please split into multiple uploads.";
        }
    }

    // ---- Manual row management ----
    function addManualRow() {
        manualRows = [
            ...manualRows,
            { name: "", email: "", roles: ["end user"] },
        ];
    }

    function removeManualRow(idx: number) {
        manualRows = manualRows.filter((_, i) => i !== idx);
    }

    // ---- CSV file handling ----
    async function handleCsvFile(e: Event) {
        const input = e.target as HTMLInputElement;
        const file = input.files?.[0];
        if (!file) return;
        csvFile = file;
        csvParseError = "";
        csvParsing = true;
        csvPreview = [];

        try {
            const text = await file.text();
            const lines = text
                .split("\n")
                .map((l) => l.trim())
                .filter((l) => l.length > 0);

            if (lines.length < 2) {
                csvParseError =
                    "CSV must have a header row and at least one data row.";
                csvParsing = false;
                return;
            }

            // Parse header
            const headers = lines[0]
                .split(",")
                .map((h) => h.trim().toLowerCase());
            const nameIdx = headers.indexOf("name");
            const emailIdx = headers.indexOf("email");
            const rolesIdx = headers.indexOf("roles");

            if (nameIdx === -1 || emailIdx === -1) {
                csvParseError =
                    "CSV must have 'Name' and 'Email' columns. Found: " +
                    headers.join(", ");
                csvParsing = false;
                return;
            }

            const results: ParsedUser[] = [];

            for (let i = 1; i < lines.length; i++) {
                // Simple CSV parsing (handles basic quoting)
                const parts = parseCSVLine(lines[i]);
                if (parts.length <= Math.max(nameIdx, emailIdx)) continue;

                const name = parts[nameIdx]?.trim() ?? "";
                const email = parts[emailIdx]?.trim() ?? "";

                if (!name || !email) continue;

                let roles: string[] = ["end user"];
                if (rolesIdx >= 0 && rolesIdx < parts.length) {
                    const raw = parts[rolesIdx]?.trim() ?? "";
                    if (raw) {
                        const parsed = raw
                            .split(";")
                            .map((r) => r.trim().toLowerCase())
                            .filter((r) => ALL_ROLES.some((ar) => ar.id === r));
                        if (parsed.length > 0) roles = parsed;
                    }
                }

                if (!email.includes("@") || !email.includes(".")) {
                    results.push({
                        name,
                        email,
                        roles,
                        error: `Row ${i + 1}: Invalid email "${email}"`,
                    });
                    continue;
                }

                results.push({ name, email, roles });
            }

            csvPreview = results;
        } catch {
            csvParseError =
                "Failed to read CSV file. Make sure it's a valid text file.";
        } finally {
            csvParsing = false;
        }
    }

    // Simple CSV line parser that handles quoted fields
    function parseCSVLine(line: string): string[] {
        const result: string[] = [];
        let current = "";
        let inQuotes = false;

        for (let i = 0; i < line.length; i++) {
            const ch = line[i];
            if (inQuotes) {
                if (ch === '"') {
                    if (i + 1 < line.length && line[i + 1] === '"') {
                        current += '"';
                        i++;
                    } else {
                        inQuotes = false;
                    }
                } else {
                    current += ch;
                }
            } else {
                if (ch === '"') {
                    inQuotes = true;
                } else if (ch === ",") {
                    result.push(current);
                    current = "";
                } else {
                    current += ch;
                }
            }
        }
        result.push(current);
        return result;
    }

    // ---- Get users to submit ----
    function usersToSubmit(): {
        name: string;
        email: string;
        roles: string[];
    }[] {
        if (mode === "paste") {
            return parsedUsers
                .filter((u) => !u.error)
                .map((u) => ({
                    name: u.name,
                    email: u.email,
                    roles: u.roles,
                }));
        }
        if (mode === "csv") {
            return csvPreview
                .filter((u) => !u.error)
                .map((u) => ({
                    name: u.name,
                    email: u.email,
                    roles: u.roles,
                }));
        }
        return manualRows.filter((r) => r.name && r.email);
    }

    let validUsersCount = $derived(usersToSubmit().length);
    let errorUsersCount = $derived(
        mode === "paste"
            ? parsedUsers.filter((u) => u.error).length
            : mode === "csv"
              ? csvPreview.filter((u) => u.error).length
              : 0,
    );
    let canSubmit = $derived(validUsersCount > 0 && !submitting);

    // ---- Submit ----
    async function handleSubmit() {
        submitError = "";
        submitting = true;
        try {
            if (mode === "csv" && csvFile) {
                // Use multipart form upload
                const formData = new FormData();
                formData.append("file", csvFile);
                const res = await fetch("/api/admin/users/bulk/csv", {
                    method: "POST",
                    credentials: "include",
                    body: formData,
                });

                if (!res.ok) {
                    const err = await res.json();
                    submitError = err.error ?? "Failed to create users";
                    return;
                }

                const result = await res.json();
                createdUsers = result.users ?? [];
                resultsOpen = true;

                showToast(
                    `${result.created_count} user(s) created. ${result.emails_failed > 0 ? result.emails_failed + " welcome email(s) failed to send." : "Welcome emails sent."}`,
                    {
                        title: "Users created",
                        variant:
                            result.emails_failed > 0 ? "default" : "success",
                    },
                );

                onCreated?.();
            } else {
                // JSON body for paste/manual
                const users = usersToSubmit();
                const res = await fetch("/api/admin/users/bulk", {
                    method: "POST",
                    credentials: "include",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify({ users }),
                });

                if (!res.ok) {
                    const err = await res.json();
                    submitError = err.error ?? "Failed to create users";
                    return;
                }

                const result = await res.json();
                createdUsers = result.users ?? [];
                resultsOpen = true;

                showToast(
                    `${result.created_count} user(s) created. ${result.emails_failed > 0 ? result.emails_failed + " welcome email(s) failed to send." : "Welcome emails sent."}`,
                    {
                        title: "Users created",
                        variant:
                            result.emails_failed > 0 ? "default" : "success",
                    },
                );

                onCreated?.();
            }
        } catch {
            submitError = "Network error";
        } finally {
            submitting = false;
        }
    }

    // ---- Copy results ----
    function copyResults() {
        const header = "Name\tEmail\tRoles";
        const rows = createdUsers.map(
            (u) => `${u.name}\t${u.email}\t${u.roles?.join("; ") ?? ""}`,
        );
        navigator.clipboard.writeText([header, ...rows].join("\n"));
        resultsCopied = true;
        setTimeout(() => (resultsCopied = false), 2000);
    }
</script>

{#if open}
    <div
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
        role="dialog"
        aria-modal="true"
        onkeydown={(e) => {
            if (e.key === "Escape") close();
        }}
    >
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div class="fixed inset-0" onclick={close}></div>

        <div
            class="relative z-10 w-full max-w-2xl max-h-[90vh] overflow-hidden rounded-2xl border border-border bg-card shadow-2xl flex flex-col"
        >
            <!-- Results mode -->
            {#if resultsOpen}
                <div
                    class="flex items-center justify-between border-b border-border px-6 py-4"
                >
                    <div class="min-w-0">
                        <h3 class="text-base font-semibold text-foreground">
                            Users created
                        </h3>
                        <p class="text-sm text-muted-foreground mt-0.5">
                            {createdUsers.length} account(s) created.
                            {#if createdUsers.every((u) => u.email_sent)}
                                Welcome emails sent successfully.
                            {:else if createdUsers.some((u) => u.email_sent)}
                                Some welcome emails failed — check below.
                            {:else}
                                No welcome emails were sent (mailer not
                                configured).
                            {/if}
                        </p>
                    </div>
                    <div class="flex items-center gap-2">
                        <Button.Root
                            variant="outline"
                            size="sm"
                            onclick={copyResults}
                        >
                            {#if resultsCopied}
                                <Check class="size-3.5" />
                                Copied
                            {:else}
                                <Copy class="size-3.5" />
                                Copy all
                            {/if}
                        </Button.Root>
                        <Button.Root
                            variant="ghost"
                            size="icon"
                            class="size-8 text-muted-foreground hover:text-foreground"
                            onclick={close}
                        >
                            <XCircle class="size-5" />
                        </Button.Root>
                    </div>
                </div>
                <div class="overflow-y-auto px-6 py-4">
                    <table class="w-full text-sm">
                        <thead>
                            <tr
                                class="border-b border-border text-left text-xs text-muted-foreground"
                            >
                                <th class="py-2 pr-4 font-medium">Name</th>
                                <th class="py-2 pr-4 font-medium">Email</th>
                                <th class="py-2 pr-4 font-medium">Roles</th>
                                <th class="py-2 font-medium">Email</th>
                            </tr>
                        </thead>
                        <tbody>
                            {#each createdUsers as u}
                                <tr class="border-b border-border/40">
                                    <td
                                        class="py-2.5 pr-4 font-medium text-foreground"
                                    >
                                        {u.name}
                                    </td>
                                    <td
                                        class="py-2.5 pr-4 text-muted-foreground"
                                    >
                                        {u.email}
                                    </td>
                                    <td class="py-2.5">
                                        <div class="flex flex-wrap gap-1">
                                            {#each u.roles ?? ["end user"] as r}
                                                <span
                                                    class="inline-flex items-center rounded border border-border px-1.5 py-0.5 text-[10px] text-muted-foreground"
                                                >
                                                    {r}
                                                </span>
                                            {/each}
                                        </div>
                                    </td>
                                    <td class="py-2.5">
                                        {#if u.email_sent}
                                            <span
                                                class="inline-flex items-center gap-1 text-xs text-success"
                                            >
                                                <MailCheck class="size-3" />
                                                Sent
                                            </span>
                                        {:else}
                                            <span
                                                class="inline-flex items-center gap-1 text-xs text-destructive"
                                            >
                                                <MailX class="size-3" />
                                                Failed
                                            </span>
                                        {/if}
                                    </td>
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                </div>
            {:else}
                <!-- Input mode -->
                <div
                    class="flex items-center justify-between border-b border-border px-6 py-4"
                >
                    <div class="min-w-0">
                        <h3 class="text-base font-semibold text-foreground">
                            Add users
                        </h3>
                        <p class="text-sm text-muted-foreground mt-0.5">
                            Create multiple accounts at once. Welcome emails
                            with temporary passwords will be sent automatically.
                        </p>
                    </div>
                    <Button.Root
                        variant="ghost"
                        size="icon"
                        class="size-8 text-muted-foreground hover:text-foreground"
                        onclick={close}
                    >
                        <XCircle class="size-5" />
                    </Button.Root>
                </div>

                <!-- Mode tabs -->
                <div class="flex border-b border-border px-6">
                    <button
                        class="px-3 py-2.5 text-sm font-medium border-b-2 transition-colors {mode ===
                        'paste'
                            ? 'border-primary text-primary'
                            : 'border-transparent text-muted-foreground hover:text-foreground'}"
                        onclick={() => (mode = "paste")}
                    >
                        Paste list
                    </button>
                    <button
                        class="px-3 py-2.5 text-sm font-medium border-b-2 transition-colors {mode ===
                        'manual'
                            ? 'border-primary text-primary'
                            : 'border-transparent text-muted-foreground hover:text-foreground'}"
                        onclick={() => (mode = "manual")}
                    >
                        Add manually
                    </button>
                    <button
                        class="px-3 py-2.5 text-sm font-medium border-b-2 transition-colors {mode ===
                        'csv'
                            ? 'border-primary text-primary'
                            : 'border-transparent text-muted-foreground hover:text-foreground'}"
                        onclick={() => (mode = "csv")}
                    >
                        Upload CSV
                    </button>
                </div>

                <div class="overflow-y-auto px-6 py-4 flex flex-col gap-4">
                    {#if submitError}
                        <div
                            class="flex items-center gap-2 rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-xs text-destructive"
                        >
                            <AlertCircle class="size-3.5 shrink-0" />
                            <span>{submitError}</span>
                        </div>
                    {/if}

                    {#if mode === "paste"}
                        <div class="flex flex-col gap-2">
                            <label
                                class="text-xs font-medium text-foreground"
                                for="paste-input"
                            >
                                Paste users
                            </label>
                            <p class="text-xs text-muted-foreground">
                                One per line: <code>Name, Email, Role</code>.
                                Role defaults to "End User" if omitted. For
                                multiple roles, separate with semicolons:
                                <code>Content Creator;Approver</code>.
                            </p>
                            <textarea
                                id="paste-input"
                                class="w-full min-h-[160px] rounded-lg border border-border bg-background px-3 py-2.5 text-sm text-foreground placeholder:text-muted-foreground resize-y focus:outline-none focus:ring-2 focus:ring-primary/30"
                                placeholder="John Doe, john@company.com&#10;Jane Smith, jane@company.com, Content Creator&#10;Bob Wilson, bob@company.com, Content Creator;Approver"
                                bind:value={pasteText}></textarea>
                            <div class="flex justify-between items-center">
                                <Button.Root
                                    variant="outline"
                                    size="sm"
                                    onclick={parsePasted}
                                >
                                    <Upload class="size-3.5" />
                                    Parse
                                </Button.Root>
                                {#if parseError}
                                    <span class="text-xs text-destructive"
                                        >{parseError}</span
                                    >
                                {/if}
                            </div>
                        </div>

                        {#if parsedUsers.length > 0}
                            <div class="flex flex-col gap-2">
                                <div
                                    class="flex items-center justify-between text-xs"
                                >
                                    <span class="text-muted-foreground">
                                        {validUsersCount} valid ·
                                        {errorUsersCount} with errors
                                    </span>
                                </div>
                                <div
                                    class="rounded-lg border border-border overflow-hidden"
                                >
                                    <table class="w-full text-sm">
                                        <thead>
                                            <tr
                                                class="bg-muted/30 text-left text-xs text-muted-foreground"
                                            >
                                                <th
                                                    class="px-3 py-2 font-medium"
                                                >
                                                    Name
                                                </th>
                                                <th
                                                    class="px-3 py-2 font-medium"
                                                >
                                                    Email
                                                </th>
                                                <th
                                                    class="px-3 py-2 font-medium"
                                                >
                                                    Roles
                                                </th>
                                            </tr>
                                        </thead>
                                        <tbody>
                                            {#each parsedUsers as u, i}
                                                <tr
                                                    class="border-t border-border/40 {u.error
                                                        ? 'bg-destructive/5'
                                                        : ''}"
                                                >
                                                    <td
                                                        class="px-3 py-2 text-xs text-foreground"
                                                    >
                                                        {u.name || "—"}
                                                    </td>
                                                    <td
                                                        class="px-3 py-2 text-xs {u.error
                                                            ? 'text-destructive'
                                                            : 'text-muted-foreground'}"
                                                    >
                                                        {u.email || "—"}
                                                        {#if u.error}
                                                            <span
                                                                class="block text-[10px] text-destructive mt-0.5"
                                                            >
                                                                {u.error}
                                                            </span>
                                                        {/if}
                                                    </td>
                                                    <td class="px-3 py-2">
                                                        <div
                                                            class="flex flex-wrap gap-0.5"
                                                        >
                                                            {#each u.roles as r}
                                                                <span
                                                                    class="inline-flex items-center rounded border border-border px-1 py-0.5 text-[10px] text-muted-foreground"
                                                                >
                                                                    {r}
                                                                </span>
                                                            {/each}
                                                        </div>
                                                    </td>
                                                </tr>
                                            {/each}
                                        </tbody>
                                    </table>
                                </div>
                            </div>
                        {/if}
                    {:else if mode === "csv"}
                        <!-- CSV upload mode -->
                        <div class="flex flex-col gap-2">
                            <label class="text-xs font-medium text-foreground">
                                Upload CSV file
                            </label>
                            <p class="text-xs text-muted-foreground">
                                Upload a CSV with columns:
                                <code>Name, Email</code> (required) and
                                <code>Roles</code> (optional). Roles should be
                                semicolon-separated:
                                <code>Content Creator;Approver</code>.
                            </p>

                            <label
                                class="flex flex-col items-center justify-center gap-3 rounded-lg border-2 border-dashed border-muted-foreground/30 bg-muted/20 p-8 cursor-pointer hover:border-primary/40 hover:bg-primary/5 transition-colors {csvFile
                                    ? 'border-primary/50 bg-primary/5'
                                    : ''}"
                            >
                                {#if csvFile}
                                    <FileUp class="size-8 text-primary" />
                                    <span
                                        class="text-sm font-medium text-foreground"
                                    >
                                        {csvFile.name}
                                    </span>
                                    <span class="text-xs text-muted-foreground">
                                        Click to change file
                                    </span>
                                {:else}
                                    <Upload
                                        class="size-8 text-muted-foreground"
                                    />
                                    <span class="text-sm text-muted-foreground">
                                        Drop a CSV file here or click to browse
                                    </span>
                                {/if}
                                <input
                                    type="file"
                                    accept=".csv,text/csv"
                                    class="hidden"
                                    onchange={handleCsvFile}
                                />
                            </label>

                            {#if csvParsing}
                                <div
                                    class="flex items-center justify-center py-4"
                                >
                                    <LoaderCircle
                                        class="size-4 text-muted-foreground animate-spin"
                                    />
                                    <span
                                        class="ml-2 text-sm text-muted-foreground"
                                        >Parsing CSV…</span
                                    >
                                </div>
                            {:else if csvParseError}
                                <div
                                    class="flex items-center gap-2 rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-xs text-destructive"
                                >
                                    <AlertCircle class="size-3.5 shrink-0" />
                                    <span>{csvParseError}</span>
                                </div>
                            {/if}

                            {#if csvPreview.length > 0}
                                <div class="flex flex-col gap-2">
                                    <div
                                        class="flex items-center justify-between text-xs"
                                    >
                                        <span class="text-muted-foreground">
                                            {validUsersCount} valid ·
                                            {errorUsersCount} with errors
                                        </span>
                                    </div>
                                    <div
                                        class="rounded-lg border border-border overflow-hidden"
                                    >
                                        <table class="w-full text-sm">
                                            <thead>
                                                <tr
                                                    class="bg-muted/30 text-left text-xs text-muted-foreground"
                                                >
                                                    <th
                                                        class="px-3 py-2 font-medium"
                                                    >
                                                        Name
                                                    </th>
                                                    <th
                                                        class="px-3 py-2 font-medium"
                                                    >
                                                        Email
                                                    </th>
                                                    <th
                                                        class="px-3 py-2 font-medium"
                                                    >
                                                        Roles
                                                    </th>
                                                </tr>
                                            </thead>
                                            <tbody>
                                                {#each csvPreview as u, i}
                                                    <tr
                                                        class="border-t border-border/40 {u.error
                                                            ? 'bg-destructive/5'
                                                            : ''}"
                                                    >
                                                        <td
                                                            class="px-3 py-2 text-xs text-foreground"
                                                        >
                                                            {u.name || "—"}
                                                        </td>
                                                        <td
                                                            class="px-3 py-2 text-xs {u.error
                                                                ? 'text-destructive'
                                                                : 'text-muted-foreground'}"
                                                        >
                                                            {u.email || "—"}
                                                            {#if u.error}
                                                                <span
                                                                    class="block text-[10px] text-destructive mt-0.5"
                                                                >
                                                                    {u.error}
                                                                </span>
                                                            {/if}
                                                        </td>
                                                        <td class="px-3 py-2">
                                                            <div
                                                                class="flex flex-wrap gap-0.5"
                                                            >
                                                                {#each u.roles as r}
                                                                    <span
                                                                        class="inline-flex items-center rounded border border-border px-1 py-0.5 text-[10px] text-muted-foreground"
                                                                    >
                                                                        {r}
                                                                    </span>
                                                                {/each}
                                                            </div>
                                                        </td>
                                                    </tr>
                                                {/each}
                                            </tbody>
                                        </table>
                                    </div>
                                </div>
                            {/if}
                        </div>
                    {:else}
                        <!-- Manual mode -->
                        <div class="flex flex-col gap-3">
                            {#each manualRows as row, i}
                                <div
                                    class="flex items-start gap-2 p-3 rounded-lg border border-border bg-background"
                                >
                                    <div
                                        class="flex-1 grid grid-cols-1 sm:grid-cols-2 gap-2"
                                    >
                                        <Input.Root
                                            type="text"
                                            placeholder="Name"
                                            bind:value={row.name}
                                        />
                                        <Input.Root
                                            type="email"
                                            placeholder="Email"
                                            bind:value={row.email}
                                        />
                                    </div>
                                    <div
                                        class="flex flex-wrap gap-1 mt-1 sm:mt-0 sm:col-span-2"
                                    >
                                        {#each ALL_ROLES as r}
                                            <label
                                                class="flex items-center gap-1 rounded border px-1.5 py-0.5 text-[10px] cursor-pointer hover:bg-muted/30 transition-colors {row.roles.includes(
                                                    r.id,
                                                )
                                                    ? 'border-primary bg-primary/5 text-primary'
                                                    : 'border-border text-muted-foreground'}"
                                            >
                                                <input
                                                    type="checkbox"
                                                    class="sr-only"
                                                    checked={row.roles.includes(
                                                        r.id,
                                                    )}
                                                    onchange={() => {
                                                        row.roles =
                                                            row.roles.includes(
                                                                r.id,
                                                            )
                                                                ? row.roles.filter(
                                                                      (x) =>
                                                                          x !==
                                                                          r.id,
                                                                  )
                                                                : [
                                                                      ...row.roles,
                                                                      r.id,
                                                                  ];
                                                        manualRows = manualRows;
                                                    }}
                                                />
                                                {r.label}
                                            </label>
                                        {/each}
                                    </div>
                                    <Button.Root
                                        variant="ghost"
                                        size="icon"
                                        class="size-7 text-muted-foreground hover:text-destructive shrink-0"
                                        onclick={() => removeManualRow(i)}
                                    >
                                        <Trash2 class="size-3.5" />
                                    </Button.Root>
                                </div>
                            {/each}

                            <Button.Root
                                variant="outline"
                                size="sm"
                                class="self-start"
                                onclick={addManualRow}
                            >
                                + Add row
                            </Button.Root>
                        </div>
                    {/if}
                </div>

                <!-- Footer -->
                <div
                    class="flex items-center justify-between border-t border-border px-6 py-4"
                >
                    <span class="text-xs text-muted-foreground">
                        {validUsersCount} user(s) ready to create
                    </span>
                    <div class="flex items-center gap-2">
                        <Button.Root
                            variant="outline"
                            size="sm"
                            onclick={close}
                        >
                            Cancel
                        </Button.Root>
                        <Button.Root
                            size="sm"
                            disabled={!canSubmit}
                            onclick={handleSubmit}
                        >
                            {#if submitting}
                                <LoaderCircle class="size-3.5 animate-spin" />
                                Creating…
                            {:else}
                                <Users class="size-3.5" />
                                Create {validUsersCount}
                                user{validUsersCount !== 1 ? "s" : ""}
                            {/if}
                        </Button.Root>
                    </div>
                </div>
            {/if}
        </div>
    </div>
{/if}
