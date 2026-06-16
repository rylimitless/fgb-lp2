# DESIGN_SYSTEM.md — FGB Academy Design System Reference

> Comprehensive reference for the FGB Academy design system. Use this as the single source of truth when building new surfaces or auditing existing ones.
>
> See also: [`BRAND.md`](./BRAND.md) (brand bible), [`gia.md`](./gia.md) (mascot voice), [`MCP_LOG.md`](./MCP_LOG.md) (per-component MCP usage), [`RESEARCH.md`](./RESEARCH.md) (competitive patterns), [`ACCESSIBILITY.md`](./ACCESSIBILITY.md) (WCAG AA checklist).

---

## 1 · Foundations

### 1.1 Colour tokens

All tokens are CSS custom properties defined in [`src/routes/layout.css`](../../routes/layout.css) and exposed as Tailwind utility classes via `@theme inline`. Light mode is canonical; dark mode lifts hues.

**Brand**

- `--primary` — FGB navy (`#00548e` in sRGB; OKLCH `0.42 0.13 245` light, `0.62 0.13 240` dark). Use for primary CTAs, focus rings, brand surfaces, sidebar primary.
- `--accent` — FGB gold (`#d6c47e`; OKLCH `0.81 0.07 90` light, `0.85 0.08 88` dark). Use for hero accents, badges, the mark crossbar dot, XP rings, gold-glow shadows. **Never use as a button fill outside of celebration/ceremony.**
- `--primary-soft`, `--accent-soft` — 8%/18% alpha variants for hover backgrounds and chip fills.

**Surfaces**

- `--background` — page canvas.
- `--card`, `--popover` — elevated surfaces.
- `--surface-1`, `--surface-2`, `--surface-3` — soft tinted surfaces for grouping without a hard border. Use `surface-2` for empty-state cards, `surface-3` for inactive areas.

**Semantic**

- `--success` (forest green) — completion, approved, correct.
- `--warning` (amber) — pending review, expiring, low-streak.
- `--info` (blue) — tips, hints, "thinking" states (Gia coach loading uses this).
- `--destructive` (coral red) — errors, deletes, **incorrect quiz answers** — but never punishment-flavoured copy.

**Learning specials**

- `--xp` (gold) — XP rings, awarded badges.
- `--streak` (warm orange) — `StreakFlame` only.

**Borders, inputs, rings**

- `--border`, `--input`, `--ring` — single source of truth for all line work. `--ring` doubles as the focus-visible colour.

**Charts**

- `--chart-1`..`--chart-5` — navy / blue / gold / green / orange (light mode); lifted equivalents in dark. **Always use these tokens** in Chart.js datasets (see `analytics/+page.svelte` for the canonical `brandPalette` map; Chart.js cannot resolve CSS variables on canvas, so hex equivalents live in the component).

### 1.2 Typography

- **Family**: Inter Variable (`@fontsource-variable/inter`). One family across the entire platform.
- **Body**: `--font-sans` → `text-sm`/`text-base` with `leading-relaxed`.
- **Display ramp** (declared in `layout.css` `@layer base`):
  - `.text-display-2xl` — hero numbers, ceremony titles. `clamp(3rem, 5vw + 1rem, 4.5rem)`.
  - `.text-display-xl` — login/landing hero headlines.
  - `.text-display-lg` — dashboard hero greeting, page heroes.
  - `.text-display-md` — `StatCard` values, section heroes.
- **Tabular numerals**: `.tabular` — apply to any digit that ticks (timers, counters, scores, percentages, XP, theta).
- **Eyebrow**: `text-[11px] font-semibold uppercase tracking-[0.18em]` is the codified eyebrow recipe (used in `PageHeader`, dashboard hero, completion ceremony).

### 1.3 Spacing & radius

- Tailwind's default scale unchanged.
- Radius scale derived from `--radius` (`0.625rem`):
  - `--radius-sm` → `--radius-3xl`, all interpolations of `--radius`.
  - `--radius-pill` → `9999px` for streak pills, badge chips.
- **Common recipes**: cards `rounded-2xl`, hero panels `rounded-3xl`, pills `rounded-full`, inputs/buttons `rounded-md`.

### 1.4 Elevation

Three shadow tokens, navy-tinted in light, deep-ink in dark:

- `--shadow-sm` — subtle press states.
- `--shadow-md` — default card lift.
- `--shadow-lg` — applied on `.lift:hover`.
- `--shadow-glow` — **celebration only**. Gold halo. Used on ceremony CTAs, achievement reveals, the `BadgeMedal` outer glow. Never sprinkle.

### 1.5 Motion tokens

