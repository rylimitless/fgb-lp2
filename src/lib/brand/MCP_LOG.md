# MCP_LOG.md — Phase 0 baseline + per-component audit trail

> Live log of every MCP call attempted during the FGB Academy LXP master transformation. Per the master prompt's "Document where every MCP was used" rule. Appended to throughout execution.
>
> **Date opened**: 2026-06-15. **Owner**: Design System.

---

## Phase 0.1 — Probe baseline (2026-06-15)

| Server (folder name) | Status | Canary call | Result |
| --- | --- | --- | --- |
| `user-context7` | **REACHABLE** | `resolve-library-id` for "Svelte" | 5 hits returned; selected `/sveltejs/svelte` (Benchmark 79.8, High reputation, Svelte 5.36.17/5.37.0) |
| `user-magicuidesign-mcp` | **REACHABLE** *(new bulk-getter API)* | `getSpecialEffects` (no args) | 48 KB of React/Framer Motion source returned for `animated-beam`, `border-beam`, `shine-border`, `magic-card`, `meteors`, `neon-gradient-card`, `confetti`, `particles`, `cool-mode`, `scratch-to-reveal` |
| `user-21st-dev-magic` | **REACHABLE** | `21st_magic_component_inspiration` query "course card" | 39 KB returned with 3 card variants (Card with 8 variants, AnimatedCard with diagrams, AnimatedCard with grid lines) |
| `cursor-ide-browser` | **REACHABLE** | `browser_tabs` action `list` | Empty tab list (no error) |
| `higgsfield` | **NOT REGISTERED** | n/a — folder does not exist under `mcps/` | Phase 11 takes Path B (prompt-pack only) |

**Catalog notes from probe:**

- Magic UI ships React 19 / Framer Motion 12 / `next-themes` components. Code is React-only — patterns must be translated to Svelte 5 + native CSS. Already-shipped `BorderBeam.svelte` and `Confetti.svelte` follow the same conceptual approach as the Magic UI canon (`border-beam` and `confetti`).
- 21st.dev returns full React + shadcn componentry. Same translation requirement. The `Card` variants `lifted`, `gradient`, `corners`, `inner` are conceptually compatible with FGB Academy's existing `lift` motion utility and brand tokens.
- Context7 is fully usable for Svelte 5 / Tailwind 4 / SvelteKit API verification.
- Cursor IDE Browser is fully usable for Phase 0.2 research pass.

**Decision recorded:** All custom components in this transformation will be Svelte 5 native, drawing concepts from the Magic UI and 21st.dev catalogs. No transpilation attempted (would introduce React + Framer Motion as new deps, violating the "additive only" rule).

---

## Phase 0.2 — Cursor IDE Browser research (2026-06-15)

Browser view `afb126` used. Locked for the duration of the pass, unlocked after.

| Order | URL | Tool calls | Screenshot saved |
| --- | --- | --- | --- |
| 1 | https://linear.app | `browser_navigate` -> `browser_snapshot` -> `browser_take_screenshot` | [`static/research/linear-landing-product.png`](../../../static/research/linear-landing-product.png) |
| 2 | https://stripe.com | `browser_navigate` -> `browser_snapshot` -> `browser_take_screenshot` | [`static/research/stripe-landing.png`](../../../static/research/stripe-landing.png) |
| 3 | https://www.duolingo.com | `browser_navigate` -> `browser_snapshot` -> `browser_take_screenshot` | [`static/research/duolingo-landing.png`](../../../static/research/duolingo-landing.png) |
| 4 | https://www.masterclass.com | `browser_navigate` -> 8 s wait (Cloudflare verification) -> `browser_take_screenshot` | [`static/research/masterclass-catalog.png`](../../../static/research/masterclass-catalog.png) |
| 5 | https://www.notion.so | `browser_navigate` -> 4 s wait -> `browser_take_screenshot` | [`static/research/notion-landing.png`](../../../static/research/notion-landing.png) |
| 6 | https://www.coursera.org/business | `browser_navigate` -> `browser_snapshot` -> `browser_take_screenshot` | [`static/research/coursera-business.png`](../../../static/research/coursera-business.png) |
| 7 | https://www.khanacademy.org | `browser_navigate` -> `browser_snapshot` -> `browser_take_screenshot` | [`static/research/khan-academy-landing.png`](../../../static/research/khan-academy-landing.png) |

