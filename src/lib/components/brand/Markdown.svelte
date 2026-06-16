<script lang="ts" module>
	// Lightweight, dependency-free Markdown -> safe HTML renderer.
	// Input is HTML-escaped first, then a small set of Markdown constructs are
	// converted. No raw HTML passes through, so {@html} is safe here.

	function escapeHtml(s: string): string {
		return s
			.replace(/&/g, "&amp;")
			.replace(/</g, "&lt;")
			.replace(/>/g, "&gt;")
			.replace(/"/g, "&quot;");
	}

	function inline(s: string): string {
		// inline code first (protect its contents)
		const codes: string[] = [];
		s = s.replace(/`([^`]+)`/g, (_m, c) => {
			codes.push(c);
			return `\u0000${codes.length - 1}\u0000`;
		});
		// links [text](url) — only safe schemes
		s = s.replace(/\[([^\]]+)\]\(([^)\s]+)\)/g, (_m, t, url) => {
			const safe = /^(https?:\/\/|\/|mailto:)/i.test(url) ? url : "#";
			return `<a href="${safe}" class="text-primary underline underline-offset-2 hover:text-primary/80"${safe.startsWith("http") ? ' target="_blank" rel="noopener noreferrer"' : ""}>${t}</a>`;
		});
		// bold then italic
		s = s.replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>");
		s = s.replace(/__([^_]+)__/g, "<strong>$1</strong>");
		s = s.replace(/(^|[^*])\*([^*\n]+)\*/g, "$1<em>$2</em>");
		s = s.replace(/(^|[^_])_([^_\n]+)_/g, "$1<em>$2</em>");
		// restore code
		s = s.replace(/\u0000(\d+)\u0000/g, (_m, i) => `<code class="rounded bg-muted px-1 py-0.5 text-[0.85em] font-mono">${codes[+i]}</code>`);
		return s;
	}

	export function renderMarkdown(src: string): string {
		const text = escapeHtml(src ?? "").replace(/\r\n/g, "\n");
		const lines = text.split("\n");
		const out: string[] = [];
		let i = 0;
		let para: string[] = [];
		let listType: "ul" | "ol" | null = null;

		const flushPara = () => {
			if (para.length) {
				out.push(`<p>${inline(para.join(" "))}</p>`);
				para = [];
			}
		};
		const closeList = () => {
			if (listType) {
				out.push(`</${listType}>`);
				listType = null;
			}
		};

		while (i < lines.length) {
			const line = lines[i] ?? "";
			const trimmed = line.trim();

			// fenced code block
			if (trimmed.startsWith("```")) {
				flushPara();
				closeList();
				const buf: string[] = [];
				i++;
				while (i < lines.length && !(lines[i] ?? "").trim().startsWith("```")) {
					buf.push(lines[i] ?? "");
					i++;
				}
				i++; // skip closing fence
				out.push(`<pre class="my-3 overflow-x-auto rounded-lg bg-muted p-3 text-xs font-mono leading-relaxed"><code>${buf.join("\n")}</code></pre>`);
				continue;
			}

			// blank line
			if (trimmed === "") {
				flushPara();
				closeList();
				i++;
				continue;
			}

			// horizontal rule
			if (/^(-{3,}|_{3,}|\*{3,})$/.test(trimmed)) {
				flushPara();
				closeList();
				out.push(`<hr class="my-4 border-border" />`);
				i++;
				continue;
			}

			// headings
			const h = trimmed.match(/^(#{1,6})\s+(.*)$/);
			if (h) {
				flushPara();
				closeList();
				const level = h[1]!.length;
				const sizes = ["text-2xl", "text-xl", "text-lg", "text-base", "text-sm", "text-sm"];
				out.push(
					`<h${level} class="mt-4 mb-2 font-semibold text-foreground ${sizes[level - 1]} first:mt-0">${inline(h[2]!)}</h${level}>`,
				);
				i++;
				continue;
			}

			// blockquote
			if (trimmed.startsWith("&gt;")) {
				flushPara();
				closeList();
				const q = trimmed.replace(/^&gt;\s?/, "");
				out.push(`<blockquote class="my-3 border-l-2 border-accent pl-3 text-muted-foreground italic">${inline(q)}</blockquote>`);
				i++;
				continue;
			}

			// unordered list
			const ul = trimmed.match(/^[-*+]\s+(.*)$/);
			if (ul) {
				flushPara();
				if (listType !== "ul") {
					closeList();
					out.push(`<ul class="my-2 ml-5 list-disc space-y-1 marker:text-accent">`);
					listType = "ul";
				}
				out.push(`<li>${inline(ul[1]!)}</li>`);
				i++;
				continue;
			}

			// ordered list
			const ol = trimmed.match(/^\d+\.\s+(.*)$/);
			if (ol) {
				flushPara();
				if (listType !== "ol") {
					closeList();
					out.push(`<ol class="my-2 ml-5 list-decimal space-y-1 marker:text-muted-foreground">`);
					listType = "ol";
				}
				out.push(`<li>${inline(ol[1]!)}</li>`);
				i++;
				continue;
			}

			// paragraph text
			closeList();
			para.push(trimmed);
			i++;
		}
		flushPara();
		closeList();
		return out.join("\n");
	}
</script>

<script lang="ts">
	import { cn } from "$lib/utils.js";

	type Props = { source?: string; class?: string };
	let { source = "", class: className }: Props = $props();

	let html = $derived(renderMarkdown(source));
</script>

<div class={cn("fgb-md text-foreground", className)}>
	<!-- eslint-disable-next-line svelte/no-at-html-tags — input is escaped in renderMarkdown -->
	{@html html}
</div>

<style>
	.fgb-md :global(p) {
		margin-block: 0.5rem;
		line-height: 1.7;
	}
	.fgb-md :global(p:first-child) {
		margin-top: 0;
	}
	.fgb-md :global(li) {
		line-height: 1.6;
	}
	.fgb-md :global(strong) {
		font-weight: 600;
		color: var(--foreground);
	}
</style>
