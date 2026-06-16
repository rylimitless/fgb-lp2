<script lang="ts">
	import { AlertTriangle, CheckCircle, Info, X, XCircle } from "@lucide/svelte";
	import { dismissToast, toasts, type ToastMessage } from "./toast.svelte";

	const style: Record<NonNullable<ToastMessage["variant"]>, { icon: any; cls: string; dot: string }> = {
		default: { icon: Info, cls: "border-border bg-card text-foreground", dot: "text-info" },
		success: { icon: CheckCircle, cls: "border-success/30 bg-success/10 text-success", dot: "text-success" },
		warning: { icon: AlertTriangle, cls: "border-warning/30 bg-warning/10 text-warning", dot: "text-warning" },
		error: { icon: XCircle, cls: "border-destructive/30 bg-destructive/10 text-destructive", dot: "text-destructive" },
	};
</script>

<div class="pointer-events-none fixed bottom-4 right-4 z-[120] flex w-[min(24rem,calc(100vw-2rem))] flex-col gap-2">
	{#each toasts as toast (toast.id)}
		{@const s = style[toast.variant ?? "default"]}
		<div
			class={`pointer-events-auto flex items-start gap-3 rounded-2xl border p-3 shadow-lg backdrop-blur motion-rise-in ${s.cls}`}
			role="status"
			aria-live="polite"
		>
			<s.icon class={`mt-0.5 size-4 shrink-0 ${s.dot}`} />
			<div class="min-w-0 flex-1">
				{#if toast.title}
					<p class="text-sm font-semibold text-foreground">{toast.title}</p>
				{/if}
				<p class="text-xs leading-relaxed text-muted-foreground">{toast.message}</p>
			</div>
			<button
				type="button"
				class="flex size-7 shrink-0 items-center justify-center rounded-full text-muted-foreground hover:bg-muted hover:text-foreground"
				aria-label="Dismiss notification"
				onclick={() => dismissToast(toast.id)}
			>
				<X class="size-3.5" />
			</button>
		</div>
	{/each}
</div>