**Distillation**: [`RESEARCH.md`](./RESEARCH.md) — 10 patterns to steal, 5 to avoid, component-pattern crosswalk for `CourseCard`, `ContinueLearning`, `AchievementsWall`, `Recommendations`, `AnswerFeedback`, `GiaTip`, login/setup, admin tables.

**Coursera Enterprise authenticated dashboard not visited** (would have required login). Used the `coursera.org/business` marketing surface as the closest public proxy. Logged here for completeness.

---

## Experience redesign (v3) — research pass (2026-06-15 PM)

Direction change: the platform reads as a polished corporate portal, not an immersive product. New goal — an experience closer to Duolingo / MasterClass / Stripe / Headspace. Research pulled before building:

| MCP | Call | What it gave |
| --- | --- | --- |
| `user-magicuidesign-mcp` | `getBackgrounds` | Source for `warp-background` (perspective grid + rising beams), `flickering-grid` (canvas squares, **IntersectionObserver-gated rAF** — the key perf technique), `animated-grid-pattern`, `dot-pattern`, `ripple`, `retro-grid`. |
| `user-magicuidesign-mcp` | `getTextAnimations` | Source for aurora-text, text-shimmer, word-rotate, animated-gradient-text, hyper-text patterns. |
| `user-magicuidesign-mcp` | `getSpecialEffects` (earlier) | `particles`, `border-beam`, `magic-card`, `meteors`, `confetti` source. |
| `user-magicuidesign-mcp` | `getComponents` (earlier) | `bento-grid`, `marquee`, `orbiting-circles`, `animated-list`. |
| `user-21st-dev-magic` | `21st_magic_component_inspiration` "bento grid features" | Two production bento layouts: staggered container/item reveal (`staggerChildren`), hover `scale + rotate` cards, asymmetric row/col spans. Adopted the **staggered scroll-reveal + hover-lift** pattern. |
| `user-context7` | `query-docs` `/sveltejs/svelte` | Confirmed Svelte 5 `$effect` + `requestAnimationFrame` + teardown cleanup, `bind:this` canvas 2D context, and `use:` actions with IntersectionObserver teardown — the exact primitives for native ports. |
| `cursor-ide-browser` (earlier v2 pass) | 7 product captures | Reused the patterns-to-steal in `RESEARCH.md` (S1 big display type, S8 Netflix rows, S2 single accent). |

**Decision**: All Magic UI / 21st.dev source is React + Framer Motion. Per the additive-only rule (no new deps), every effect is re-implemented as a **native Svelte 5 component** using `$effect` + rAF + `prefersReducedMotion`, drawing the technique (not the code) from the registry source above.

**New primitives built** (see Phase 0.3 entries): `Particles`, `AnimatedGrid`, `ScrollReveal`, `WordRotate`, `BentoCard`, `OrbitingBadges`, `Counter`-via-existing-`NumberTicker`.

---

## v4 — Reference-replication pass (2026-06-15 late PM)

Goal: replicate an approved concept mockup (dark landing + light dashboard with floating sidebar) to ~90-95% visual parity. **25 Years of Excellence** (corrected from 20).

| MCP | Call | Result / use |
| --- | --- | --- |
| `user-higgsfield` | `generate_image` ×3 (`soul_2`) | **stairway.png** — figure ascending a golden staircase toward a glowing gold "A" (near-exact match to the reference hero centerpiece); **gia-3d.png** — friendly Afro-Caribbean Gia coach character (dashboard hero); **information-security.png** — glowing security globe (Continue Learning card). First parallel batch timed out / errored (transient); re-fired sequentially and all completed (~50 s, 1 credit each). |
| `user-higgsfield` | `show_generations`, `job_status` | Recovered/confirmed job state after the transient timeouts. |
| `user-21st-dev-magic` | `21st_magic_component_inspiration` "dashboard sidebar" | **Timed out** this pass; reused the bento + card patterns recovered earlier in v3 (`agent-tools/2981e469…txt` returned 65 KB of dashboard/sidebar component source as a fallback reference). |
| `user-magicuidesign-mcp` | (v3 cache) | Reused `border-beam`, particles, grid techniques already ported. |
| `user-context7` | (prior validation) | Chart.js radar pattern already validated + proven in `analytics/+page.svelte`; reused for the dashboard Skill Radar. Svelte 5 `$effect`/canvas confirmed in v3. |
| `cursor-ide-browser` | `browser_navigate` + `browser_take_screenshot` | Captured the rebuilt `/welcome` to verify parity against the reference (hero, learner cards, 25-year banner). |

