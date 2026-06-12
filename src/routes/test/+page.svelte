<!-- fgb/src/routes/test/+page.svelte -->
<script lang="ts">
    import { Button } from "$lib/components/ui/button/index.js";
    let { data } = $props();
</script>

<Button variant="outline" class="text-sm p-4">Button {data.message}</Button>
