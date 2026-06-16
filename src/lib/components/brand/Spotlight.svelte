<script lang="ts">
	import { cn } from "$lib/utils.js";

	type Props = {
		color?: string;
		size?: number;
		class?: string;
		opacity?: number;
	};

	let {
		color = "var(--accent)",
		size = 720,
		opacity = 0.55,
		class: className,
	}: Props = $props();
</script>

<!--
	Single spotlight — soft, warm radial glow that follows the upper-left rule
	of the FGB brand. Decorative only.
-->
<div
	aria-hidden="true"
	class={cn("sp-root pointer-events-none absolute -z-0", className)}
	style:--sp-color={color}
	style:--sp-size="{size}px"
	style:--sp-opacity={opacity}
></div>

<style>
	.sp-root {
		left: -10%;
		top: -20%;
		width: var(--sp-size, 720px);
		height: var(--sp-size, 720px);
		background: radial-gradient(
			circle at center,
			var(--sp-color, var(--accent)) 0%,
			transparent 60%
		);
		filter: blur(40px);
		opacity: var(--sp-opacity, 0.55);
	}
</style>