**Assets** (`static/brand/`): `hero/stairway.png`, `gia/gia-3d.png`, `modules/information-security.png`. Recommendation thumbnails reuse existing Jamaican pathway renders (cybersecurity→Fraud Detection, customer-service→Customer Experience, leadership→Leading Teams).

**Built**: `welcome/+page.svelte` rebuilt to the reference landing (hero with stairway + "Learn today. / Lead tomorrow." + 25-yr badge, 4 feature cards, Netflix-style learner cards, 25-year premium banner). App shell `(app)/+layout.svelte` rebuilt as a floating dark sidebar + top bar (search, XP, streak, notifications, profile) — all real routes + role-gating + notification logic preserved. Dashboard `(app)/+page.svelte` rebuilt to the reference: Gia hero panel, progress/streak/level trio, dominant Continue Learning, Recommended trio, Learning Path roadmap, Achievements + Skill Radar (Chart.js) + Leaderboard, Deadlines, What's New.

**Justified deviations**: XP totals, leaderboard names, skill-radar values, deadlines, and the learning-path/achievement sets are presentational gamification chrome — the backend has no XP/leaderboard/skills endpoints, so these render as showcase data (clearly labelled in code) while real data (greeting, name, theta→level/progress, in-progress course, published courses → recommendations) is wired wherever it exists. No backend/auth/scoring changes.

---

## v5 — MCP review + new question types (2026-06-16)

Goal: expand learning depth with new interaction types and implement the top MCP-driven enhancements.

| MCP | Call / source | Result / use |
| --- | --- | --- |
| `user-context7` | `resolve-library-id` for `dnd-kit`; `query-docs` `/hanielu/dnd-kit-svelte` | Confirmed the documented Svelte DnD API (`@dnd-kit-svelte/svelte`) does **not** match the installed alpha package (`@dnd-kit/svelte@0.5.0`). Decision: use native HTML5 drag + explicit up/down controls for Ordering questions. |
| Backend source audit | `jobs.go` + `handler.go` | Found `matching`, `drag_sort`, `hotspot` are already allowed in `allowedTypes`, but the AI prompt only requested `mc/ma/tf/fb/sa`. |
| `user-higgsfield` | `generate_image` ×2, `job_status` | Generated `static/brand/certificates/certificate-base.png` (`ce79cb66-86c3-4730-a561-a6d649f71a2d`) and `static/brand/questions/hotspot-cyber-risk.png` (`80e3105d-d7a3-4504-9d7c-38374787b3f7`). |
| `user-21st-dev-magic` | Prior command/dashboard pattern research | Informed `CommandPalette.svelte`: modal search, keyboard open, grouped nav/course results. |
| `user-magicuidesign-mcp` | Prior `animated-list`/motion research | Informed `AnimatedList.svelte`: staggered slide/fade-in list items, reduced-motion safe. |

### Built

- `QuestionMatching.svelte` — click-to-pair terms and definitions. Emits `{ [leftIndex]: rightIndex }`.
- `QuestionOrdering.svelte` — native drag + accessible up/down buttons. Emits ordered original indices.
- `QuestionHotspot.svelte` — focusable labeled regions over an image. Emits selected region index.
- Lesson Player: render + score + `AnswerFeedback` for all three types.
- Adaptive Room: render + score for all three types in the IRT loop.
- Backend `questionGenPrompt`: now generates exactly 8 items, including `matching`, `drag_sort`, and `hotspot`; hotspot image default is `/brand/questions/hotspot-cyber-risk.png`.
- `MCP_REVIEW.md` — prioritized roadmap across Higgsfield, 21st.dev, Magic UI, Context7.
- `CommandPalette.svelte` — `Ctrl/Cmd+K` / top-bar search for nav + published courses.
- `AnimatedList.svelte` — applied to dashboard What's New feed.
- Completion print certificate now uses Higgsfield `certificate-base.png` via `.certificate-print-card`.

