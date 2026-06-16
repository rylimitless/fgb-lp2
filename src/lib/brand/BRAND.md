# FGB Academy — Brand Bible

> A complete sub-brand for the FGB learning ecosystem. This file is the single source of truth for the platform's name, story, personality, look, motion, and voice. The companion file [`gia.md`](./gia.md) governs the platform mascot specifically.

---

## 1 · The decision: why **FGB Academy**

We evaluated five candidate names — _Academy, Elevate, Nexus, Learn, Growth Hub_ — against five criteria: clarity, professional weight, longevity, distinctiveness in financial services, and how naturally it pairs with the existing FGB brand. The result:

| Name             | Clarity | Authority | Longevity | Distinct in BFSI | Pairs with FGB | Score |
| ---------------- | :-----: | :-------: | :-------: | :--------------: | :------------: | :---: |
| **FGB Academy**  |    5    |     5     |     5     |        4         |       5        | **24** |
| FGB Elevate      |    3    |     4     |     3     |        3         |       4        |  17   |
| FGB Nexus        |    2    |     4     |     3     |        2         |       3        |  14   |
| FGB Learn        |    5    |     3     |     4     |        3         |       4        |  19   |
| FGB Growth Hub   |    3    |     3     |     2     |        3         |       3        |  14   |

**Academy** wins because it carries the weight FGB needs (this isn't a side-project — it's an institutional learning platform), it is timeless (we do not have to rebrand it in 2030), and it is the word an executive would actually use in a board paper without flinching. _Elevate_ and _Nexus_ feel like growth-stage SaaS spinouts; _Learn_ and _Growth Hub_ feel like internal projects. **FGB Academy** feels like an institution.

The name is already in production across the codebase (see `static/brand/academy-lockup.svg`, `app.html`, `gia.md`). We are confirming and codifying it — not introducing it.

---

## 2 · Brand story

> **At FGB, the best decisions are made by the best-prepared people.** FGB Academy exists to make every employee, leader, partner, and customer one of those people — through learning that is precise, personal, and trusted.

Three sentences. Memorise them.

1. Banking moves at the speed of trust. Trust is built by people who know what they are doing.
2. FGB Academy is how FGB invests in that knowledge — at scale, in context, on demand.
3. It is not a training portal. It is the bank's institutional memory, rendered as a learning experience.

### The promise

> _Every learner leaves with a measurable upgrade — in skill, judgement, or compliance — that the institution can point to._

Every screen, every interaction, every Higgsfield image we generate should reinforce that single promise.

---

## 3 · Brand personality

A 5-axis dial (Coach, not cheerleader. Mentor, not mascot.):

| Axis             | Far left           | **Where FGB Academy sits** | Far right          |
| ---------------- | ------------------ | :------------------------: | ------------------ |
| Tone             | Playful            |   **Composed**             | Severe             |
| Pace             | Frantic            |   **Steady**               | Glacial            |
| Authority        | Hesitant           |   **Earned**               | Lecturing          |
| Aesthetic        | Decorative         |   **Refined**              | Brutalist          |
| Emotion          | Saccharine         |   **Warm**                 | Cold               |

### Voice — five rules

1. **Be specific.** "3 modules left — about 18 minutes" beats "almost done!".
2. **Be short.** If a sentence runs more than 18 words, split it.
3. **Cite the source.** When in doubt, point at the document the learner uploaded.
4. **Frame setbacks as data.** Never as failures.
5. **Never speak for the bank.** Coach the learner; let the bank speak for itself elsewhere.

Detailed voice rules for the Gia persona live in [`gia.md`](./gia.md). Anything written by a human marketer should also follow the five rules above.

### Seven attributes — operationalised

The master prompt asked the brand to feel **Premium, Intelligent, Aspirational, Trusted, Innovative, Human, Professional**. Each one is delivered by an enforceable copy rule, not by an adjective in a slide:

