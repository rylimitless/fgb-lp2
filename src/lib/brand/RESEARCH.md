# RESEARCH.md — Competitive UI/UX Research

> Phase 0.2 deliverable. Distilled from a Cursor IDE Browser session on 2026-06-15. Screenshots captured into [`fgb/static/research/`](../../../static/research/). Used as a reference throughout the FGB Academy LXP master transformation; each new component's MCP_LOG entry cites the patterns it draws from.

---

## 0 · How to read this doc

Each pattern below is tagged with **Steal** or **Avoid**, followed by the source product, the screenshot reference, and the one-line takeaway. The "Apply to" line tells you which new FGB Academy surface should pick this up.

---

## 1 · Patterns to steal

### S1 — Massive editorial display type for hero
- **Source**: Coursera for Business — [`coursera-business.png`](../../../static/research/coursera-business.png), Linear — [`linear-landing-product.png`](../../../static/research/linear-landing-product.png)
- Coursera's hero packs a 5-word headline into 7 lines of 64 px bold serif, taking up the left half of the viewport. Linear pushes 4xl-5xl bold sans on dark background with tight tracking. Both let the typography do the heavy lifting; no decorative chrome.
- **Apply to**: existing dashboard hero (push the greeting display type one step bigger); new login + setup brand panels (anchor the marketing copy with a true display-2xl headline, not the current display-lg).

### S2 — Single accent over duotone everywhere else
- **Source**: Linear — gold star on a deep-neutral UI; Stripe — purple over near-white. MasterClass — coral pink over editorial black.
- Premium platforms restrict themselves to **one** accent. The rest is two-tone (background + foreground). Color noise is the most reliable tell of a corporate intranet.
- **Apply to**: the global token migration sweep (Phase 7b). Strip every `emerald/blue/amber/rose/cyan/violet/sky` utility from admin pages and replace with one of `primary` / `accent` / `success` / `info` / `warning` / `destructive` from FGB tokens.

### S3 — Cinematic side-portrait beside the hero
- **Source**: Coursera for Business — full-bleed portrait fills the right half. MasterClass — stacked masters' portraits.
- The portrait says "real person, real expertise, real outcomes" without any caption. The face faces forward, the photograph is shallow-DOF and lit warm.
- **Apply to**: login + setup brand panels (drop a single editorial portrait — produced via `HIGGSFIELD_PROMPTS.md §1` — behind the gradient panel content). Course catalog category headers (a single portrait per category).

### S4 — Role-pick onboarding cards
- **Source**: Khan Academy — [`khan-academy-landing.png`](../../../static/research/khan-academy-landing.png): "I'm a learner / I'm a teacher / I'm a parent" stacked cards with a right-chevron and no other affordance.
- The cards are deliberately oversized (h-16+) with high tap targets. One choice per row. Strong precedent for first-run experiences.
- **Status**: deferred. Setup is now premium and brand-aligned, but a full role-pick onboarding wizard would be a new product flow rather than a visual enhancement.

### S5 — Sidebar with grouped sections and tiny iconography
- **Source**: Linear — [`linear-landing-product.png`](../../../static/research/linear-landing-product.png): "Inbox / My issues / Reviews / Pulse / Workspace / Initiatives / Projects / More" — each prefixed by a 14 px monochrome icon. No accent colors on the sidebar.
- The icons act as a quiet rhythm; the labels carry the meaning. Active state is a faint highlight, never a coloured pill.
- **Apply to**: any future sidebar (FGB header tabs already follow this principle; reinforce in admin pages). Also use this restraint as a pattern for the dashboard quick-action grid.

### S6 — Live data signal in the hero
- **Source**: Stripe — [`stripe-landing.png`](../../../static/research/stripe-landing.png): "Global GDP running on Stripe: 1.66451573%" small-print line above the hero.
- A precise running number anchors the brand as a serious operator. Not a marketing claim — a *measurement*.
- **Applied**: landing hero now includes a live-data style trust signal ("4,200+ FGB learners ready for the next capability sprint"). Dashboard continues to use user-specific learning metrics.

### S7 — Two-tone copy: highlight one phrase in the brand colour
- **Source**: Stripe — "Financial infrastructure to **grow your revenue**" (purple verb phrase). Linear — "system for **teams and agents**" (subtle weight shift).
- The two-tone is a wayfinding device — reader lands on the most important phrase first.
- **Apply to**: existing login pillar copy (already uses `text-accent` on key word — confirm and propagate to setup + dashboard greeting subhead).

### S8 — Category-banded catalog (Netflix-style rows)
- **Source**: MasterClass — [`masterclass-catalog.png`](../../../static/research/masterclass-catalog.png): horizontal rows of portrait thumbnails, one row per category, each row independently scrollable.
- The pattern is so dominant in education + content products that learners expect it. Anything else feels like a database table.
- **Apply to**: Phase 6.2 — `lesson-player` course list. Replace the 2-column grid with category bands ("Continue learning", "Recommended for you", "All courses"), each as a horizontally scrollable row of `CourseCard`s.