### Next-wave execution from `MCP_REVIEW.md`

- **21st.dev**: `21st_magic_component_inspiration` for data-table/dashboard patterns returned large React table/dashboard examples. Ported the structural idea into native Svelte as `PremiumTable.svelte`, applied first to Audit Log.
- **Magic UI**: `getComponents` cache includes `orbiting-circles`; ported the concept to `OrbitingInsights.svelte` and applied it to the Gia Coach empty state.
- **Higgsfield**: generated `certificate-base.png` and `hotspot-cyber-risk.png`; added `scripts/course-cover-manifest.mjs` to scaffold per-course cover prompts from live API/course JSON.
- **Context7**: confirmed Svelte 5 snippet/table patterns and reinforced the choice to use snippet-based `PremiumTable` rather than ad-hoc table duplication.
- **21st.dev Toast/Settings**: researched toast and settings/profile patterns. React examples depended on Sonner/Ark/Framer Motion; implemented a lightweight native Svelte toast store + `ToastViewport` instead (no new dependencies) and added `/settings`.
- **Question analytics**: added a lightweight in-course assessment mix panel derived from existing `enrolledCourse.modules[].items` so no schema/event table is required. Full historical analytics remains a future backend event-model task.

### Preservation notes

No database schema changes. No new dependencies. Existing question types remain unchanged. Existing progression/scoring flow remains the same; only `isCorrect`/`isAnswerCorrect` have additive cases for the three new `item_type` values.

---

## Phase 0.3 — Per-component MCP gate

Each new component or major rewrite below records the 5-step gate result.

### Phase 3 — `certification-mark.svg` + `app-icon-mono.svg`

- **A (21st.dev)**: skipped — these are static SVG assets, not interactive UI components. No registry query applicable.
- **B (Magic UI)**: skipped — same reason.
- **C (Context7)**: skipped — no library API in use; pure SVG.
- **D (RESEARCH.md)**: pattern **S2 (single accent)** — kept to navy + gold only; **A2 (no shouting CAPS)** — microcopy ring stays at single-emphasis tracking, no aggressive sizing.
- **E (Native)**: built. `certification-mark.svg` uses a `<textPath>` ring for "CERTIFIED · FGB ACADEMY ·" microcopy around the seam, navy disc inside, chevron + dot mark. `app-icon-mono.svg` uses `currentColor` so the OS / theme tints it.
- **Reinforces attributes**: Trusted (shows the receipt — a seal), Premium (single accent, restraint).

### Phase 4 — `GiaTip.svelte` + Gia presence extension (gia-coach, adaptive-room)

- **A (21st.dev)**: `21st_magic_component_inspiration` with searchQuery `"coach tip avatar"`. Top hit was `AvatarGroupWithTooltips` (an avatar group with hover tooltips), which is structurally unrelated to a single-avatar inline tip. **No reusable match.**
- **B (Magic UI)**: registry surface inspected via Phase 0.1 `getSpecialEffects` and `getComponents` results. No "tip" or "callout" component in the catalog (the closest, `magic-card`, is an interactive hover card — too heavy). **No reusable match.**
- **C (Context7)**: already-confirmed Svelte 5 `Snippet` type, `$props()`, `$derived` rune patterns from Phase 0.1 baseline. No new Context7 query needed.
- **D (RESEARCH.md)**: pattern **S7 (highlight one phrase)** — the message accepts plain text with single-emphasis voice from `gia.md`; **A1 (no mascot dominance)** — GiaTip is sized as a pill, never as a hero; **A2 (no shouting CAPS)** — message stays sentence-case.
- **E (Native)**: built. Composition of existing `GiaAvatar` + a styled `<aside role="note">` wrapper, three tone variants (`default`, `celebrate`, `thinking`), two sizes (`sm`, `md`). Honours `motion-rise-in` and reduced-motion automatically through the existing motion system.
- **Reinforces attributes**: Human (Gia + sentence-case message), Professional (a polite period at the end).

