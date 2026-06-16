# FGB Academy — Higgsfield Visual Prompt Pack

> This file is the production-ready prompt library for the Higgsfield plugin. Every prompt has been engineered to produce **a single consistent visual language** across the entire FGB Academy ecosystem.
>
> **How to use.** Open the Higgsfield plugin → paste a prompt verbatim → set aspect ratio per the "Format" line → render 3 variants → pick the strongest → save into `static/brand/<folder>/<slug>.webp` (always WebP for product UI, PNG only when transparent assets are required).

---

## 0 · The style spine (paste into every prompt)

> _"Editorial photography, FGB Academy brand: refined, premium financial-services aesthetic. **Cinematic single-source light from the upper-left, soft gold-warm fill.** Palette restricted to **deep navy (#00548e), warm gold (#d6c47e), cream, and graphite**. Shallow depth of field, 35mm full-frame lens, ISO 200. Subtle film grain. No text, no logos in the image. Composition leaves negative space on the left for UI overlays. Subjects are intelligent, calm, and adult — never overtly cheerful."_

This block is referenced below as **`[STYLE]`**. Always prepend it.

### 0.1 · Caribbean / Jamaican context addendum

> _"Caribbean / Jamaican professional context. Adult Afro-Caribbean Jamaican subjects (mid-30s to mid-50s, mixed gender). Setting: modern Kingston banking environment — sleek glass towers, mahogany and blue mahoe hardwood interiors, occasional glimpse of Kingston harbour or Blue Mountains through a window. Light retains the cinematic gold-warm upper-left rule. Wardrobe: tailored navy / charcoal suits, white or cream shirts. **AVOID: beach scenes, steel drums, dreadlocks-as-cliché, rum/cigars, casual or vacation imagery, tropical stereotypes.**"_

This block is referenced below as **`[JM]`**. Append it after `[STYLE]` for every prompt that contains a human subject or geographic context. Skip it for pure abstract macros where the cultural cue would be forced (e.g. §4 achievement laurel emblem). When the abstract still life can credibly carry a quiet Jamaican grounding (e.g. §5 compliance manual on Jamaican mahogany; §2 dashboard ambient with Jamaican blue mahoe slab), prefer the grounded version.

**Audience rationale.** FGB Academy is a learning ecosystem for the First Global Bank community, headquartered and operated in Jamaica. Generic banker imagery imported from US/UK financial-services stock libraries would feel imported and patronising. The mandate is to render bankers, leaders, customers, and analysts who look like the actual learners on the platform, in environments those learners recognise.

---

## 1 · Landing & login hero (1)

> **Format**: 16:9, 2560×1440, WebP, save to `static/brand/hero/hero-login.webp`
> **Use**: Right panel of the login screen, app marketing site.

```
[STYLE]
A composed mid-30s banking professional standing at a floor-to-ceiling window
looking thoughtfully out across a city at dawn. She holds a tablet showing
faint abstract data visualisations. Behind her, a softly out-of-focus modern
banking office in navy and warm wood. The light kisses her shoulder in gold.
She is mid-thought, not smiling, not stern. Negative space dominates the
upper-left third for the FGB Academy lockup.
```

---

## 2 · Dashboard ambient hero

> **Format**: 21:9, 2560×1080, WebP, `static/brand/hero/dashboard-ambient.webp`
> **Use**: Background of the dashboard hero greeting card.

```
[STYLE]
An abstract macro photograph of brushed navy metal meeting a single thread of
warm gold leaf, lit from upper-left. Subtle bokeh, the gold catches the light
in three or four discrete highlights. No subject — purely textural. The
composition reads as a thin horizontal band of warmth across a navy field.
Intended as a low-contrast background behind white text.
```

---

## 3 · Learning pathways (5 cards — one per pillar)

> **Format**: 3:2, 1500×1000, WebP, save to `static/brand/pathways/<slug>.webp`
> **Use**: Top of each course-catalogue category page; pathway cards.

Use the same `[STYLE]` block for all five. The subject swaps:

