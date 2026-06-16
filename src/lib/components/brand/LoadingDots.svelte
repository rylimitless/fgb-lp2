<script lang="ts">
	import { cn } from "$lib/utils.js";

	type Props = {
		class?: string;
		label?: string;
	};

	let { class: className, label = "Loading" }: Props = $props();
</script>

<span
	role="status"
	aria-label={label}
	class={cn("inline-flex items-center gap-1", className)}
>
	<span class="ld-dot" style="animation-delay: 0ms"></span>
	<span class="ld-dot" style="animation-delay: 140ms"></span>
	<span class="ld-dot" style="animation-delay: 280ms"></span>
	<span class="sr-only">{label}</span>
</span>

<style>
	.ld-dot {
		display: inline-block;
		width: 0.4rem;
		height: 0.4rem;
		border-radius: 9999px;
		background: currentColor;
		opacity: 0.4;
		animation: ld-pulse 1.1s var(--ease-out-quart, ease) infinite;
	}

	@keyframes ld-pulse {
		0%, 80%, 100% {
			opacity: 0.3;
			transform: scale(0.85);
		}
		40% {
			opacity: 1;
			transform: scale(1.1);
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.ld-dot {
			animation: none;
			opacity: 0.8;
		}
	}
</style>
