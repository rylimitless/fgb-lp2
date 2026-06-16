<script lang="ts">
	import { cn } from "$lib/utils.js";
	import type { HTMLAttributes } from "svelte/elements";

	type Variant = "mark" | "lockup" | "wordmark";

	type Props = HTMLAttributes<HTMLDivElement> & {
		variant?: Variant;
		size?: number;
		class?: string;
		"aria-label"?: string;
	};

	let {
		variant = "mark",
		size = 32,
		class: className,
		"aria-label": ariaLabel = "FGB Academy",
		...rest
	}: Props = $props();

	let height = $derived(size);
	let width = $derived(
		variant === "mark" ? size : variant === "lockup" ? size * 5 : size * 3.75,
	);
</script>

<div
	class={cn("inline-flex items-center text-foreground", className)}
	style:height="{height}px"
	style:width="{width}px"
	role="img"
	aria-label={ariaLabel}
	{...rest}
>
	{#if variant === "mark"}
		<svg
			xmlns="http://www.w3.org/2000/svg"
			viewBox="0 0 64 64"
			width="100%"
			height="100%"
			aria-hidden="true"
		>
			<path
				d="M 12 54 L 32 12 L 52 54"
				fill="none"
				stroke="currentColor"
				stroke-width="7"
				stroke-linecap="round"
				stroke-linejoin="round"
			/>
			<circle cx="32" cy="40" r="4.5" fill="var(--accent)" />
		</svg>
	{:else if variant === "lockup"}
		<svg
			xmlns="http://www.w3.org/2000/svg"
			viewBox="0 0 320 64"
			width="100%"
			height="100%"
			aria-hidden="true"
		>
			<g>
				<path
					d="M 12 54 L 32 12 L 52 54"
					fill="none"
					stroke="currentColor"
					stroke-width="7"
					stroke-linecap="round"
					stroke-linejoin="round"
				/>
				<circle cx="32" cy="40" r="4.5" fill="var(--accent)" />
			</g>
			<line
				x1="76"
				y1="14"
				x2="76"
				y2="50"
				stroke="currentColor"
				stroke-opacity="0.18"
				stroke-width="1.5"
			/>
			<text
				x="92"
				y="32"
				font-family="'Inter Variable', Inter, system-ui, sans-serif"
				font-size="22"
				font-weight="700"
				letter-spacing="-0.02em"
				fill="currentColor">FGB</text
			>
			<text
				x="92"
				y="50"
				font-family="'Inter Variable', Inter, system-ui, sans-serif"
				font-size="11"
				font-weight="600"
				letter-spacing="0.22em"
				fill="var(--accent)">ACADEMY</text
			>
		</svg>
	{:else}
		<svg
			xmlns="http://www.w3.org/2000/svg"
			viewBox="0 0 240 64"
			width="100%"
			height="100%"
			aria-hidden="true"
		>
			<text
				x="0"
				y="32"
				font-family="'Inter Variable', Inter, system-ui, sans-serif"
				font-size="22"
				font-weight="700"
				letter-spacing="-0.02em"
				fill="currentColor">FGB</text
			>
			<text
				x="0"
				y="50"
				font-family="'Inter Variable', Inter, system-ui, sans-serif"
				font-size="11"
				font-weight="600"
				letter-spacing="0.22em"
				fill="var(--accent)">ACADEMY</text
			>
		</svg>
	{/if}
</div>