All in [`src/lib/styles/motion.css`](../../styles/motion.css).

**Easings**

- `--ease-out-quart` (default decel)
- `--ease-out-quint` (smoother decel)
- `--ease-spring` (overshoot — bump animations)
- `--ease-emphasized` (long, weighty motion — glow pulses)

**Durations**

- `--dur-fast` 120ms — micro-interactions (press).
- `--dur-base` 200ms — hover-state transitions.
- `--dur-slow` 320ms — rise-in entrances.
- `--dur-ceremony` 720ms — completion ceremony only.

**Utilities (Tailwind classes)**

- `.motion-rise-in` — fade + 8px upward translate, slow duration.
- `.motion-fade-in` — opacity only.
- `.motion-bump` — spring scale 1.18 then back.
- `.motion-glow` — infinite pulsing gold halo.
- `.motion-spin-slow` — 14s linear rotate.
- `.motion-shimmer` — skeleton shimmer.
- `.motion-stagger-1` … `.motion-stagger-6` — animation-delay 40–240ms; pair with `motion-rise-in` for cascading lists.
- `.lift` — hover: translate up 2px + `--shadow-lg`.
- `.press` — active: translate down 1px + scale 0.998.

All utilities are **disabled** under `prefers-reduced-motion: reduce`.

---

## 2 · Component recipes

Every brand component lives in [`src/lib/components/brand/`](../components/brand/) and is re-exported from the barrel `index.ts`. Import like:

```ts
import { StatCard, GiaAvatar, BadgeMedal, GiaTip } from "$lib/components/brand";
```

### 2.1 When to use which

- **`BrandLogo`** — app header (lockup, size 26), login/setup panel (lockup, size 32+), hero watermark (mark only, opacity 25%). Three variants: `mark`, `lockup`, `wordmark`.
- **`GiaAvatar`** — anywhere Gia speaks. Three states: `idle` (default), `thinking` (loading / generating), `celebrating` (completion). Halo is on by default; turn off only when wrapping in a custom ring.
- **`GiaTip`** — single inline coach hint. Pair with sentence-case copy from `gia.md` §6. Three tones: `default`, `celebrate` (gold accent fill), `thinking` (info fill).
- **`PageHeader`** — every page's primary header. Provide `title`, optional `eyebrow`, `description`, `icon`, `actions`. The `hero` variant adds a brand-gradient panel and a watermark mark — reserve for marketing-heavy pages.
- **`StatCard`** — every KPI tile. Includes `NumberTicker` animation, optional `trend` percentage, hover gold underline. Pass `tone` to switch the icon chip colour (`neutral` / `primary` / `success` / `warning` / `info` / `destructive` / `accent`).
- **`XpRing`** — circular progress at any size. Default centre shows `{value}%`; pass a `center` snippet for custom content (used on dashboard hero, completion ceremony, lesson-player module rings).
- **`BadgeMedal`** — tiered achievement medal (`bronze` / `silver` / `gold`). Use only on completion ceremony, the `AchievementsWall` row, and the certificate.
- **`StreakFlame`** — single, in-header or dashboard streak pill. Bumps when count changes.
- **`AnimatedGradient`** — decorative background. Three variants: `primary` (navy hero), `subtle` (soft brand gradient), `celebration` (radial top spotlight). Use behind `Spotlight` on login/setup hero.
- **`Spotlight`** — warm radial glow, always from the upper-left per the FGB lighting rule. Layer behind brand gradients to add depth.
- **`BorderBeam`** — animated border highlight. Use sparingly: `Continue Learning` resume card, premium reveal cards. Not on every card or it becomes noise.
- **`NumberTicker`** — count-up animation. Uses `svelte/motion` `Tween`. Already wired into `StatCard`; reuse directly for percentages on completion screens.
- **`Confetti`** — fixed-overlay particle burst. Fires once on mount and re-fires when `trigger` prop changes. Use only on course completion when score ≥ 50% (`lesson-player`).
- **`Marquee`** — pause-on-hover horizontal scroll. Reserve for featured-courses strip on dashboard or admin recent-activity ribbons.
- **`LoadingDots`** — three-dot pulse with `role="status"`. Default loading affordance everywhere a spinner would otherwise sit.
- **`EmptyState`** — Gia-led empty surface with title, description, and CTA snippet slot. Use everywhere a list/table can be empty.

### 2.2 Component prop quick-ref

Authoritative type signatures live in each `.svelte` file. Common patterns:

