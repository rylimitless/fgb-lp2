# MCP_REVIEW.md — FGB Academy Enhancement Roadmap

> Review of where the available MCP ecosystem can continue improving the FGB Academy experience. This document separates what was implemented now from what should be scheduled next.

---

## Executive Summary

The strongest near-term MCP value is **not another visual reskin**. The platform now needs more interaction depth:

1. More varied assessment mechanics.
2. Search/navigation that feels like a modern SaaS command surface.
3. Activity motion that makes the product feel alive.
4. Higgsfield visuals that support assessments, credentials, and moments of reward.

This pass implements the first high-impact set: `matching`, `drag_sort`, `hotspot`, a command palette, animated activity feed, Higgsfield assessment/certificate assets, the first PremiumTable pass, and Gia orbiting insights.

---

## 1. Higgsfield

| Opportunity | Impact | Effort | Status |
| --- | --- | --- | --- |
| Hotspot diagrams for assessments | High | Low | Built this wave |
| Certificate background / credential art | High | Low | Built this wave |
| Per-course module covers | High | Medium | Workflow scaffold built (`scripts/course-cover-manifest.mjs`) |
| Achievement art variants | Medium | Low | Next |
| Gia thinking / celebrating 3D state variants | Medium | Medium | Next |
| 6-second landing / completion loops via `generate_video` | High | Medium | Deferred: requires final motion direction + credit approval |

### Recommendation

Keep Higgsfield as the source of product imagery, but maintain the rule from `HIGGSFIELD_PROMPTS.md`: Jamaican professional context, no imported stock-photo banking culture, no vacation/tropical clichés.

---

## 2. 21st.dev

| Opportunity | Impact | Effort | Status |
| --- | --- | --- | --- |
| Command palette / search overlay | High | Medium | Built this wave |
| Premium data table patterns for Audit Log / User Management / Repository | High | Medium | Audit Log + User Management on `PremiumTable`; Repository kept on its richer responsive table (per-column hiding + async loading state not modelled by `PremiumTable`) |
| Leaderboard / gamification card patterns | Medium | Low | Done enough for current dashboard |
| Toast / notification patterns | Medium | Medium | Built (`toast.svelte.ts` + `ToastViewport`); wired into Settings and User Management success paths |
| Settings / profile screens | Low | Medium | Built native profile/settings route |

### Recommendation

Use 21st.dev for **workflow surfaces**, not brand expression. Its React examples translate well structurally (command menus, tables, dashboards), but the visual language should remain FGB-specific.

---

## 3. Magic UI

| Opportunity | Impact | Effort | Status |
| --- | --- | --- | --- |
| Animated list for activity feed / notifications | High | Low | Built this wave |
| Orbiting circles around Gia / AI Coach | Medium | Low | Built (`OrbitingInsights`) |
| Marquee for featured courses | Medium | Low | Built on dashboard featured-learning strip |
| Shine/border effects on premium cards | Medium | Low | Already available via `BorderBeam` |
| Particles / animated grid / aurora text | High | Done | Already implemented |

### Recommendation

Magic UI should drive **moments and motion**: animated lists, transition cues, celebration effects, ambient depth. Avoid applying every effect everywhere; reserve high-motion effects for hero, completion, coach, and activity.

---

## 4. Context7

| Area | Validation |
| --- | --- |
| Svelte 5 | `$effect` cleanup, `bind:this` canvas, `use:` actions with IntersectionObserver |
| Chart.js | Radar chart already validated/working in Analytics and Dashboard |
| dnd-kit | `@dnd-kit-svelte` docs found, but the installed package is `@dnd-kit/svelte@0.5.0`; API maturity is uncertain |

### Recommendation

Do **not** build critical learning interactions on the current alpha DnD package. Use native HTML5 drag + keyboard buttons for ordering questions. Revisit a full DnD library upgrade later.

---

## 5. Top Items Implemented This Wave

- New question mechanics:
  - `matching`
  - `drag_sort`
  - `hotspot`
- Command palette (`Ctrl/Cmd+K`) for navigation and course search.
- Animated activity feed component.
- Higgsfield certificate background and hotspot diagram imagery.
- AI generation prompt extended so future generated courses can emit new question types.

---

## 6. Next Wave

1. **Premium admin data tables** via 21st.dev patterns: grouped filters, sticky columns, saved views.
2. **Gia orbiting insight panel** via Magic UI orbiting-circles pattern.
3. **Course-specific Higgsfield cover automation**: map every generated course title to a persisted visual.
4. **Question analytics**: first lightweight UI built in Lesson Player; full historical analytics still requires schema/events.
5. **Video moments**: 6-second completion and onboarding loops via Higgsfield `generate_video`.

---

_Last updated: 2026-06-15._
