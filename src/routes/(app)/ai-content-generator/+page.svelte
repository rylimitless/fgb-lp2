<script lang="ts">
    import { LoaderCircle } from "@lucide/svelte";
    // This page redirects to /course-builder via +page.server.ts.
    // This component never renders — the redirect fires during load — but
    // SvelteKit requires a +page.svelte to exist. Showing a spinner in case
    // the redirect is slow.
</script>

<div class="flex items-center justify-center py-24">
    <LoaderCircle class="size-5 text-muted-foreground animate-spin" />
</div>