- All visual components accept a `class` prop merged via `cn()` from `$lib/utils.js`.
- Sized components (`GiaAvatar`, `BadgeMedal`, `XpRing`, `BrandLogo`) accept `size?: number` in pixels.
- Stateful brand components (`GiaAvatar`, `Confetti`, `NumberTicker`) honour `prefersReducedMotion.current` from `svelte/motion`.
- Snippet slots use Svelte 5 snippet syntax: declare in `Props` as `Snippet`, render with `{@render snippetName()}`.

---

## 3 · Pattern library

### 3.1 Page shell

Every page should start with:

```svelte
<PageHeader title="…" eyebrow="…" description="…">
    {#snippet icon()}
        <SomeLucideIcon class="size-6 text-primary" />
    {/snippet}
</PageHeader>
```

### 3.2 Stat row

```svelte
<div class="grid grid-cols-2 md:grid-cols-4 gap-3">
    <div class="motion-rise-in motion-stagger-1">
        <StatCard label="…" value={n} tone="primary" hint="…">
            {#snippet icon()}<SomeIcon class="size-3.5" />{/snippet}
        </StatCard>
    </div>
    <!-- stagger-2, stagger-3, stagger-4 -->
</div>
```

### 3.3 Card grid with lift

```svelte
{#each items as item, i}
    <button
        class="rounded-2xl border border-border bg-card p-5 text-left lift press motion-rise-in"
        style="animation-delay: {Math.min(i * 40, 240)}ms"
    >…</button>
{/each}
```

### 3.4 Loading state

```svelte
<div class="flex items-center gap-2 text-muted-foreground">
    <LoadingDots label="…" />
    <span class="text-sm">Reading your sources…</span>
</div>
```

### 3.5 Empty state with Gia

```svelte
<EmptyState
    title="…"
    description="Sentence-case copy from gia.md §6."
>
    {#snippet action()}
        <Button.Root variant="outline">…</Button.Root>
    {/snippet}
</EmptyState>
```

### 3.6 Ceremony card

```svelte
<section class="relative overflow-hidden rounded-3xl border border-border brand-gradient px-6 py-10 md:px-12 md:py-14 text-primary-foreground motion-rise-in">
    <Spotlight class="h-full w-full" opacity={0.7} />
    <div class="relative z-10 flex flex-col items-center text-center gap-6">
        <BadgeMedal tier={tier} size={120} />
        <GiaAvatar state="celebrating" size={56} />
        <h1 class="text-display-lg font-bold tracking-tight text-primary-foreground">…</h1>
    </div>
</section>
```

---

## 4 · Accessibility promises

Every component in this system commits to:

- **Contrast**: every fg/bg pair declared in `layout.css` clears WCAG AA (4.5:1 text, 3:1 UI components). See [`ACCESSIBILITY.md`](./ACCESSIBILITY.md) for the verified matrix.
- **Keyboard**: every interactive element is reachable via Tab, has a visible focus ring (`:focus-visible` consumes `--ring`), and triggers on Enter/Space when appropriate.
- **ARIA**: every icon-only button has an `aria-label`; every loading dots has `role="status"`; every alert uses `role="alert"`; every Gia avatar has a descriptive `alt`.
- **Reduced motion**: every keyframe animation and component-level motion is gated by `@media (prefers-reduced-motion: reduce)` and/or `prefersReducedMotion.current` from `svelte/motion`.
- **Colour independence**: no UI conveys state by colour alone — every status pill carries a text label, every chart series has a label and a stroke-pattern fallback.
- **Screen reader**: every `GiaAvatar` reads "Gia, your FGB Academy coach, {state}". Every chart canvas should have an `aria-label` summarising the dataset.

---

## 5 · Adding a new component

The master prompt requires the **5-step MCP gate** for every new component. Follow the procedure in [`MCP_LOG.md`](./MCP_LOG.md):

1. **A** — Search 21st.dev via `21st_magic_component_inspiration`.
2. **B** — Search Magic UI via the appropriate `get*` tool.
3. **C** — Validate the chosen Svelte 5 / Tailwind 4 / shadcn-svelte API via Context7 `query-docs`.
4. **D** — Cross-reference a pattern in `RESEARCH.md` (steal or avoid).
5. **E** — Build native in Svelte 5 only if A and B come up empty (always the case so far — both registries are React).

Record each component's gate result under "Phase 0.3 / Per-component MCP gate" in `MCP_LOG.md` before moving on.

---

## 6 · Versioning

This is a living document. The convention:

- A **token change** (colour, radius, motion) updates this file's §1 + the `layout.css` comment block on the same commit.
- A **new component** updates §2 + adds an `MCP_LOG.md` entry.
- A **deprecation** moves the component to a "Deprecated" section here for one release before removal.

_Last reviewed: 2026-06-15 — initial codification. Token set unchanged from previous turn._