| Attribute       | What it forbids                                                          | What it demands                                                                                            |
| --------------- | ------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------- |
| **Premium**     | Stock-photo phrasing ("Unlock your potential!"). Filler verbs.           | One sentence, one job. Tight tracking on display type. A single accent — never two.                        |
| **Intelligent** | Vague claims ("AI-powered"). Generic encouragement.                      | Cite the source — the document, the module, the score — every time. Reference numbers the learner can verify. |
| **Aspirational**| Promising effortless mastery. "10 minutes to expert" claims.             | Name the next concrete milestone: "_one more module to silver_". Never the past — the next step.            |
| **Trusted**     | "We believe…", "Our commitment to…" Marketing-speak in product surfaces. | Show the receipt — last reviewed date on policies, source documents on coach answers, audit trail visible. |
| **Innovative**  | Calling something "revolutionary". Buzzword soup.                        | Demonstrate via behaviour: an adaptive question that actually adapts, a generated course that cites pages. |
| **Human**       | Speaking on behalf of the bank ("FGB thinks you should…"). Robotic tone. | First-person _learner_ perspective ("you", "your"). Acknowledge difficulty in plain language.              |
| **Professional**| Emoji bursts. Memes. "lol", "u", informal abbreviations.                 | Contractions OK. Sentence case for CTAs. A polite period at the end of micro-copy.                         |