| Slug                  | Subject |
| --------------------- | ------- |
| `banking-foundations` | A senior banker reviewing a printed term sheet with a junior — two pairs of hands, the document is the hero, faces partially out of frame. |
| `risk-credit`         | A risk analyst at a multi-monitor desk, charts faintly visible, gold reflection off a coffee cup catches the eye. |
| `leadership`          | A small group of three leaders mid-discussion at a glass-walled table, one is speaking, two are listening — captured as a candid mid-conversation moment. |
| `customer-service`    | A relationship manager seated across from a customer, both leaning slightly forward — warmth in posture, not in expression. |
| `cybersecurity`       | A security analyst's workspace at night — a single monitor's gold-warm glow illuminates an architectural diagram of a banking network. |
| `compliance`          | An open compliance manual on a clean desk, a navy fountain pen and a single gold paperclip. No people; institutional calm. |

---

## 4 · Achievement screens

> **Format**: 1:1, 1080×1080, WebP, `static/brand/achievements/<slug>.webp`
> **Use**: Course-completion ceremony background, certificate hero.

```
[STYLE]
A single gold laurel-style emblem sitting on a deep navy velvet surface,
lit from upper-left. Soft golden particle bokeh drifts upward behind it.
The emblem fills 40% of the frame, centred low. Reads as quietly triumphant.
Format: square, intended as the backdrop for a centered "Course complete"
ceremony card.
```

Pair with the in-app `BadgeMedal` SVG + `Confetti` motion overlay — never replace either.

---

## 5 · Certificates

> **Format**: A4 landscape, 3508×2480, PNG (transparent margin retained), `static/brand/certificates/certificate-base.png`
> **Use**: Composited into the printable certificate template.

```
[STYLE]
A clean, formal certificate background: rich cream paper texture with a very
faint navy guilloché border at 6% opacity. Upper-right corner: a single
embossed gold seal element (purely decorative, no readable text). The
remaining 80% of the surface is empty — designed to receive layered text,
the FGB Academy lockup, and a signature line.
```

---

## 6 · Executive learning programs

> **Format**: 16:9, 2560×1440, WebP, `static/brand/programs/executive.webp`
> **Use**: Hero of executive-tier program pages.

```
[STYLE]
A reflective C-suite executive seated alone in an empty boardroom at dusk,
back-three-quarters to camera, looking out a tall window onto a city skyline.
The city lights are warm gold pinpricks against navy night. Their suit reads
as deep charcoal, almost navy. Mood: deliberate, weighty, calm. Wide framing.
```

---

## 7 · Onboarding journey (3-frame storyboard)

> **Format**: 4:3, 1600×1200, WebP, `static/brand/onboarding/step-{1,2,3}.webp`
> **Use**: The 3-step onboarding flow's right-panel illustrations.

```
Frame 1 — "Choose your path"
[STYLE] A clean still life: three glass tokens on a navy linen surface,
each containing a tiny gold symbol — a chevron, a circle, and a square.
One token is gently held between thumb and index finger entering frame from
the right. Reads as choosing without ceremony.

Frame 2 — "Meet Gia"
[STYLE] A woman in her early 30s in a navy blazer and white blouse, head and
shoulders, looking warmly but professionally toward camera. Background is
soft cream gradient. Her halo is implied by the gold-warm rim light from
upper-left. Designed to be composited beside the Gia SVG avatar.

Frame 3 — "Start learning"
[STYLE] Over-the-shoulder shot of a learner reading on a tablet at a kitchen
island, morning light from a window upper-left. The tablet content is
abstract — we read posture and softness, not detail. Coffee cup foreground.
```

---

## 8 · Module covers (procedurally generated)

> **Format**: 16:9, 1600×900, WebP, `static/brand/modules/<slug>.webp`
> **Use**: Course card thumbnails in the catalog.

A generic template the content team can adapt — keeps every module on-brand without manual art-direction per course:

```
[STYLE]
An abstract macro photograph composed for a 16:9 module card. The image must
read instantly as one of these themes: {{THEME}}. Avoid people unless the
theme demands one. Composition pulls the eye toward the right-hand third so
the left-hand third can carry the module title in white type.

THEME options (pick one and substitute):
- "documents and continuity"    → layered paper edges, gold paperclip
- "data and pattern"            → macro of a navy graph paper grid, single
                                   gold marker dot
- "trust and exchange"          → two hands lightly clasping at the wrist
- "vigilance"                   → an empty leather chair in a quiet hallway,
                                   single gold lamp lit
- "growth"                      → a young plant in a navy ceramic pot,
                                   shallow focus, dawn light
- "speed"                       → blurred motion of a navy carbon-fibre weave,
                                   one gold thread reading sharp
```

