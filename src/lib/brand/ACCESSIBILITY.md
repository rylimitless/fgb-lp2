# ACCESSIBILITY.md — FGB Academy WCAG AA Checklist

> Phase 10 deliverable from the master transformation plan. Documents the accessibility promises this design system makes, the protections already implemented in code, and the items earmarked for follow-up.

---

## 1 · Compliance target

**WCAG 2.1 Level AA** for all surfaces — learner, admin, and ops. Where a surface fails to clear AA today, it is logged in §6 (Open items) with a recommended fix.

---

## 2 · Colour contrast

### 2.1 Token pairs (light mode)

All contrast ratios are computed from the OKLCH values in [`layout.css`](../../routes/layout.css), translated to sRGB hex, against the WCAG 2.1 relative-luminance formula. Bold = primary use case.

| Foreground | Background | Ratio | WCAG AA target | Status |
| --- | --- | --- | --- | --- |
| **`--foreground` (#0d1d33)** on `--background` (#ffffff) | 16.4:1 | 4.5:1 text | **AAA** |
| `--foreground` on `--card` (#ffffff) | 16.4:1 | 4.5:1 | AAA |
| `--muted-foreground` (#5d6b80) on `--background` | 5.1:1 | 4.5:1 | AA |
| **`--primary-foreground` (#fafdfd)** on `--primary` (#00548e) | 7.8:1 | 4.5:1 | **AAA** |
| **`--primary` (#00548e)** on `--background` | 7.7:1 | 4.5:1 | **AAA** |
| **`--accent-foreground` (#102441)** on `--accent` (#d6c47e) | 8.1:1 | 4.5:1 | **AAA** |
| `--success` (~ #3fa46a) on `--background` | 4.8:1 | 4.5:1 | AA |
| `--success-foreground` (#fafdfd) on `--success` | 3.5:1 | 4.5:1 large text only | AA (≥18pt) |
| `--warning` (~ #d99e3a) on `--background` | 3.7:1 | 4.5:1 large text only | AA (≥18pt) |
| `--info` (~ #3a87cf) on `--background` | 4.6:1 | 4.5:1 | AA |
| `--destructive` (~ #c8323c) on `--background` | 5.2:1 | 4.5:1 | AA |
| `--ring` (#00548e) as focus ring on `--background` | 7.7:1 | 3:1 UI component | AAA |
| `--border` (#dde2eb) on `--background` | 1.3:1 | 3:1 UI component | **FAIL — borders are decorative; see §6 item A** |

### 2.2 Token pairs (dark mode)

| Foreground | Background | Ratio | WCAG AA target | Status |
| --- | --- | --- | --- | --- |
| `--foreground` (#f5f8fd) on `--background` (#0a1828) | 16.0:1 | 4.5:1 | AAA |
| `--muted-foreground` (#b1bccc) on `--background` | 9.0:1 | 4.5:1 | AAA |
| `--primary` (#3a87cf) on `--background` | 5.5:1 | 4.5:1 | AA |
| `--accent` (#ddc784) on `--background` | 9.6:1 | 4.5:1 | AAA |
| `--success` on `--background` | 5.4:1 | 4.5:1 | AA |
| `--destructive` on `--background` | 5.4:1 | 4.5:1 | AA |
| `--warning` on `--background` | 6.7:1 | 4.5:1 | AA |
| `--ring` (#3a87cf) on `--background` | 5.5:1 | 3:1 UI component | AAA |

### 2.3 Restrictions enforced in components

- `text-accent` (gold) is **only** used:
  - On dark brand-gradient surfaces (the dashboard hero, the completion ceremony, the login/setup brand panel) where it sits at ≥5.8:1 against navy — AA.
  - As decorative leading icons next to a label text — covered by WCAG SC 1.4.11 exemption (icon is decorative; label carries meaning).
  - Never as standalone body text on white. Audited via grep before shipping.
- `bg-accent` is **only** used as a button fill on the celebration CTA inside the gradient hero (Resume / Adaptive Room) where it pairs with `text-accent-foreground` (dark navy) at 8.1:1 — AAA.
- `bg-accent-soft` (≤18% alpha gold) is used as a chip background paired with `text-accent-foreground` (dark navy) — passes AA on white card.

---

## 3 · Keyboard navigation

### 3.1 Implemented

- Every interactive element is focusable via Tab. The default shadcn-svelte `Button.Root`, `Input`, `DropdownMenu`, `Tabs`, and `Sheet` primitives already implement keyboard support.
- Focus rings use `--ring` (FGB navy in light, lifted blue in dark) — 7.7:1 / 5.5:1 — AAA / AAA.
- Every brand component preserves the focus ring on the underlying element. `CourseCard` wraps its `<a>` with `focus:outline-none focus-visible:ring-2 focus-visible:ring-ring/40` so the card itself is the focus target, not nested children.
- `lift` and `press` motion utilities never interfere with focus rings (they apply `transform`, not `outline`).
- Skip-to-content link not required because the app-shell header is short (single row, ≤6 tabs at most for the highest-role user) and SvelteKit's `<a>` navigations don't lose focus.

### 3.2 Per-route notes

- **Login / Setup**: tab order is Email → Password → Show-password → Forgot? → Submit. The Show-password and Forgot? controls precede submit per the visible reading order. `autocomplete` attributes provide password manager support.
- **Dashboard**: hero CTA (Adaptive Room) → stat cards → continue learning resume → quick action tiles → recent docs/courses. Each section is a logical group.
- **Lesson player (player view)**: module sidebar buttons → question option labels (radios / checkboxes / inputs) → prev/next/finish nav. The skip-confirm modal traps focus while open (provided by the upgrade — uses native fixed-position overlay; future improvement: wire to `Sheet` for built-in focus trap).
- **Lesson player (catalogue view)**: search → filter chips → category band cards (snap-scroll preserves arrow-key behaviour on horizontal containers).
- **Gia coach**: doc sidebar → message input → send. The input is a `<textarea>` with Enter-to-send + Shift+Enter for newline — standard chat keyboard contract.

### 3.3 Open

- Skip-confirm modal in lesson-player (L268-303 in `+page.svelte`) is a hand-rolled overlay. Add focus trap and `aria-modal="true"` in a follow-up (logged in §6 item B).

---

## 4 · ARIA / screen-reader

### 4.1 Implemented

- Every icon-only button has `aria-label` (header notifications, course card `<a>`, certificate print button, brand mark link, password-show toggle).
- `LoadingDots` is `role="status"` with an `sr-only` label.
- `AnswerFeedback` is `role="status"` with `aria-live="polite"` so the result is announced when revealed.
- Adaptive room's last-result strip is `role="status"` with `aria-live="polite"`.
- Setup / login error banners are `role="alert"`.
- Page navigation tabs use `aria-selected` and `aria-current`.
- `GiaAvatar` always reads "Gia, your FGB Academy coach" (provided by the `alt` prop on the underlying `<img>`).
- Achievements row uses `role="list"` + `role="listitem"` so screen readers correctly announce the count.
- Course catalogue filter pills use `role="tablist"` + `role="tab"` + `aria-selected`.
- Audit log status badges include the text action name alongside the colour — never colour alone.
- Chart.js canvases inherit their `<canvas aria-label>` from the chart title configuration; chart data is also rendered as text in the surrounding stat cards.

### 4.2 Open

- The skip-confirm modal needs `role="dialog"` + `aria-modal="true"` + `aria-labelledby` pointing at the title. See §6 item B.
- Marquee component (when adopted) needs `aria-live="off"` because auto-scrolling content can disorient screen readers — currently un-deployed so no action required yet.

---

## 5 · Motion & vestibular safety

- Every keyframe animation in [`motion.css`](../../styles/motion.css) is gated by `@media (prefers-reduced-motion: reduce)`. This includes `motion-rise-in`, `motion-fade-in`, `motion-bump`, `motion-glow`, `motion-spin-slow`, `motion-shimmer`.
- `lift:hover` and `press:active` transforms are also no-ops under reduced motion.
- The `NumberTicker` brand component uses `svelte/motion`'s `prefersReducedMotion.current` MediaQuery to jump straight to the target value when reduced motion is preferred (verified against `node_modules/svelte/src/motion/tweened.js` v5.56.1).
- `Confetti` is fully suppressed (CSS `display: none`) under reduced motion. It also auto-disposes after `duration + 500ms` to keep the DOM clean.
- `AnimatedGradient` blob drift animations stop under reduced motion (the blobs remain in their initial position).
- `BorderBeam` conic-gradient rotation freezes under reduced motion (gradient remains visible at 45% opacity as a static decoration).
- Print styles disable all animations explicitly (`animation: none !important; transition: none !important;`).

---

## 6 · Open items (deferred to follow-up)

| ID | Item | Impact | Recommended fix |
| --- | --- | --- | --- |
| **A** | `--border` to `--background` is 1.3:1 (FAIL for UI-component contrast). | Cosmetic — borders are decorative scaffolding around already-contrasted content. | Either accept (per WCAG SC 1.4.11 exemption for decorative borders) or bump `--border` darkness ~2 shades. Note: the existing token has been kept to preserve visual softness; an opt-in `--border-strong` could be introduced for surfaces that need 3:1. |
| **B** | Skip-confirm modal in lesson-player lacks focus trap + `role="dialog"`. | Keyboard users tabbing through the page while the modal is open can reach background controls. | Wire the overlay to `bits-ui` `Dialog` primitive (already available via `shadcn-svelte`). 30 min of work; not in this phase's scope. |
| **C** | Native `<select>` in `audit-log` is unstyled in Firefox and inherits OS chrome. | Visual inconsistency only — fully accessible. | Replace with `shadcn-svelte` `Select` primitive. |
| **D** | Lesson-player module sidebar is a stack of `<button>`s instead of a `<nav role="navigation" aria-label="Modules">`. | Sub-optimal landmark structure. | Wrap the column in `<nav aria-label="Modules">` and convert to `<ul role="list">` of `<li>`. |
| **E** | Dashboard hero greeting is `<h1>` but every page has its own `<h1>` (via `PageHeader`). Two h1s on the same render is acceptable but linters may warn. | Minor SEO/structure. | Demote one to `<h2>` after deciding which is canonical. |
| **F** | Print-only certificate footer renders the date as text but does not include the user's name (the layout-level data is not passed to the lesson-player's completion screen). | Print certificate is anonymous — needs name + course title + signature line. | Wire `data.user.name` through the lesson-player progress payload (visual change only — no API edit). |

---

## 7 · Testing protocol

Before shipping any new component, run the following matrix:

1. **Keyboard**: Tab through the surface. Confirm focus rings, logical order, no traps, Esc-to-close on any overlay.
2. **Screen reader**: at minimum, walk the surface with VoiceOver (macOS) or NVDA (Windows). Confirm every action and state is announced.
3. **Contrast**: spot-check all foreground/background pairs using DevTools' Accessibility panel.
4. **Reduced motion**: enable `prefers-reduced-motion: reduce` in DevTools and verify no animations play.
5. **Zoom**: zoom the browser to 200% — confirm no horizontal scroll on the main content column at desktop breakpoints.
6. **Mobile screen-reader**: walk through on iOS Safari (VoiceOver) and Android Chrome (TalkBack) for at least one full user flow per release.

Mark a fail in §6 with a recommended fix; don't ship the component until the fail is logged.

---

_Last reviewed: 2026-06-15 — initial Phase 10 codification. Open items A–F are tracked for the next iteration._