Cross-reference: every component built in this transformation is required (per the master prompt's MCP gate) to log which of these attributes it reinforces. See [`MCP_LOG.md`](./MCP_LOG.md) per-component entries.

---

## 4 · Visual identity

### 4.1 Wordmark

Two lock-ups, both in `static/brand/`:

* **Mark + Academy lockup** (`academy-lockup.svg`) — the canonical horizontal logo. Used in the app header, login, exports, certificates, and OG cards.
* **Mark only** (`academy-mark.svg`) — used at < 96 px, as an avatar, or when paired with a screen title that already says "FGB Academy".

### 4.2 Mark anatomy

A stylised **A** built from two converging strokes (the FGB chevron) crossed with a single **gold dot** — a quiet anniversary reference, a fixed-income coupon, an underline. The mark is currentColor + accent, so it inherits navy in light mode and white in dark mode without re-export.

```svg
<path d="M 12 54 L 32 12 L 52 54" stroke="currentColor" stroke-width="7" />
<circle cx="32" cy="40" r="4.5" fill="#d6c47e" />
```

### 4.3 Colour

| Token        | Light                 | Dark                  | When to use                                |
| ------------ | --------------------- | --------------------- | ------------------------------------------ |
| `--primary`  | FGB navy `#00548e`    | Lifted navy           | Brand surfaces, primary CTAs, focus rings  |
| `--accent`   | FGB gold `#d6c47e`    | Lifted gold           | Hero accents, badges, XP, the mark crossbar |
| `--success`  | Forest green          | Lifted green          | Completion, approved, correct              |
| `--warning`  | Amber                 | Lifted amber          | Pending review, expiring, low streak       |
| `--info`     | Blue                  | Lifted blue           | Tips, hints, "thinking" states             |
| `--destructive` | Coral red          | Lifted coral red      | Errors, deletes — never punishment         |
| `--streak`   | Warm orange           | Warm orange           | Streak flame only                          |
| `--xp`       | Gold                  | Gold                  | XP rings, awarded badges only              |

Full token table lives in `src/routes/layout.css`. Use semantic tokens — never raw hex — in component code.

### 4.4 Typography

* **Family**: Inter Variable (`@fontsource-variable/inter`) — only.
* **Display ramp**: `.text-display-2xl` → `.text-display-md` (declared in `layout.css`). Used for hero numbers, ceremony titles, and large quoted copy.
* **Numerals**: `.tabular` for any digit that ticks (timers, counters, scores).

### 4.5 Radius & elevation

* `--radius` 0.625rem; tokens `--radius-sm` through `--radius-3xl` derived.
* Three shadows — `sm`, `md`, `lg` — navy-tinted in light, deep ink in dark.
* `--shadow-glow` is reserved for **celebration only**: badges, completion, achievement unlocks. Don't sprinkle it.

### 4.6 Imagery (Higgsfield)

See [`HIGGSFIELD_PROMPTS.md`](./HIGGSFIELD_PROMPTS.md) for the full prompt pack. Four rules:

1. **One light source, always from the upper-left.** Cinematic, soft, gold-warm.
2. **Navy and gold dominate; neutrals fill.** Never a third hero colour.
3. **Always one human in frame** — even on conceptual covers — because learning is human.
4. **The human is Jamaican.** FGB Academy is built for the First Global Bank community in Jamaica. Every figure on every surface is an Afro-Caribbean Jamaican adult in a recognisable Kingston banking environment (mahogany / blue mahoe interiors, glimpses of Kingston harbour or the Blue Mountains). Generic imported stock imagery is forbidden. Tropical clichés (beach, steel drums, vacation tropes) are also forbidden. See `HIGGSFIELD_PROMPTS.md §0.1` for the operationalised `[JM]` style addendum that ships with every prompt.

---

## 5 · Motion identity

Motion is part of the brand, not decoration.

* **Easings**: `--ease-out-quart`, `--ease-out-quint`, `--ease-spring`, `--ease-emphasized` (in `motion.css`).
* **Durations**: `120ms` (fast), `200ms` (base), `320ms` (slow), `720ms` (ceremony).
* **Stagger**: `motion-stagger-1` through `motion-stagger-6` for cascading lists.
* **Lift**: cards rise 2 px and gain `--shadow-lg` on hover. Press depresses 1 px.
* **Glow**: gold halo, used only on celebrations and active Gia states.
* **Reduced motion**: every motion utility is disabled under `prefers-reduced-motion: reduce`. Test it.

---

## 6 · Mascot — Gia

Gia is the single character users meet across the Academy. She is **not** the brand, she is the voice of the brand. Full guide: [`gia.md`](./gia.md).

Quick rules:
* Gia appears in the **app**, never on **marketing collateral that lives outside the product**.
* Gia uses three states only: `idle` (default), `thinking` (loading / generating), `celebrating` (completion).
* Gia never speaks on behalf of FGB the institution.

---

## 7 · Where the brand appears

| Surface                | Mark               | Wordmark | Gia        | Hero gradient |
| ---------------------- | ------------------ | -------- | ---------- | ------------- |
| Login                  | ✓ (lockup, large)  | ✓         | ✗          | ✓             |
| App header             | ✓ (lockup, small)  | ✓         | ✓ (idle)   | ✗             |
| Dashboard hero         | ✗                  | ✗         | ✓ (idle)   | ✓ (soft)      |
| Empty states           | ✗                  | ✗         | ✓ (idle)   | ✗             |
| Course completion      | ✗                  | ✗         | ✓ (celebrating) | ✓ (full)  |
| Certificates           | ✓ (lockup)         | ✓         | ✗          | ✗             |
| Admin (analytics, etc) | ✗                  | ✗         | ✗          | ✗             |
| OG card / social       | ✓ (lockup)         | ✓         | ✓ (idle)   | ✓             |

---

## 8 · What we will not do

* No third hero colour.
* No emoji bursts in copy (🎉🔥💯).
* No cartoon mascots beyond Gia.
* No stock photography in product UI.
* No screen with more than one primary CTA.
* No animation longer than 720 ms in the product surface.
* No gamification language that infantilises ("You crushed it, champ!").
* No mention of competitors, even by allusion.

---

## 9 · Living documents

| File                                | Owns                                            |
| ----------------------------------- | ----------------------------------------------- |
| `src/lib/brand/BRAND.md` (this)     | Name, story, personality, visual & motion identity |
| `src/lib/brand/gia.md`              | Mascot voice, states, and surfaces              |
| `src/lib/brand/HIGGSFIELD_PROMPTS.md` | Image-generation prompt pack                  |
| `src/routes/layout.css`             | Implemented design tokens                       |
| `src/lib/styles/motion.css`         | Implemented motion tokens                       |
| `src/lib/components/brand/*`        | Brand-aware Svelte components                   |
| `static/brand/*`                    | Exported brand assets (SVG, PNG, OG)            |

When any of these change, update this file's "Last reviewed" line below.

---

_Last reviewed: 2026-06-15 — initial codification._