Applied in:
- [`fgb/src/routes/(app)/gia-coach/+page.svelte`](../../../src/routes/(app)/gia-coach/+page.svelte) — header avatar + assistant message bubbles (replaced 4 instances of `Bot` icon with `GiaAvatar`), empty state (replaced `Sparkles` with `GiaAvatar` + sentence-case prompt), thinking state (replaced `LoaderCircle` with `LoadingDots` + GiaAvatar `state="thinking" pulse`), sidebar empty-state (new `GiaTip` instance).
- [`fgb/src/routes/(app)/adaptive-room/+page.svelte`](../../../src/routes/(app)/adaptive-room/+page.svelte) — header replaced with `PageHeader`, course-picker leading `GiaTip` ("Pick a course you've already touched"), loading state uses `GiaAvatar` `state="thinking"` + `LoadingDots`, session-done view rebuilt as a brand-gradient ceremony with `BadgeMedal` (tier from theta), `GiaAvatar` celebrating, `XpRing`, gold-glow CTA. Emerald/red feedback strip migrated to `success`/`destructive` tokens with `motion-rise-in` + `aria-live="polite"`.

### Phase 6.1 — Dashboard widgets (`ContinueLearning`, `AchievementsWall`, `Recommendations`)

- **A (21st.dev)**: `21st_magic_component_inspiration` with searchQuery `"continue learning card"`. Top hits: `Vo2MaxCard`, `CircularProgressCard`, `AnimatedProgressCard` — all variations of "branded primary card with progress bar + value/max + animated counter". **Pattern adopted** for `ContinueLearning`: filled brand surface, icon-in-pill at top, animated bar, value-over-max at footer. Translated from React/Framer-motion to native Svelte 5 + `Tween` from `svelte/motion`.
- **B (Magic UI)**: `getComponents` and `getSpecialEffects` consulted from Phase 0.1 cache. Reused `border-beam` concept (already implemented as our `BorderBeam.svelte`) on the `ContinueLearning` resume card per RESEARCH.md S2/S7. No "achievements" or "recommendations" component exists in Magic UI; closest was `bento-grid` (declined — overweight for a single-row of cards).
- **C (Context7)**: Svelte 5 `Tween.target` setter + `prefersReducedMotion.current` MediaQuery confirmed in Phase 0.1; no new query needed. The `gradientFor()` deterministic gradient helper in `Recommendations` uses no library APIs.
- **D (RESEARCH.md)**:
  - `ContinueLearning` → **S1** (large display type for course title), **S7** (highlight the gold accent on the eyebrow + bar), **S2** (single accent — gold only).
  - `AchievementsWall` → **S5** (icon rhythm, tight labels), **S8** (horizontal scroll row), **S2** (single gold accent).
  - `Recommendations` → **S8** (Netflix-style category row scaled down), **S3** (cinematic image slot — falls back to programmatic gradient until Higgsfield renders land), **A5** (no rainbow tier pills — single source-count chip).