---

## 9 · Empty-state companions

> **Format**: 1:1, 600×600, WebP transparent if possible, `static/brand/empty/<slug>.webp`
> **Use**: Behind `EmptyState` cards when Gia alone isn't enough.

```
[STYLE]
A small, quiet still life: a single hardcover book lying closed on a navy
surface, a gold ribbon bookmark spilling out, dust motes catching upper-left
light. Format square, subject 55% of frame, generous negative space.
Use case: "no documents yet" / "no courses yet" empty states.
```

---

## 10 · Motion / generative video (optional)

If the Higgsfield video module is available, generate **6-second loops** at 1920×1080, 24fps, mp4 (h.264), saved to `static/brand/motion/`.

### 10.1 `welcome-loop.mp4` — login background

```
[STYLE]
A 6-second loop: extremely slow lateral camera glide across a navy textured
surface; gold dust motes drift slowly upward; one single brighter gold
particle crosses the frame at second 2. No subject. Seamless loop — first
and last frames identical. Use as a 30% opacity background behind the login
form.
```

### 10.2 `achievement-burst.mp4` — completion ceremony

```
[STYLE]
A 3-second one-shot: a single gold leaf-shaped particle gently unfolds into
a soft expanding halo of warm light against a deep navy background. No
subject. Ends on a held glow. Intended to play once on course-completion.
```

### 10.3 `gia-thinking-loop.mp4` — coach ambient

```
[STYLE]
A 4-second loop: the gold halo of the Gia avatar gently pulses — bright at
beat 1, soft at beat 3, back to bright at beat 4. Transparent background.
1:1 square. Use only behind the Gia avatar component when she is processing.
```

---

## 11 · Asset slot manifest

Once generated, drop each file into the path below. The product code already references these paths — no further changes needed.

```
static/brand/
├── hero/
│   ├── hero-login.webp
│   └── dashboard-ambient.webp
├── pathways/
│   ├── banking-foundations.webp
│   ├── risk-credit.webp
│   ├── leadership.webp
│   ├── customer-service.webp
│   ├── cybersecurity.webp
│   └── compliance.webp
├── achievements/
│   └── ceremony.webp
├── certificates/
│   └── certificate-base.png
├── programs/
│   └── executive.webp
├── onboarding/
│   ├── step-1.webp
│   ├── step-2.webp
│   └── step-3.webp
├── modules/                       # one per published course slug
├── empty/
│   └── library.webp
├── motion/                        # optional loops
│   ├── welcome-loop.mp4
│   ├── achievement-burst.mp4
│   └── gia-thinking-loop.mp4
└── gia/
    ├── gia-idle.png               # already generated
    ├── gia-thinking.png           # already generated
    └── gia-celebrating.png        # already generated
```

---

## 12 · Quality control

Reject any Higgsfield output that:

* Adds a third hero colour (anything outside navy / gold / cream / graphite).
* Includes legible text in the image.
* Shows a subject grinning or wearing overtly bright workwear.
* Has more than one obvious light source.
* Cannot accommodate a left-side text overlay without visual collision.

Regenerate. The cost of one extra render is lower than the cost of off-brand imagery shipping.

---

## 13 · Module-cover slugs (procedurally generated per course)

These slugs are derived from courses currently published in the system. **The content team should render one image per slug, using the `THEME` template in §8.** Drop each result at `static/brand/modules/{slug}.webp`. The `CourseCard` component will pick it up automatically.

**Live snapshot rule.** The slug list below is illustrative — the actual list at any moment can be fetched at run time by:

```bash
curl --cookie 'session=...' http://localhost:5555/api/courses \
  | jq -r '.[] | select(.status == "published") | .title' \
  | sed -E 's/[^a-zA-Z0-9]+/-/g' | tr '[:upper:]' '[:lower:]' | sed -E 's/^-|-$//g'
```

Illustrative slug → theme pairings (substitute live slugs at execution time):