### S9 — Centered minimal login with social pile-up
- **Source**: Notion — [`notion-landing.png`](../../../static/research/notion-landing.png): brand mark, headline, email field, "Continue" primary, then 5 stacked auth options (Google / Apple / Microsoft / Passkey / SSO).
- Each auth option is a card with icon + label. No password-first emphasis.
- **Apply to**: future enterprise SSO addition to FGB login. Recorded for the roadmap section of `DELIVERY.md` v2 (current FGB login still uses email + password — no auth backend change in scope).

### S10 — Trust strip below the hero
- **Source**: Stripe — Amazon / NVIDIA / Ford / Coinbase / Google logos in a single horizontal strip.
- One row of grayscale customer logos under the hero is the fastest credibility signal in B2B SaaS.
- **Applied**: the same trust-signal pattern is expressed in the landing hero with the FGB learner count; department logo strip deferred until real department/branch data exists.

---

## 2 · Patterns to avoid

### A1 — Mascot-dominant hero
- **Source**: Duolingo — [`duolingo-landing.png`](../../../static/research/duolingo-landing.png): cartoon mascot + characters fill the whole hero.
- For Duolingo it works — they sell joyful daily streaks to consumers. For a financial-services institution, "cartoon mascot first" reads as juvenile and undercuts the brand's authority.
- **Why we resist**: Gia is the mascot *of the product*, not of the platform's marketing. She lives inside the experience (header, empty states, completion ceremony), never as a hero subject on login or the dashboard hero. Already codified in [`gia.md` §5](./gia.md).

### A2 — All-caps body copy
- **Source**: Duolingo — "I ALREADY HAVE AN ACCOUNT", "GET STARTED". MasterClass — "LEARN FROM THE BEST".
- All-caps reduces scan speed and reads as shouting. Acceptable for ≤2-word labels (badges, eyebrows, pill chips) but never for full sentences or CTAs longer than 2 words.
- **Why we resist**: FGB Academy uses sentence case for all CTAs and uppercase only for `tracking-wider` eyebrow text. Already enforced in `BrandLogo` and `gia.md` voice rules.

### A3 — Cookie / consent overlay covering the hero
- **Source**: Khan Academy — [`khan-academy-landing.png`](../../../static/research/khan-academy-landing.png): cookie modal covers 35% of the viewport on first load.
- The first impression is a legal disclaimer instead of the product.
- **Why we resist**: FGB Academy is an internal-facing platform with implicit consent via employment terms — no consent modal exists, and we will not add one.

### A4 — Five social-login options stacked
- **Source**: Notion — [`notion-landing.png`](../../../static/research/notion-landing.png): Google / Apple / Microsoft / Passkey / SSO in a 3-then-2 grid.
- For consumer SaaS this is necessary. For an internal bank platform with a single SSO surface, this would inflate cognitive load on first sign-in.
- **Why we resist**: FGB Academy uses email + password (current) and will eventually add a single SSO option. Never multi-provider.

### A5 — Rainbow status indicators
- **Source**: not in the captures, but a common pattern across many SaaS dashboards (and present in FGB Academy's own admin pages today — emerald + amber + cyan + violet badges).
- Six accent colours in a single table reads as a debug console. Premium dashboards (Linear, Stripe) use ≤2 hues per surface.
- **Why we resist**: Phase 7b (token migration) explicitly collapses every existing rainbow utility to FGB semantic tokens.

---

## 3 · Component-pattern crosswalk

For each new component being built in this transformation, the source patterns it should draw from:

| New component | Steal | Avoid |
| --- | --- | --- |
| `CourseCard` (Phase 6.2) | S8 (Netflix rows), S2 (single accent), S3 (cinematic image) | A2 (no all-caps title), A5 (no rainbow status pill) |
| `ContinueLearning` (Phase 6.1) | S1 (large display type for course title), S7 (highlight the % left) | A1 (no Gia avatar in the hero) |
| `AchievementsWall` (Phase 6.1) | S5 (icon rhythm, tight labels), S2 (single gold accent) | A2 (no shouting CAPS) |
| `Recommendations` (Phase 6.1) | S8 (row pattern, smaller scale than catalog) | A5 (no rainbow tier pills) |
| `AnswerFeedback` (Phase 6.4) | S2 (one semantic accent per state — green or red, never both) | A2 (no all-caps "CORRECT!") |
| `GiaTip` (Phase 4) | S7 (highlight one phrase) | A1 (Gia is supporting, never dominant) |
| Login + setup brand panel | S1, S3, S7, S10 | A1, A2 |
| Admin tables (Phase 7) | S5 | A5 |

---

## 4 · Method notes

- All captures taken at default viewport (~1024×768 effective) on the cursor-ide-browser MCP, 2026-06-15.
- Snapshots and screenshots stored alongside this doc for traceability. Remove via `git rm` once the components are shipped if size is a concern (each PNG is ~150–250 KB).
- Live URLs visited: linear.app, stripe.com, duolingo.com, masterclass.com, notion.so, coursera.org/business, khanacademy.org.
- Not visited (gated, would have required login): Coursera Enterprise dashboard, Headspace member home. Captured the public marketing surface as the closest available proxy.

---

_Last updated: 2026-06-15. Owner: Design System._