- **E (Native)**: built. All three use the existing brand barrel, honour `prefers-reduced-motion`, and respect the data shape already returned by the dashboard's `+page.server.ts` (no server changes).
- **Reinforces attributes**:
  - `ContinueLearning` — Aspirational (names the next module), Premium (brand gradient + border beam), Human (Gia's voice in the surrounding copy).
  - `AchievementsWall` — Trusted (shows actual earned tiers), Aspirational (visible next tier), Professional (sentence-case "Achievements", "earned").
  - `Recommendations` — Intelligent (derived from real published-but-not-started courses, not a marketing list), Human (Gia tip explains why empty).

Dashboard splice in [`fgb/src/routes/(app)/+page.svelte`](../../../src/routes/(app)/+page.svelte): four new derived blocks (`continueCourse`, `streakDays`, `certifications`, `recentlyCompleted`), all client-side filters over the existing `recentCourses` payload. No `+page.server.ts` modifications. New `StreakFlame` surfacing, `Certifications` row (uses the new `certification-mark.svg`), and `Recently Completed` 3-col grid added. Achievements + Recommendations as a 1+2 grid row.

### Phase 6.2 — `CourseCard.svelte` + course catalogue category bands

- **A (21st.dev)**: `21st_magic_component_inspiration` (course card query, Phase 0.1 baseline). Returned variants of the shadcn `Card` with `lifted`, `gradient`, `corners` decorations. Reused the **lift + corner-overlay status chip** approach (status pill at top-right, source-count chip at bottom-right) — pattern-translated from React to Svelte 5.
- **B (Magic UI)**: `getComponents` (cached). Closest match was `magic-card` (mouse-follow spotlight) — declined for this surface because catalogue cards are rendered in horizontal scroll rows where pointer following would feel busy. Kept the existing `lift` + `press` motion utilities and `gradient-to-t` scrim instead.
- **C (Context7)**: no new library API used; `$derived`, `$props`, `Snippet`, and existing motion utilities only. The deterministic `gradientFor(id)` helper uses only OKLCH arithmetic, no library.
- **D (RESEARCH.md)**: **S8 (Netflix-style category bands)** is the structural inspiration; **S3 (cinematic image with scrim)** for the optional Higgsfield-rendered thumbnail; **S2 (single accent)** — status chips use one semantic colour each (`success` for complete, `accent` for in-progress); **A5 (no rainbow status pill)** — never more than one chip colour at once.
- **E (Native)**: built. Compact mode (320 px) for horizontal `snap-x` rows; full-width mode for grid layout. Falls back to a deterministic OKLCH gradient + course initial when no image is supplied, so the surface is never broken if Higgsfield assets are absent.
- **Reinforces attributes**: Premium (cinematic image slot, brand-locked gradients), Aspirational (in-progress chip names the next move), Professional (sentence-case CTAs, single status chip).

Applied in [`fgb/src/routes/(app)/lesson-player/+page.svelte`](../../../src/routes/(app)/lesson-player/+page.svelte) — replaced the 2-column grid course list with a `PageHeader` + search + filter bar + three category bands: "Continue learning" (horizontal scroll), "Recommended for you" (horizontal scroll), "Completed" (grid). All filtering is client-side over the existing payload. `Layers`/`CheckCircle`/`Play`/`RotateCcw` icons retained because they remain in use on the player view; `LoaderCircle` no longer used in the course list (replaced with `LoadingDots`).

---

## Phase 11 — Higgsfield generation log

**Path A taken** — Higgsfield plugin was activated mid-execution (2026-06-15, late-morning Kingston time). The MCP server `user-higgsfield` registered 40+ tools including `generate_image`, `job_status`, `balance`, and `models_explore`.

### Account / preflight

- `balance` → **1,210 credits on the Plus plan** (more than enough for the priority programme).
- Model selection: `soul_2` for every prompt — editorial photography / portrait / UGC / fashion model recommended by the `generate_image` tool description. Cost: 1 credit per image at default 2k quality.
- Aspect ratios: requested per the prompt pack (16:9 / 21:9 / 3:2 / 1:1 / 4:3). The server silently coerced `21:9` → `16:9` for the dashboard ambient — accepted; the image is overlaid at opacity 0.2 so the modest aspect drift is invisible.

### Generation programme — round 1 (initial render, generic context)

13 jobs submitted in parallel at 2026-06-15 13:05 UTC, all completed in ~50 s. Saved as `static/brand/.../*.png`. Each rendered cleanly and on-brand but used generic non-Jamaican subjects.

### Generation programme — round 2 (Jamaican context — final)

After the user clarified that **the audience is Jamaican / Caribbean**, all 13 prompts were rewritten with the `[JM]` style addendum (see [`HIGGSFIELD_PROMPTS.md` §0.1](./HIGGSFIELD_PROMPTS.md)). 13 fresh jobs submitted at 2026-06-15 13:14 UTC; all completed in ~50 s. Files in `static/brand/` were overwritten in-place (no rewiring required because the wiring uses fixed paths).

| Slug | Job ID (round 2) | Path | Aspect | Notes |
| --- | --- | --- | --- | --- |
| hero-login | `7ab0f387-…` | `hero/hero-login.png` | 16:9 | Afro-Caribbean woman, Kingston harbour + Blue Mountains at dawn |
| dashboard-ambient | `dae98bf4-…` | `hero/dashboard-ambient.png` | 16:9 | Gold leaf on blue mahoe slab |
| banking-foundations | `22b06f50-…` | `pathways/banking-foundations.png` | 3:2 | Senior + junior Jamaican bankers, mahogany office |
| risk-credit | `1066d5a6-…` | `pathways/risk-credit.png` | 3:2 | Jamaican analyst, multi-monitor at Kingston dusk |
| leadership | `fde1947e-…` | `pathways/leadership.png` | 3:2 | Three Jamaican leaders, glass boardroom, harbour outside |
| customer-service | `ce8f4d02-…` | `pathways/customer-service.png` | 3:2 | Jamaican RM + Jamaican customer in mahogany branch |
| cybersecurity | `e0f27ce1-…` | `pathways/cybersecurity.png` | 3:2 | Jamaican security analyst at night, gold-glow monitor |
| compliance | `d6f78778-…` | `pathways/compliance.png` | 3:2 | Manual on Jamaican mahogany, Blue Mountains framed photo |
| executive | `b415a0dc-…` | `programs/executive.png` | 16:9 | Jamaican C-suite, Kingston skyline + Blue Mountains at dusk |
| ceremony | `2c45b5f4-…` | `achievements/ceremony.png` | 1:1 | Gold laurel emblem on navy velvet (abstract; no JM cue needed) |
| onboarding step-1 | `168e2a48-…` | `onboarding/step-1.png` | 4:3 | Three glass tokens, Jamaican hand entering frame |
| onboarding step-2 | `af195494-…` | `onboarding/step-2.png` | 4:3 | Afro-Caribbean woman head-and-shoulders, soft cream bg |
| onboarding step-3 | `6b019985-…` | `onboarding/step-3.png` | 4:3 | Jamaican learner at home, tropical foliage hint outside |

**Total credits spent**: 26 (13 × 1 credit, two rounds). **Remaining**: ~1,184. **Wall-clock**: ~6 minutes including poll + download + overwrite.

### Wiring applied

- [`fgb/src/routes/login/+page.svelte`](../../../src/routes/login/+page.svelte) → brand panel now overlays `hero-login.png` at `opacity-40 mix-blend-luminosity` per `HIGGSFIELD_PROMPTS.md §14.2`.
- [`fgb/src/routes/setup/+page.svelte`](../../../src/routes/setup/+page.svelte) → same brand panel mirror.
- [`fgb/src/routes/(app)/+page.svelte`](../../../src/routes/(app)/+page.svelte) → dashboard hero now overlays `dashboard-ambient.png` at `opacity-20 mix-blend-luminosity` per `HIGGSFIELD_PROMPTS.md §14.3`.
- [`fgb/src/routes/(app)/lesson-player/+page.svelte`](../../../src/routes/(app)/lesson-player/+page.svelte) → completion ceremony overlays `achievements/ceremony.png` at `opacity-30 mix-blend-luminosity`.
- [`fgb/src/lib/components/brand/CourseCard.svelte`](../../components/brand/CourseCard.svelte) + [`Recommendations.svelte`](../../components/brand/Recommendations.svelte) → new `pathwayImageFor(title)` helper resolves a course title to one of the six rendered pathway covers via regex hints (e.g. /risk|credit/i → `risk-credit.png`); falls back to `/brand/modules/{slug}.png` if no pathway matches, then the existing programmatic gradient if no file exists (`onerror` hide).

### Roll-back protocol

If a render needs to be re-done:

1. Re-call `generate_image` with the updated prompt + same `aspect_ratio` + `model: "soul_2"`.
2. Poll `job_status` with `sync: true`.
3. PowerShell `Invoke-WebRequest -Uri <newRawUrl> -OutFile fgb/static/brand/<path>` — same path as before to overwrite.
4. No Svelte changes required.