* `data-protection-essentials` → theme: `documents and continuity`
* `risk-credit-foundations` → theme: `data and pattern`
* `leadership-fundamentals` → theme: `trust and exchange`
* `customer-service-mastery` → theme: `trust and exchange`
* `cybersecurity-awareness` → theme: `vigilance`
* `compliance-fundamentals` → theme: `documents and continuity`
* `aml-foundations` → theme: `vigilance`
* `branch-operations` → theme: `growth`
* `digital-banking-101` → theme: `speed`

For each slug:

1. Open the Higgsfield plugin.
2. Paste the §8 procedural template with `THEME` substituted.
3. Render 3 variants at 1600×900.
4. Pick the strongest; export as WebP, q=80.
5. Save to `fgb/static/brand/modules/{slug}.webp`.

No code changes required. The image will appear in the course catalogue and on the `Recommendations` row on next page reload.

---

## 14 · Wiring appendix

This section documents exactly how the product code consumes the assets above, so the content team can ship renders without engineering involvement.

### 14.1 `CourseCard` thumbnail resolver

The Svelte 5 source in [`src/lib/components/brand/CourseCard.svelte`](../components/brand/CourseCard.svelte) accepts an optional `image` prop. The catalogue does not pass it by default — it relies on a deterministic gradient fallback. To wire the Higgsfield-rendered thumbnails in, no Svelte changes are required: the catalogue's parent component should derive an image URL from a convention. The simplest convention is to slugify the title and look in `static/brand/modules/`:

```svelte
<script lang="ts">
    // Anywhere a course list is rendered:
    function slugify(title: string): string {
        return title
            .toLowerCase()
            .replace(/[^a-z0-9]+/g, "-")
            .replace(/^-|-$/g, "");
    }
    function moduleImage(course: { title: string }): string | undefined {
        // Convention: static/brand/modules/{slug}.webp
        // If the file doesn't exist, the <img> will 404 silently and the
        // CourseCard's programmatic gradient fallback handles the visual.
        return `/brand/modules/${slugify(course.title)}.webp`;
    }
</script>

<CourseCard {course} image={moduleImage(course)} />
```

For a guaranteed-no-broken-image behaviour, gate the image on a server-side fetch (out of scope for this Phase 11; the product currently uses the safe gradient fallback for any missing thumbnail).

### 14.2 Login hero

In [`src/routes/login/+page.svelte`](../../routes/login/+page.svelte) the brand panel currently uses `AnimatedGradient` + `Spotlight` over `brand-gradient`. To use `hero-login.webp` as the panel background, prepend the panel `<aside>` with:

```svelte
<img
    src="/brand/hero/hero-login.webp"
    alt=""
    aria-hidden="true"
    class="absolute inset-0 size-full object-cover opacity-40 mix-blend-luminosity"
    fetchpriority="high"
/>
```

Inside the existing brand-gradient `<aside>`, above `<AnimatedGradient />`. The mix-blend-luminosity desaturates the image into the gradient — keeps brand restraint while landing the cinematic.

### 14.3 Dashboard ambient

In [`src/routes/(app)/+page.svelte`](../../routes/(app)/+page.svelte), inside the hero `<section>` that already contains `<Spotlight />`:

```svelte
<img
    src="/brand/hero/dashboard-ambient.webp"
    alt=""
    aria-hidden="true"
    class="absolute inset-0 size-full object-cover opacity-20"
    fetchpriority="low"
    loading="lazy"
/>
```

Same restraint — kept low opacity so the personalised greeting (the actual content) reads first.

### 14.4 Pathway category headers

When the catalogue grows enough to warrant category landing pages, drop one of `static/brand/pathways/{slug}.webp` above each band as a 21:9 banner with the band title overlaid in `text-display-md text-primary-foreground`.

### 14.5 Onboarding storyboard

Reserved for a future first-run experience. The three frames map 1:1 to a three-step wizard (Choose path / Meet Gia / Start learning) and are referenced in [`RESEARCH.md`](./RESEARCH.md) pattern S4.

---

_Last updated: 2026-06-15. Owner: Design System. Higgsfield MCP status at time of write: **not registered** in this sandbox — prompts above ship as a manual runbook; sections 13–14 are pre-built so a single Higgsfield session populates the entire visual programme._

