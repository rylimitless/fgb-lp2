package ai

import (
	"encoding/json"
	database "fgb-lp/database/queries"
	"fgb-lp/embeddings"
	"fgb-lp/middlewares"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

// ---- System prompts ----

// Step 1: Course Outliner — produces Markdown, not JSON.
const coursePlanSystemPrompt = `You are an expert instructional designer. Given source material and a course topic, your job is to produce a high-level course plan consisting of module titles and descriptions.

CRITICAL RULES:
1. Structure your output EXACTLY like the Markdown example below. Do NOT use JSON, and do not write any introductory or concluding conversational filler.
2. The number of modules MUST be proportional to the source material's actual size and depth. Use this scale:
   - Under 1,000 chars of source → 1 module
   - 1,000–8,000 chars → 2-3 modules
   - 8,000–30,000 chars → 4-6 modules
   - 30,000–80,000 chars → 7-12 modules
   - 80,000–200,000 chars → 13-20 modules
   - Over 200,000 chars → up to 25 modules
   Each module should target ~2,000-4,000 words of finished teaching content (~1-2 hours of learner effort). Do NOT split thin content into multiple modules, and do NOT cram a large source into a handful of overstuffed modules — if the source is large, more focused modules is correct.
3. Order modules logically — build from foundational concepts to advanced applications.
4. Base ALL module topics strictly on the provided source material. Every topic must be directly traceable to the source. If there isn't enough material to justify a distinct module, don't create one.
5. Make titles and descriptions specific, academic, and substantive — not generic.
6. The course overview should reflect the ACTUAL scope of the source — if the source is narrow, the overview should be 1-2 sentences, not inflated.

OUTPUT FORMAT:
# Course Title: [Insert Compelling Title]
[Insert a course overview explaining what the learner will master — length proportional to source scope.]

## Module 1: [Module Title]
**Description:** [2-3 sentences describing what this module covers and its specific learning objectives.]

## Module 2: [Module Title]
**Description:** [2-3 sentences describing what this module covers and its specific learning objectives.]`

// Discovery prompt — the Stage-0 flow. Before the user commits to a course
// description, this prompt looks at what's actually in the source material and
// proposes distinct angles they could take. Honest about scope: if the docs
// are thin, it says so rather than inflating the description.
const discoveryPrompt = `You are an expert instructional designer helping a course author explore what's possible with their source material.

You will receive:
- The author's topic or area of interest (may be vague or empty)
- Retrieved source material (may be sparse if no relevant docs exist)

Your job:
1. Summarize what's ACTUALLY in the source material — its scope, depth, and themes. Be honest. If the material is thin or narrow, say so plainly. Do not inflate.
2. Propose 2-4 DISTINCT course angles the author could take. Each angle must be a genuinely different framing — not minor variations of the same idea. If the source is thin, fewer angles is fine.
3. For each angle, explain why it works given the source material.

CRITICAL RULES:
- Base angles strictly on what's in the source material. Do not invent topics that aren't supported.
- If there's little or no source material, be honest: say so in the summary and propose angles based on the author's topic using general knowledge, clearly noting which angles need additional source material.
- Titles should be specific and compelling, not generic.
- Descriptions should be 2-3 sentences describing what the course would cover.

Output ONLY valid JSON — no markdown, no surrounding text:

{
  "summary": "2-4 sentence honest assessment of what the source material covers and its depth.",
  "angles": [
    {
      "title": "Compelling Course Title",
      "description": "What this course would cover and the learner would master.",
      "rationale": "Why this angle fits the source material."
    }
  ]
}`

// coursePlanRevisionPrompt is used when the user asks the AI to revise an
// existing outline ("fewer modules", "more practical", "add a module on X").
// It produces the same Markdown format as coursePlanSystemPrompt but is
// instructed to keep what already works and only apply the requested changes
// — a full re-outline would discard the user's curation so far.
const coursePlanRevisionPrompt = `You are an expert instructional designer revising an existing course outline based on author feedback.

You will receive:
- The CURRENT outline (title, description, list of modules)
- FEEDBACK from the course author
- Source material statistics and content

Apply the feedback faithfully. Keep modules that already work; only change what the feedback asks for. Do not throw out good structure just to seem responsive — if the feedback is "add a module on X", add that module and leave the rest alone.

CRITICAL RULES:
1. Structure your output EXACTLY like the Markdown example below. Do NOT use JSON, and do not write any introductory or concluding conversational filler.
2. Module count must still be proportional to the source material (see scale below), unless the author gives an explicit target count. When the author says "to N modules", the output MUST contain exactly N total modules; merge or split modules as needed to reach that total. Never interpret "to N modules" as adding N modules or removing N modules. If the feedback asks for fewer modules without an explicit target, merge related ones rather than deleting content outright. If it asks for more without an explicit target, split overstuffed modules — don't invent material that isn't in the source.
   - Under 1,000 chars of source → 1 module
   - 1,000–8,000 chars → 2-3 modules
   - 8,000–30,000 chars → 4-6 modules
   - 30,000–80,000 chars → 7-12 modules
   - 80,000–200,000 chars → 13-20 modules
   - Over 200,000 chars → up to 25 modules
3. Order modules logically — foundational concepts first, then advanced applications.
4. Base ALL module topics strictly on the provided source material. Every topic must be directly traceable to the source.
5. Make titles and descriptions specific, academic, and substantive — not generic.
6. The course overview should reflect the ACTUAL scope of the source.

OUTPUT FORMAT:
# Course Title: [Insert Compelling Title]
[Insert a course overview explaining what the learner will master — length proportional to source scope.]

## Module 1: [Module Title]
**Description:** [2-3 sentences describing what this module covers and its specific learning objectives.]

## Module 2: [Module Title]
**Description:** [2-3 sentences describing what this module covers and its specific learning objectives.]`

// Step 2a: Section Lister — identifies 1-5 key sub-topics within a module.
const moduleSectionListerPrompt = `You are an expert instructional designer. Given a course module's title, description, and source material, identify key sub-topics or sections that comprehensively break down this module's content.

Output ONLY a valid JSON array of section titles — no markdown, no other text:

["Section Title 1", "Section Title 2", "Section Title 3"]

CRITICAL RULES:
- The number of sections MUST match the depth of the available source material for this module. Produce 1-5 sections — use fewer when the source is thin, more when it's rich.
- If the source material is very short (only a few paragraphs), 1-2 sections is perfectly appropriate.
- Order sections logically — foundational concepts first, then deeper material.
- Each title should be specific and substantive, not generic.
- Cover ALL key ideas from the source. Do not skip important concepts.
- Do NOT create sections that aren't supported by the source — do not invent topics to fill a quota.`

// Step 2b: Section Content Writer — produces raw Markdown content for ONE specific section.
const moduleSectionWriterPrompt = `You are an expert instructional designer and university-level educator. Your primary job is to produce faithful teaching material for ONE SPECIFIC SECTION of a course module.

You are writing the raw content for this section ONLY. Do not include quiz questions, formatting code, or structural wrappers.

CRITICAL RULES:
1. Match your output length to the source material available for this section. If the source provides only a few sentences on this topic, write 1-2 concise paragraphs. If the source is rich and detailed, you may write longer. NEVER stretch thin content into long passages — conciseness is valuable.
2. Be absolutely FAITHFUL to the information in the source. Paraphrasing is encouraged, but every fact, claim, concept, and example must accurately reflect the source. Do NOT invent, embellish, or add information not supported by the source.
3. Organize your writing using clear Markdown subheadings (####), bold text, and bullet points where appropriate to make it highly readable and scannable.
4. Do NOT use placeholders, TBD, or lorem ipsum.
5. Do NOT include any quiz questions, multiple choice, true/false, or assessment items. This is PURE CONTENT only.
6. If the source material for this section is very thin, it is better to be short and accurate than long and fabricated.`

// Step 3: Per-Section Question Generator — produces 1-3 assessment items
// that test the content JUST covered in one section. Used inside the per-section
// loop so questions are interleaved with content, not batched at the end.
//
// The prompt explicitly asks the model to skip question types it cannot
// produce faithfully for this section, and to record the reason in a
// `limitations` array. The worker collects these across sections and stores
// them on the module row so the user can see why e.g. "matching" was skipped
// for a conceptual section that has no natural pairs.
const perSectionQuestionPrompt = `You are an expert assessment designer. Given ONE section of educational content, generate 1-3 high-quality assessment items that test understanding of THIS section specifically.

CRITICAL RULES:
1. Generate questions ONLY about the content in this specific section — do not draw from other topics.
2. Pick question types that BEST fit the section material. Choose from: mc (multiple choice), ma (multiple answer), tf (true/false), fb (fill-in-the-blank), sa (short answer). Matching, drag_sort, hotspot, and sequence are also allowed if the content naturally suits them.
3. Vary the types — don't use the same format for all questions.
4. Every question MUST be answerable strictly from the section content.
5. Every question MUST include an "explanation" field explaining the correct answer(s).
6. MC: exactly 4 plausible options with distractions that are common misconceptions; "correct" is a 0-based index.
7. MA: 4-6 options with 2-3 correct; "correct" is an array of 0-based indices.
8. TF: "statement" + "answer" (boolean).
9. FB: "text" with "___" blanks + "blanks" array of answers.
10. SA: "question" + "sample_answer".
11. Content must be substantive — not trivial recall of names or dates.
12. Do not use placeholders, TBD, or lorem ipsum.
13. Every question MUST include "irt_beta" and "irt_alpha" fields in the data object. These are Item Response Theory parameters that the adaptive engine uses to select questions at the learner's difficulty level:
    - irt_beta: item difficulty from -3 (trivially easy) to +3 (extremely hard). Estimate based on Bloom's Taxonomy level of the question — simple recall = -2 to -1, comprehension = -1 to 0, application = 0 to +1, analysis/synthesis = +1 to +2, evaluation = +2 to +3.
    - irt_alpha: item discrimination from 0.5 (poor discriminator — everyone gets it right or wrong) to 2.5 (excellent discriminator — sharply separates strong from weak learners). Most well-written questions are around 1.0-1.5.

BE HONEST ABOUT QUESTION TYPES THAT DON'T FIT:
If a requested question type cannot be produced faithfully for THIS section, DO NOT force it. Forcing a bad question is worse than skipping it. Examples:
  - "matching" requires at least 4 natural pairs in the content — skip if the section is a single concept or definition.
  - "drag_sort" / "sequence" requires a clear ordered process — skip if the content is conceptual with no inherent ordering.
  - "hotspot" REQUIRES a real image with spatial regions to click. Since you are generating TEXT content with no actual images, you CANNOT produce a valid hotspot question. ALWAYS skip "hotspot" and add it to the limitations array with reason "hotspot requires a real image; text-only content has no clickable regions".
  - "fb" works best for factual recall (names, terms, numbers) — skip if the section is purely conceptual prose.
For each requested type you skip, add an entry to the "limitations" array explaining why.

Output ONLY valid JSON — no markdown, no surrounding text — in this shape:

{
  "items": [
    {
      "type": "mc",
      "data": { "question": "...", "options": ["...","...","...","..."], "correct": 0, "explanation": "...", "irt_beta": 0.5, "irt_alpha": 1.2 }
    },
    {
      "type": "tf",
      "data": { "statement": "...", "answer": true, "explanation": "...", "irt_beta": -0.3, "irt_alpha": 1.0 }
    }
  ],
  "limitations": [
    { "type": "matching", "reason": "Section defines a single concept with no natural pairs to match." }
  ]
}`

// conceptVariantPrompt is the adaptive-learning prompt. Instead of picking a
// few question types per section, it identifies distinct assessment CONCEPTS
// and renders EACH concept in EVERY question type that fits it. Variants of one
// concept share a concept_id so the worker can write question_group_id, which
// lets the adaptive engine serve each learner their preferred/best type for a
// given concept rather than always the same format.
//
// Key rules carried over from perSectionQuestionPrompt: only test this
// section, every item needs irt params, and be honest about types that don't
// fit (record them in limitations instead of forcing a bad item). hotspot is
// always skipped for text-only content.
const conceptVariantPrompt = `You are an expert assessment designer building an adaptive-learning item bank. Given ONE section of educational content, identify the distinct concepts worth assessing, then render EACH concept as MULTIPLE question-type variants so a learner can be served their preferred or best-performing format for that same concept.

CRITICAL RULES:
1. Generate questions ONLY about the content in this specific section — do not draw from other topics.
2. Identify 1-3 distinct assessment CONCEPTS in this section. A concept is one testable idea (e.g. "the three stages of X", "why Y causes Z", "the definition of W"). Each concept becomes a group that can hold several type-variants.
3. For EACH concept, produce a variant in EVERY question type that fits it, drawn from: mc (multiple choice), ma (multiple answer), tf (true/false), fb (fill-in-the-blank), sa (short answer), matching (matching), drag_sort (drag and sort / sequence). Do NOT limit yourself to one or two types — the goal is breadth, since adaptive learning picks the format per learner.
4. Variants of the SAME concept MUST test the same underlying idea (same answer/fact), just expressed in that type's format. They share the concept_id.
5. Every variant MUST be answerable strictly from the section content, and MUST include an "explanation" field explaining the correct answer(s).
6. Format specifics:
   - MC: exactly 4 plausible options with distractions that are common misconceptions; "correct" is a 0-based index.
   - MA: 4-6 options with 2-3 correct; "correct" is an array of 0-based indices.
   - TF: "statement" + "answer" (boolean).
   - FB: "text" with "___" blanks + "blanks" array of answers.
   - SA: "question" + "sample_answer".
   - matching: "prompt" + "pairs" array of {left, right} (at least 4 pairs).
   - drag_sort: "prompt" + "items" array to be put in order (the correct order is the array order as written; the UI shuffles them).
7. Every variant MUST include "irt_beta" and "irt_alpha" in its data object (Item Response Theory params the adaptive engine uses to target difficulty):
   - irt_beta: item difficulty from -3 (trivially easy) to +3 (extremely hard). Use Bloom's level — recall = -2 to -1, comprehension = -1 to 0, application = 0 to +1, analysis/synthesis = +1 to +2, evaluation = +2 to +3. Variants of one concept should share roughly the same beta.
   - irt_alpha: discrimination from 0.5 to 2.5; well-written questions ~1.0-1.5.

BE HONEST ABOUT QUESTION TYPES THAT DON'T FIT A CONCEPT:
For each concept, include a variant for a type ONLY if that type can faithfully test the concept. If a type does not fit, OMIT its variant and add a limitations entry naming that concept_id and type with a reason. Examples:
  - matching needs at least 4 natural pairs — skip on a single-concept definition.
  - drag_sort needs a clear ordered process — skip on purely conceptual material.
  - fb works best for factual recall (names/terms/numbers) — skip on conceptual prose.
  - hotspot REQUIRES a real image with clickable regions; this is text-only content, so NEVER produce a hotspot variant — always add it to limitations.
  - tf needs an unambiguous true/false statement — skip if the concept is nuanced/multi-part.
Forcing a bad question is worse than skipping it.

Output ONLY valid JSON — no markdown, no surrounding text — in this shape:

{
  "concepts": [
    {
      "concept_id": "c1",
      "stem": "One short phrase naming the core idea this concept tests.",
      "variants": [
        { "type": "mc", "data": { "question": "...", "options": ["...","...","...","..."], "correct": 0, "explanation": "...", "irt_beta": 0.5, "irt_alpha": 1.2 } },
        { "type": "tf", "data": { "statement": "...", "answer": true, "explanation": "...", "irt_beta": 0.5, "irt_alpha": 1.0 } }
      ]
    }
  ],
  "limitations": [
    { "concept_id": "c1", "type": "matching", "reason": "Concept is a single definition with no natural pairs." }
  ]
}`

// Step 3 (legacy): Question Generator — produces a batch of 8 mixed-format items
// from a content summary. The full content is NOT sent to the LLM — only a summary
// for context. The JSON wrapping is done programmatically in Go code.
const questionGenPrompt = `You are an expert assessment designer. Based on the educational content summary provided, generate EXACTLY 8 high-quality assessment items with VARIED formats.

The 8 items MUST use these exact formats, in this order:
1. Multiple choice (mc) — single correct answer
2. Multiple answer (ma) — multiple correct answers
3. True / false (tf)
4. Fill in the blank (fb)
5. Matching (matching)
6. Ordering / sequence (drag_sort)
7. Hotspot / identify area (hotspot)
8. Short answer (sa)

Output ONLY a valid JSON array — no markdown, no other text:

[
  {
    "type": "mc",
    "data": {
      "question": "...",
      "options": ["...","...","...","..."],
      "correct": 0,
      "explanation": "Why the correct answer is right and the others are wrong.",
      "irt_beta": 0.5,
      "irt_alpha": 1.2
    }
  },
  {
    "type": "ma",
    "data": {
      "question": "Select all that apply.",
      "options": ["...","...","...","...","..."],
      "correct": [0, 2],
      "explanation": "Why these answers are correct and the others are not.",
      "irt_beta": 0.8,
      "irt_alpha": 1.4
    }
  },
  {
    "type": "tf",
    "data": {
      "statement": "A substantive statement based directly on the source.",
      "answer": true,
      "explanation": "Why the statement is true (or false).",
      "irt_beta": -0.3,
      "irt_alpha": 1.0
    }
  },
  {
    "type": "fb",
    "data": {
      "text": "A sentence with one or more ___ to fill in.",
      "blanks": ["expected answer 1", "expected answer 2"],
      "explanation": "Why those are the correct fills.",
      "irt_beta": 0.2,
      "irt_alpha": 1.1
    }
  },
  {
    "type": "matching",
    "data": {
      "question": "Match each concept with the correct definition or implication.",
      "pairs": [
        { "left": "Concept A", "right": "Definition or implication A" },
        { "left": "Concept B", "right": "Definition or implication B" },
        { "left": "Concept C", "right": "Definition or implication C" },
        { "left": "Concept D", "right": "Definition or implication D" }
      ],
      "explanation": "Why these pairings are correct.",
      "irt_beta": 0.6,
      "irt_alpha": 1.3
    }
  },
  {
    "type": "drag_sort",
    "data": {
      "question": "Put these steps in the correct order.",
      "items": ["First step", "Second step", "Third step", "Fourth step"],
      "explanation": "Why this is the correct sequence.",
      "irt_beta": 0.4,
      "irt_alpha": 1.1
    }
  },
  {
    "type": "hotspot",
    "data": {
      "question": "Select the area that best represents the correct concept.",
      "image": "/brand/questions/hotspot-cyber-risk.png",
      "regions": [
        { "label": "Correct region", "x": 42, "y": 54, "correct": true },
        { "label": "Distractor 1", "x": 25, "y": 35, "correct": false },
        { "label": "Distractor 2", "x": 68, "y": 42, "correct": false },
        { "label": "Distractor 3", "x": 55, "y": 75, "correct": false }
      ],
      "explanation": "Why the correct region represents the concept.",
      "irt_beta": 0.3,
      "irt_alpha": 1.0
    }
  },
  {
    "type": "sa",
    "data": {
      "question": "An open-ended question requiring a 2-4 sentence answer.",
      "sample_answer": "An exemplary answer demonstrating the depth expected.",
      "explanation": "What a strong answer should cover.",
      "irt_beta": 1.2,
      "irt_alpha": 1.5
    }
  }
]

CRITICAL RULES:
1. Every question MUST test meaningful understanding — never trivial recall of names or dates.
2. Every question MUST be answerable strictly from the source content; do not invent facts.
3. MC: exactly 4 plausible options; "correct" is a 0-based index; wrong options should be common misconceptions.
4. MA: 4-6 options with 2-3 correct; "correct" is an array of 0-based indices.
5. TF: state a clear claim that is unambiguously true or false based on the source.
6. FB: use literally three underscores "___" for each blank in "text"; "blanks" lists the answers in order; 1-3 blanks.
7. MATCHING: use 3-5 pairs; each left and right must be short, unambiguous, and source-grounded.
8. DRAG_SORT: use 3-5 ordered items; "items" must be in the correct order; the frontend will shuffle them.
9. HOTSPOT: use image "/brand/questions/hotspot-cyber-risk.png" unless a different existing asset is clearly more relevant from this allow-list: "/brand/pathways/compliance.png", "/brand/pathways/risk-credit.png", "/brand/pathways/customer-service.png", "/brand/pathways/cybersecurity.png", "/brand/pathways/banking-foundations.png"; include 3-4 labeled regions with exactly one correct region; x/y are percentages from 0-100 and may be approximate.
10. SA: include a substantive "sample_answer" demonstrating expected depth.
11. Every item MUST include an "explanation" field.
12. Every question MUST include "irt_beta" (difficulty: -3 trivial to +3 extremely hard, based on Bloom's level) and "irt_alpha" (discrimination: 0.5 poor to 2.5 excellent, typical 1.0-1.5).
13. Output ONLY the JSON array of exactly 8 objects. No prose, no markdown fences.`

// EditCourse system prompt — for AI-driven course editing.
const editCourseSystemPrompt = `You are an expert instructional designer and course editor. Given an existing course in JSON format and edit instructions, produce the full modified course JSON.

Output ONLY valid JSON — no markdown. Structure:

{
  "title": "Course Title",
  "description": "Course overview paragraph(s)",
  "modules": [
    {
      "title": "Module Title",
      "description": "Module description",
      "items": [
        {"type": "content", "data": { "body": "..." }},
        {"type": "mc", "data": { "question": "...", "options": [...], "correct": 0, "explanation": "...", "irt_beta": 0.5, "irt_alpha": 1.2 }}
      ]
    }
  ]
}

CRITICAL RULES:
- Apply the edit instructions faithfully while preserving the overall course structure.
- Content must remain accurate and educational.
- Keep assessment items meaningful and not trivial.
- Every assessment item (not "content") MUST include "irt_beta" (difficulty: -3 trivial to +3 extremely hard, based on Bloom's level of the question) and "irt_alpha" (discrimination: 0.5 poor to 2.5 excellent, typical 1.0-1.5).
- Do not use placeholders or lorem ipsum.`

// EditItem system prompt — for AI-driven editing of a single course item.
const editItemSystemPrompt = `You are an expert instructional designer and content editor. Given a single course item in JSON format and edit instructions, produce the modified version of JUST that item.

Output ONLY valid JSON — no markdown, no explanation. Keep the same item_type and data structure:

- "content": { "data": { "body": "..." } }
- "mc": { "data": { "question": "...", "options": [...], "correct": <index>, "explanation": "...", "irt_beta": <float>, "irt_alpha": <float> } }
- "ma": { "data": { "question": "...", "options": [...], "correct": [<indices>], "explanation": "...", "irt_beta": <float>, "irt_alpha": <float> } }
- "tf": { "data": { "statement": "...", "answer": <bool>, "explanation": "...", "irt_beta": <float>, "irt_alpha": <float> } }
- "fb": { "data": { "text": "...", "blanks": [...], "explanation": "...", "irt_beta": <float>, "irt_alpha": <float> } }
- "sa": { "data": { "question": "...", "sample_answer": "...", "irt_beta": <float>, "irt_alpha": <float> } }
- "matching": { "data": { "question": "...", "pairs": [{"left":"...","right":"..."}], "explanation": "...", "irt_beta": <float>, "irt_alpha": <float> } }
- "drag_sort": { "data": { "question": "...", "items": [...], "explanation": "...", "irt_beta": <float>, "irt_alpha": <float> } }
- "sequence": { "data": { "question": "...", "steps": [...], "explanation": "...", "irt_beta": <float>, "irt_alpha": <float> } }
- "hotspot": { "data": { "question": "...", "image": "...", "regions": [{"label":"...","x":<pct>,"y":<pct>,"correct":<bool>}], "explanation": "...", "irt_beta": <float>, "irt_alpha": <float> } }

CRITICAL RULES:
- Apply the edit instructions faithfully to this single item.
- Keep the same item_type — do not change it.
- Maintain the same data structure/fields appropriate to the item_type.
- Content must remain accurate, educational, and substantive.
- For assessment items (not "content"), include "irt_beta" (difficulty: -3 trivial to +3 extremely hard, based on Bloom's level) and "irt_alpha" (discrimination: 0.5 poor to 2.5 excellent, typical 1.0-1.5).
- Do not use placeholders or lorem ipsum.
	- Output ONLY the JSON object — no surrounding text or markdown fences.`

// ensureIRTParams injects irt_beta and irt_alpha defaults into question item data
// when the LLM omits them. Content items are passed through unchanged.
// Returns the marshalled []byte suitable for CreateCourseItem / UpdateCourseItemData.
func ensureIRTParams(data json.RawMessage, itemType string) []byte {
	if itemType == "content" {
		return []byte(data)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return []byte(data)
	}
	if _, ok := m["irt_beta"]; !ok {
		m["irt_beta"] = 0.0
	}
	if _, ok := m["irt_alpha"]; !ok {
		m["irt_alpha"] = 1.0
	}
	out, err := json.Marshal(m)
	if err != nil {
		return []byte(data)
	}
	return out
}

type Handler struct {
	Pool      *pgxpool.Pool
	Queries   *database.Queries
	LLM       *LLMClient
	EmbClient *embeddings.Client
	Jobs      *JobStore
	Worker    *Worker
}

func NewHandler(pool *pgxpool.Pool, queries *database.Queries) *Handler {
	llm := NewLLMClient()
	emb := embeddings.NewClient()
	store := newJobStore(queries)
	worker := newWorker(pool, store, queries, llm, emb)
	return &Handler{
		Pool:      pool,
		Queries:   queries,
		LLM:       llm,
		EmbClient: emb,
		Jobs:      store,
		Worker:    worker,
	}
}

// ---- JSON structures for LLM output ----

type genItem struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// conceptGroup is the output of the adaptive conceptVariantPrompt: one
// testable idea (concept_id + stem) rendered as multiple question-type
// variants. The worker writes question_group_id onto each variant's item so
// the adaptive engine can serve a learner their preferred type for the concept.
type conceptGroup struct {
	ConceptID string    `json:"concept_id"`
	Stem      string    `json:"stem"`
	Variants  []genItem `json:"variants"`
}

type conceptLimitation struct {
	ConceptID string `json:"concept_id"`
	Type      string `json:"type"`
	Reason    string `json:"reason"`
}

type genModule struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Items       []genItem `json:"items"`
}

type genCourse struct {
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Modules     []genModule `json:"modules"`
}

// planModule is the parsed result from Step 1 Markdown output.
type planModule struct {
	Title       string
	Description string
}

type plan struct {
	Title       string
	Description string
	Modules     []planModule
}

// requestedModuleCount recognizes wording that specifies the final total.
// Additive requests such as "add 5 modules" intentionally do not match.
func requestedModuleCount(feedback string) (int, bool) {
	targetRe := regexp.MustCompile(`(?i)\b(?:to|make\s+it)\s+(\d+)\s+modules?\b`)
	match := targetRe.FindStringSubmatch(feedback)
	if len(match) != 2 {
		return 0, false
	}

	target, err := strconv.Atoi(match[1])
	if err != nil || target < 1 || target > 25 {
		return 0, false
	}
	return target, true
}

// ---- Request types ----

// JobStage identifies which pipeline a GenerationJob should run.
//   - "full"    legacy one-shot: outline + every module in one job.
//   - "outline" outline only: produce the courses/modules rows and stop.
//     Modules are created with status='pending' so the user can
//     review/edit before any content is generated.
//   - "module"  generate content + items for ONE existing module row.
//     ModuleID must be set.
type JobStage string

const (
	StageFull    JobStage = "full"
	StageOutline JobStage = "outline"
	StageModule  JobStage = "module"
)

type GenerateCourseRequest struct {
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	SourceDocIDs  []int64  `json:"source_doc_ids"`
	QuestionTypes []string `json:"question_types,omitempty"`
	CreatedBy     int64    `json:"created_by,omitempty"`

	// MaxQuestions is an optional HARD cap on the number of unique questions
	// (concepts) generated for the course. 0/unset = no cap. It is persisted in
	// course settings so it carries through to per-module generation, and the
	// worker distributes the budget across modules (ceil(total/modules)).
	MaxQuestions int `json:"max_questions,omitempty"`

	// Stage selects which pipeline runs. Defaults to "full" for backwards
	// compatibility — existing callers that omit it get the old one-shot flow.
	Stage JobStage `json:"stage,omitempty"`

	// CourseID is set for StageOutline (when regenerating an outline into an
	// existing course) and required for StageModule jobs.
	CourseID int64 `json:"course_id,omitempty"`

	// ModuleID is required for StageModule jobs — it identifies the existing
	// modules row whose content/items should be (re)generated.
	ModuleID int64 `json:"module_id,omitempty"`

	// Feedback drives outline revision. When set together with CourseID, the
	// outline job runs against an existing course: it loads the current
	// outline, applies the user's feedback (e.g. "fewer modules", "more
	// practical", "add a module on X"), and replaces the modules + items.
	Feedback string `json:"feedback,omitempty"`
}

// ---- SSE writer ----

type sseWriter struct {
	c       *gin.Context
	flusher http.Flusher
}

func newSSEWriter(c *gin.Context) (*sseWriter, bool) {
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return nil, false
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	return &sseWriter{c: c, flusher: flusher}, true
}

func (w *sseWriter) send(event string, data interface{}) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w.c.Writer, "event: %s\ndata: %s\n\n", event, string(jsonBytes))
	if err != nil {
		return err
	}
	w.flusher.Flush()
	return nil
}

func (w *sseWriter) sendError(msg string) {
	w.send("error", map[string]string{"message": msg})
}

// ---- Handlers ----

// GenerateCourse enqueues a course generation job and returns immediately.
// The client should then subscribe to the job's SSE stream for progress updates.
func (h *Handler) GenerateCourse(c *gin.Context) {
	var req GenerateCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.Description) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Description is required"})
		return
	}
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	createdBy, ok := userID.(int64)
	if !ok || createdBy == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user session"})
		return
	}
	req.CreatedBy = createdBy

	job := h.Jobs.create(req)
	h.Worker.enqueue(job)

	c.JSON(http.StatusAccepted, gin.H{
		"job_id": job.ID,
		"status": job.Status,
	})
}

// GetJobStatus returns the current state of a generation job.
func (h *Handler) GetJobStatus(c *gin.Context) {
	id := JobID(c.Param("id"))
	job := h.Jobs.get(id)
	if job == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	job.mu.RLock()
	defer job.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"id":         job.ID,
		"status":     job.Status,
		"steps":      job.Steps,
		"modules":    job.Modules,
		"result":     job.Result,
		"error":      job.Error,
		"created_at": job.CreatedAt,
	})
}

// StreamJob opens an SSE connection and streams progress events for a job.
func (h *Handler) StreamJob(c *gin.Context) {
	id := JobID(c.Param("id"))
	job := h.Jobs.get(id)
	if job == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	w, ok := newSSEWriter(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming not supported"})
		return
	}

	sub := job.subscribe()
	defer job.unsubscribe(sub)

	// Replay any steps that already happened (catch-up)
	job.mu.RLock()
	for _, step := range job.Steps {
		w.send("step", step)
	}
	for _, mod := range job.Modules {
		w.send("module", gin.H{"module": mod})
	}
	if job.Status == "completed" {
		w.send("done", job.Result)
		job.mu.RUnlock()
		return
	}
	if job.Status == "failed" {
		w.send("error", map[string]string{"message": job.Error})
		job.mu.RUnlock()
		return
	}
	job.mu.RUnlock()

	// Stream new events as they arrive
	ctx := c.Request.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-sub:
			if !ok {
				return
			}
			w.send(event.Event, event.Data)
			if event.Event == "done" || event.Event == "error" {
				return
			}
		}
	}
}

// ListCourses returns all courses.
func (h *Handler) ListCourses(c *gin.Context) {
	courses, err := h.Queries.GetCourses(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch courses"})
		return
	}
	if courses == nil {
		courses = []database.Course{}
	}
	c.JSON(http.StatusOK, courses)
}

// GetCourse returns a single course with modules and items.
func (h *Handler) GetCourse(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid course ID"})
		return
	}
	course, err := h.Queries.GetCourseByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}

	modules, _ := h.Queries.GetModulesByCourseWithStatus(c.Request.Context(), id)
	items, _ := h.Queries.GetCourseItemsByCourseWithGroup(c.Request.Context(), id)

	// Parse sources from settings
	var sources interface{}
	if len(course.Settings) > 0 {
		var settingsMap map[string]json.RawMessage
		if json.Unmarshal(course.Settings, &settingsMap) == nil {
			if raw, ok := settingsMap["sources"]; ok {
				json.Unmarshal(raw, &sources)
			}
		}
	}

	itemMap := make(map[int64][]gin.H)
	for _, item := range items {
		mid := item.ModuleID.Int64
		entry := gin.H{
			"id":         item.ID,
			"item_type":  item.ItemType,
			"sort_order": item.SortOrder,
			"data":       json.RawMessage(item.Data),
		}
		if item.QuestionGroupID.Valid {
			entry["question_group_id"] = item.QuestionGroupID.String
		}
		itemMap[mid] = append(itemMap[mid], entry)
	}

	modulesResult := make([]gin.H, 0)
	for _, m := range modules {
		modItems := itemMap[m.ID]
		if modItems == nil {
			modItems = []gin.H{}
		}
		var qt any
		if len(m.QuestionTypes) > 0 {
			qt = m.QuestionTypes
		} else {
			qt = nil
		}
		limits := m.Limitations
		if limits == nil {
			limits = []map[string]any{}
		}
		modulesResult = append(modulesResult, gin.H{
			"id":                    m.ID,
			"title":                 m.Title,
			"description":           m.Description,
			"sort_order":            m.SortOrder,
			"status":                m.Status,
			"question_types":        qt,
			"question_type_targets": m.QuestionTypeTargets,
			"limitations":           limits,
			"items":                 modItems,
		})
	}

	// Parse full settings JSON
	var settings json.RawMessage
	if len(course.Settings) > 0 {
		settings = json.RawMessage(course.Settings)
	} else {
		settings = json.RawMessage("{}")
	}

	c.JSON(http.StatusOK, gin.H{
		"id":             course.ID,
		"title":          course.Title,
		"description":    course.Description,
		"status":         course.Status,
		"source_doc_ids": course.SourceDocIds,
		"sources":        sources,
		"settings":       settings,
		"modules":        modulesResult,
		"approved_by":    course.ApprovedBy,
	})
}

// PreviewCourse returns any course with modules and items for student-perspective preview,
// regardless of publish status.
func (h *Handler) PreviewCourse(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid course ID"})
		return
	}

	course, err := h.Queries.GetCourseByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}

	modules, _ := h.Queries.GetModulesByCourse(c.Request.Context(), id)
	items, _ := h.Queries.GetCourseItemsByCourseWithGroup(c.Request.Context(), id)

	itemMap := make(map[int64][]gin.H)
	for _, item := range items {
		mid := item.ModuleID.Int64
		entry := gin.H{
			"id":         item.ID,
			"item_type":  item.ItemType,
			"sort_order": item.SortOrder,
			"data":       json.RawMessage(item.Data),
		}
		if item.QuestionGroupID.Valid {
			entry["question_group_id"] = item.QuestionGroupID.String
		}
		itemMap[mid] = append(itemMap[mid], entry)
	}

	modulesResult := make([]gin.H, 0)
	for _, m := range modules {
		modItems := itemMap[m.ID]
		if modItems == nil {
			modItems = []gin.H{}
		}
		modulesResult = append(modulesResult, gin.H{
			"id":          m.ID,
			"title":       m.Title,
			"description": m.Description,
			"sort_order":  m.SortOrder,
			"items":       modItems,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"title":       course.Title,
		"description": course.Description,
		"modules":     modulesResult,
	})
}

// ListActiveJobs returns all currently active generation jobs.
// The frontend uses this to discover in-progress generations across tabs.
func (h *Handler) ListActiveJobs(c *gin.Context) {
	active := h.Jobs.listActive()
	result := make([]gin.H, len(active))
	for i, job := range active {
		job.mu.RLock()
		result[i] = gin.H{
			"id":         job.ID,
			"status":     job.Status,
			"stage":      job.Stage,
			"steps":      job.Steps,
			"modules":    job.Modules,
			"created_at": job.CreatedAt,
		}
		job.mu.RUnlock()
	}
	c.JSON(http.StatusOK, result)
}

// CancelGeneration cancels an in-progress course generation job.
func (h *Handler) CancelGeneration(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Job ID is required"})
		return
	}
	if h.Jobs.CancelJob(JobID(id)) {
		c.JSON(http.StatusOK, gin.H{"status": "cancelled"})
	} else {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found or already completed"})
	}
}

// UpdateSettings updates the course settings (max_attempts, days_to_complete, etc).
func (h *Handler) UpdateSettings(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid course ID"})
		return
	}

	var body struct {
		Settings json.RawMessage `json:"settings"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	course, err := h.Queries.UpdateCourseSettings(c.Request.Context(), database.UpdateCourseSettingsParams{
		ID:       id,
		Settings: []byte(body.Settings),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update settings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":       course.ID,
		"settings": json.RawMessage(course.Settings),
	})
}

// ---- Staged course builder endpoints ----
// These complement the legacy one-shot POST /courses/generate flow.
// They allow a course to be built up one module at a time after the outline
// is created.

// UpdateCourseMetaHandler updates a course's title and/or description.
// Unlike PUT /courses/:id/edit (which runs an AI edit), this is a plain
// authoring write — what the user types is exactly what gets stored.
func (h *Handler) UpdateCourseMetaHandler(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid course ID"})
		return
	}
	course, err := h.Queries.GetCourseByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}

	var body struct {
		Title       *string `json:"title,omitempty"`
		Description *string `json:"description,omitempty"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	title := course.Title
	if body.Title != nil {
		title = strings.TrimSpace(*body.Title)
		if title == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "title cannot be empty"})
			return
		}
	}
	desc := course.Description
	if body.Description != nil {
		desc = strings.TrimSpace(*body.Description)
	}

	updated, err := h.Queries.UpdateCourseMeta(c.Request.Context(), database.UpdateCourseMetaParams{
		ID:          id,
		Title:       title,
		Description: desc,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update course"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":          updated.ID,
		"title":       updated.Title,
		"description": updated.Description,
		"status":      updated.Status,
	})
}

// Discover is the Stage-0 flow: before the user commits to a course description,
// this endpoint explores the source material and proposes distinct course
// angles. Stateless — creates no course or job rows. Returns a summary of
// what's in the docs plus 2-4 angle cards the user can pick from.
//
// Synchronous (single LLM call) rather than an SSE job: the response is small
// and the operation is one embed + one retrieve + one chat, which matches the
// pattern used by AiEditItem / EditCourse.
func (h *Handler) Discover(c *gin.Context) {
	var body struct {
		Topic        string  `json:"topic"`
		SourceDocIDs []int64 `json:"source_doc_ids"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	topic := strings.TrimSpace(body.Topic)
	// We need something to embed for retrieval. Fall back to a broad query
	// if the user gave no topic — the retrieval will still find the most
	// central chunks across approved docs.
	embedInput := topic
	if embedInput == "" {
		// Use the titles of the selected docs (or just "overview") so the
		// embedding has something to work with.
		if len(body.SourceDocIDs) > 0 {
			var titles []string
			for _, did := range body.SourceDocIDs {
				if d, err := h.Queries.GetDocumentByID(c.Request.Context(), did); err == nil {
					titles = append(titles, d.Title)
				}
			}
			embedInput = strings.Join(titles, " ")
		}
		if embedInput == "" {
			embedInput = "overview introduction fundamentals concepts"
		}
	}

	embeddings, err := h.EmbClient.Embed([]string{embedInput})
	if err != nil {
		log.Printf("[discover] embed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to analyze source material"})
		return
	}
	promptVec := pgvector.NewVector(float64ToFloat32(embeddings[0]))

	chunks, err := h.Queries.SearchDocumentChunks(c.Request.Context(), database.SearchDocumentChunksParams{
		Embedding: promptVec,
		Limit:     60, // focused: enough to assess scope without flooding the prompt
	})
	if err != nil {
		log.Printf("[discover] search: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search documents"})
		return
	}

	sourceContext := buildDiscoveryChunkContext(chunks)

	discoverUserPrompt := fmt.Sprintf(`Author's topic or area of interest: %s

Source material (%d chunks retrieved):
%s

Summarize what's here and propose course angles.`, quoteOrEmpty(topic), len(chunks), sourceContext)

	resp, err := h.LLM.Chat(discoveryPrompt, discoverUserPrompt)
	if err != nil {
		log.Printf("[discover] llm: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Discovery failed"})
		return
	}

	jsonStr := stripMarkdownFences(resp)
	var result struct {
		Summary string `json:"summary"`
		Angles  []struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			Rationale   string `json:"rationale"`
		} `json:"angles"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		log.Printf("[discover] parse: %v — raw: %s", err, resp)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse discovery response"})
		return
	}

	// Include a compact list of the docs the chunks came from so the UI can
	// show the user which sources were actually consulted.
	sourceDocs := buildDiscoverySourceDocs(chunks)

	c.JSON(http.StatusOK, gin.H{
		"summary":     result.Summary,
		"angles":      result.Angles,
		"chunks_seen": len(chunks),
		"source_docs": sourceDocs,
	})
}

// quoteOrEmpty wraps a string in quotes for prompt embedding, or returns
// "(none provided)" for empty input so the LLM sees the distinction.
func quoteOrEmpty(s string) string {
	if s == "" {
		return "(none provided)"
	}
	return "\"" + s + "\""
}

// buildDiscoveryChunkContext renders retrieved chunks compactly for the
// discovery prompt. Less verbose than buildChunkContext (no "Source:" labels
// per chunk) since discovery is about breadth, not citation.
func buildDiscoveryChunkContext(chunks []database.SearchDocumentChunksRow) string {
	var b strings.Builder
	for i, ch := range chunks {
		b.WriteString(fmt.Sprintf("\n--- Chunk %d (from \"%s\") ---\n%s\n",
			i+1, ch.DocumentTitle, ch.Content))
	}
	return b.String()
}

// buildDiscoverySourceDocs returns a deduplicated list of {id, title} pairs
// for the documents the retrieved chunks came from.
func buildDiscoverySourceDocs(chunks []database.SearchDocumentChunksRow) []gin.H {
	seen := make(map[int64]bool)
	out := make([]gin.H, 0)
	for _, ch := range chunks {
		if !seen[ch.DocumentID] {
			seen[ch.DocumentID] = true
			out = append(out, gin.H{"id": ch.DocumentID, "title": ch.DocumentTitle})
		}
	}
	return out
}

// ReorderModulesHandler takes an array of module IDs in the desired order and
// reassigns sort_order accordingly. Module IDs not in the array keep their
// existing positions. Atomic single-query update.
func (h *Handler) ReorderModulesHandler(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid course ID"})
		return
	}
	if _, err := h.Queries.GetCourseByID(c.Request.Context(), courseID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}

	var body struct {
		ModuleIDs []int64 `json:"module_ids"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(body.ModuleIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "module_ids is required"})
		return
	}

	// Guard against a generating module being moved while its job is mid-flight
	// — the worker would otherwise set sort_order back from the stale module
	// row it loaded at job start. Generating modules are pinned in place.
	mods, err := h.Queries.GetModulesByCourseWithStatus(c.Request.Context(), courseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load modules"})
		return
	}
	statusByID := make(map[int64]string, len(mods))
	for _, m := range mods {
		statusByID[m.ID] = m.Status
	}
	for _, id := range body.ModuleIDs {
		if statusByID[id] == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "module does not belong to this course"})
			return
		}
		if statusByID[id] == "generating" {
			c.JSON(http.StatusConflict, gin.H{"error": "cannot reorder while a module is generating"})
			return
		}
	}

	if err := h.Queries.ReorderModules(c.Request.Context(), courseID, body.ModuleIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reorder modules"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "reordered", "module_ids": body.ModuleIDs})
}

// GenerateOutline enqueues an outline-only job. It runs the same Step-1
// pipeline as the one-shot generator but stops after creating the course and
// module rows (each with status='pending'). Per-module generation is then
// driven by the client via POST /courses/:id/modules/:moduleId/generate.
func (h *Handler) GenerateOutline(c *gin.Context) {
	var req GenerateCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.Description) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Description is required"})
		return
	}
	userIDsrc, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	createdBy, ok := userIDsrc.(int64)
	if !ok || createdBy == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user session"})
		return
	}
	req.CreatedBy = createdBy
	req.Stage = StageOutline

	job := h.Jobs.create(req)
	h.Worker.enqueue(job)

	c.JSON(http.StatusAccepted, gin.H{
		"job_id": job.ID,
		"status": job.Status,
		"stage":  job.Stage,
	})
}

// RegenerateOutline enqueues an outline-revision job against an existing
// course. The body carries natural-language feedback (e.g. "fewer modules",
// "add a module on X", "make it more practical"). The worker wipes the
// course's current modules + items and replaces them with the AI's revised
// outline — this is destructive by design, so the frontend should confirm.
func (h *Handler) RegenerateOutline(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid course ID"})
		return
	}
	if _, err := h.Queries.GetCourseByID(c.Request.Context(), courseID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}

	var body struct {
		Instructions string `json:"instructions"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(body.Instructions) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "instructions are required"})
		return
	}

	userIDsrc, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	createdBy, ok := userIDsrc.(int64)
	if !ok || createdBy == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user session"})
		return
	}

	req := GenerateCourseRequest{
		CreatedBy: createdBy,
		Stage:     StageOutline,
		CourseID:  courseID,
		Feedback:  strings.TrimSpace(body.Instructions),
	}
	job := h.Jobs.create(req)
	h.Worker.enqueue(job)

	c.JSON(http.StatusAccepted, gin.H{
		"job_id": job.ID,
		"status": job.Status,
		"stage":  job.Stage,
	})
}

// GenerateModule enqueues a single-module generation job. The module must
// already exist as an outline row (created by GenerateOutline or created
// manually). The job streams progress like any other generation job and the
// caller should subscribe via GET /courses/generate/:id/stream.
func (h *Handler) GenerateModule(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid course ID"})
		return
	}
	moduleID, err := strconv.ParseInt(c.Param("moduleId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid module ID"})
		return
	}

	// Verify module belongs to the course.
	mod, err := h.Queries.GetModule(c.Request.Context(), moduleID)
	if err != nil || mod.CourseID != courseID {
		c.JSON(http.StatusNotFound, gin.H{"error": "Module not found in this course"})
		return
	}
	if mod.Status == "generating" {
		c.JSON(http.StatusConflict, gin.H{"error": "This module is already being generated"})
		return
	}

	userIDsrc, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	createdBy, ok := userIDsrc.(int64)
	if !ok || createdBy == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user session"})
		return
	}

	var body struct {
		QuestionTypes []string `json:"question_types,omitempty"`
		Title         string   `json:"title,omitempty"`
		Description   string   `json:"description,omitempty"`
	}
	_ = c.ShouldBindJSON(&body)

	// Allow caller to rename the module inline before generating.
	title := mod.Title
	if strings.TrimSpace(body.Title) != "" {
		title = strings.TrimSpace(body.Title)
	}
	desc := mod.Description
	if strings.TrimSpace(body.Description) != "" {
		desc = strings.TrimSpace(body.Description)
	}
	if title != mod.Title || desc != mod.Description {
		if err := h.Queries.UpdateModuleMeta(c.Request.Context(), moduleID, title, desc, mod.SortOrder, nil); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update module"})
			return
		}
	}

	req := GenerateCourseRequest{
		Title:         title,
		Description:   desc,
		QuestionTypes: body.QuestionTypes,
		CreatedBy:     createdBy,
		Stage:         StageModule,
		CourseID:      courseID,
		ModuleID:      moduleID,
	}
	job := h.Jobs.create(req)
	h.Worker.enqueue(job)

	c.JSON(http.StatusAccepted, gin.H{
		"job_id":    job.ID,
		"status":    job.Status,
		"stage":     job.Stage,
		"module_id": moduleID,
	})
}

// CreateModuleHandler adds a new (status='pending') module row to an existing
// course. Used when the user manually adds a module to an outline that the AI
// didn't propose.
func (h *Handler) CreateModuleHandler(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid course ID"})
		return
	}
	if _, err := h.Queries.GetCourseByID(c.Request.Context(), courseID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}

	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		SortOrder   *int32 `json:"sort_order,omitempty"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(body.Title) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
		return
	}

	var order int32
	if body.SortOrder != nil {
		order = *body.SortOrder
	} else {
		// Append after the highest existing sort_order.
		existing, _ := h.Queries.GetModulesByCourseWithStatus(c.Request.Context(), courseID)
		for _, m := range existing {
			if m.SortOrder >= order {
				order = m.SortOrder + 1
			}
		}
	}

	mod, err := h.Queries.CreateModule(c.Request.Context(), database.CreateModuleParams{
		CourseID:    courseID,
		Title:       strings.TrimSpace(body.Title),
		Description: strings.TrimSpace(body.Description),
		SortOrder:   order,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create module"})
		return
	}
	_ = h.Queries.UpdateModuleStatus(c.Request.Context(), mod.ID, "pending")

	c.JSON(http.StatusCreated, gin.H{
		"id":          mod.ID,
		"course_id":   mod.CourseID,
		"title":       mod.Title,
		"description": mod.Description,
		"sort_order":  mod.SortOrder,
		"status":      "pending",
		"items":       []gin.H{},
	})
}

// UpdateModuleHandler edits an existing module's outline fields
// (title/description/sort_order). It is only meaningful for modules that
// haven't been generated yet, but is allowed regardless so the user can fix
// typos in generated modules too.
func (h *Handler) UpdateModuleHandler(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid course ID"})
		return
	}
	moduleID, err := strconv.ParseInt(c.Param("moduleId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid module ID"})
		return
	}
	mod, err := h.Queries.GetModule(c.Request.Context(), moduleID)
	if err != nil || mod.CourseID != courseID {
		c.JSON(http.StatusNotFound, gin.H{"error": "Module not found in this course"})
		return
	}

	var body struct {
		Title       *string `json:"title,omitempty"`
		Description *string `json:"description,omitempty"`
		SortOrder   *int32  `json:"sort_order,omitempty"`
		// Pointer-to-slice so we can distinguish "omitted" (nil → leave
		// unchanged) from "explicitly cleared" (non-nil empty slice → all types).
		QuestionTypes       *[]string       `json:"question_types,omitempty"`
		QuestionTypeTargets *map[string]int `json:"question_type_targets,omitempty"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	title := mod.Title
	if body.Title != nil {
		title = strings.TrimSpace(*body.Title)
		if title == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "title cannot be empty"})
			return
		}
	}
	desc := mod.Description
	if body.Description != nil {
		desc = strings.TrimSpace(*body.Description)
	}
	order := mod.SortOrder
	if body.SortOrder != nil {
		order = *body.SortOrder
	}

	// Question types are only written when the caller includes the field at
	// all — omitting it preserves whatever allowlist was previously set so
	// a rename doesn't accidentally wipe a user-curated type set.
	var questionTypesJSON []byte
	if body.QuestionTypes != nil {
		// Normalise: lowercase, trim, drop unknowns, dedupe.
		known := map[string]bool{
			"mc": true, "ma": true, "tf": true, "fb": true, "sa": true,
			"matching": true, "drag_sort": true, "hotspot": true,
		}
		seen := make(map[string]bool)
		clean := make([]string, 0, len(*body.QuestionTypes))
		for _, qt := range *body.QuestionTypes {
			qt = strings.ToLower(strings.TrimSpace(qt))
			if known[qt] && !seen[qt] {
				seen[qt] = true
				clean = append(clean, qt)
			}
		}
		questionTypesJSON, _ = json.Marshal(clean)
	}
	if body.QuestionTypeTargets != nil {
		known := map[string]bool{"mc": true, "ma": true, "tf": true, "fb": true, "sa": true, "matching": true, "drag_sort": true, "hotspot": true}
		allowed := make(map[string]bool)
		if body.QuestionTypes != nil {
			for _, questionType := range *body.QuestionTypes {
				allowed[strings.ToLower(strings.TrimSpace(questionType))] = true
			}
		} else {
			for _, questionType := range mod.QuestionTypes {
				allowed[questionType] = true
			}
		}
		cleanTargets := make(map[string]int)
		for questionType, target := range *body.QuestionTypeTargets {
			questionType = strings.ToLower(strings.TrimSpace(questionType))
			if !known[questionType] || (len(allowed) > 0 && !allowed[questionType]) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "question type target must be for an allowed question type"})
				return
			}
			if target < 1 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "question type targets must be positive"})
				return
			}
			cleanTargets[questionType] = target
		}
		targetsJSON, _ := json.Marshal(cleanTargets)
		if err := h.Queries.UpdateModuleQuestionTypeTargets(c.Request.Context(), moduleID, targetsJSON); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save question type targets"})
			return
		}
	}

	if err := h.Queries.UpdateModuleMeta(c.Request.Context(), moduleID, title, desc, order, questionTypesJSON); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update module"})
		return
	}

	// Re-read so the response reflects the actual stored allowlist (which may
	// have been pruned of unknown types above).
	updated, _ := h.Queries.GetModule(c.Request.Context(), moduleID)
	respTypes := body.QuestionTypes
	respTargets := mod.QuestionTypeTargets
	if updated != nil {
		respTypes = &updated.QuestionTypes
		respTargets = updated.QuestionTypeTargets
	}
	c.JSON(http.StatusOK, gin.H{
		"id":                    moduleID,
		"course_id":             courseID,
		"title":                 title,
		"description":           desc,
		"sort_order":            order,
		"status":                mod.Status,
		"question_types":        respTypes,
		"question_type_targets": respTargets,
	})
}

// DeleteModuleHandler removes a module and its items from a course. Blocked
// if the module is currently generating, since that would race with the
// worker.
func (h *Handler) DeleteModuleHandler(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid course ID"})
		return
	}
	moduleID, err := strconv.ParseInt(c.Param("moduleId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid module ID"})
		return
	}
	mod, err := h.Queries.GetModule(c.Request.Context(), moduleID)
	if err != nil || mod.CourseID != courseID {
		c.JSON(http.StatusNotFound, gin.H{"error": "Module not found in this course"})
		return
	}
	if mod.Status == "generating" {
		c.JSON(http.StatusConflict, gin.H{"error": "Cannot delete a module that is currently generating"})
		return
	}

	// Items are ON DELETE SET NULL on module_id, so we have to delete them
	// explicitly to actually purge content.
	if err := h.Queries.DeleteCourseItemsByModule(c.Request.Context(), pgtype.Int8{Int64: moduleID, Valid: true}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete module items"})
		return
	}
	if err := h.Queries.DeleteModule(c.Request.Context(), moduleID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete module"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted", "module_id": moduleID})
}

// DeleteModulesHandler removes multiple modules (and their items) from a
// course in one request. Used by the course builder's bulk-delete UI. Like
// the single-module delete, modules that are currently generating are refused
// so we don't race the worker. The whole operation runs in a transaction so a
// partial failure can't leave the course half-pruned.
func (h *Handler) DeleteModulesHandler(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid course ID"})
		return
	}
	if _, err := h.Queries.GetCourseByID(c.Request.Context(), courseID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}

	var body struct {
		ModuleIDs []int64 `json:"module_ids"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(body.ModuleIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "module_ids is required"})
		return
	}

	// De-dup the incoming IDs so we don't double-count in the response.
	seen := make(map[int64]struct{}, len(body.ModuleIDs))
	ids := make([]int64, 0, len(body.ModuleIDs))
	for _, id := range body.ModuleIDs {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	// Validate that every requested module belongs to this course and none
	// are mid-generation. Same guard as DeleteModuleHandler / ReorderModulesHandler.
	mods, err := h.Queries.GetModulesByCourseWithStatus(c.Request.Context(), courseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load modules"})
		return
	}
	statusByID := make(map[int64]string, len(mods))
	for _, m := range mods {
		statusByID[m.ID] = m.Status
	}
	for _, id := range ids {
		if statusByID[id] == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "module does not belong to this course"})
			return
		}
		if statusByID[id] == "generating" {
			c.JSON(http.StatusConflict, gin.H{"error": "Cannot delete a module that is currently generating"})
			return
		}
	}

	// Items are ON DELETE SET NULL on module_id, so we have to delete them
	// explicitly to actually purge content. Both deletes run in one tx.
	tx, err := h.Pool.Begin(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback(c.Request.Context())
	txQueries := h.Queries.WithTx(tx)

	if err := txQueries.DeleteCourseItemsByModules(c.Request.Context(), ids); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete module items"})
		return
	}
	if err := txQueries.DeleteModules(c.Request.Context(), courseID, ids); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete modules"})
		return
	}
	if err := tx.Commit(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit deletion"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted", "module_ids": ids})
}

// ---- Learning path generation ----

// learningPathGenPrompt instructs the LLM to act as a curriculum designer
// that assembles a learning path by selecting from an EXISTING catalog of
// approved courses. It must only reference course ids we actually provide.
const learningPathGenPrompt = `You are an expert curriculum designer assembling a learning path from an EXISTING catalog of approved courses.

You will receive:
- The author's goal or topic for the learning path (may be vague)
- A catalog of available courses, each with an id, title, and description

Your job:
1. Choose a compelling, specific title and a rich description for the learning path (see the DESCRIPTION rules below).
2. Select a sensible sequence of courses from the catalog that together achieve the goal. Order them from foundational to advanced.
3. For each selected course, decide whether it is REQUIRED (core to the goal) or OPTIONAL (a useful supplement).
4. For each selected course, give a one-sentence rationale for why it's included at that point in the sequence.

DESCRIPTION RULES (the path's description field — write this carefully):
- Write 3-5 sentences. This is the first thing learners read; it must sell the path and make the outcome concrete.
- Open with who the path is for and the concrete outcome they'll be able to achieve (e.g. "By the end, you'll be able to...").
- Reference the actual skills/concepts covered by the courses you selected — ground it in what's really in the path, not generic filler.
- Convey the learning arc: how the path moves from foundations toward the final outcome.
- Be specific and confident. Avoid empty phrases like "a comprehensive learning journey", "unlock your potential", "dive deep", "in today's world".
- Do NOT list the course titles in the description (they're shown separately); synthesize the skills/themes instead.

CRITICAL RULES:
- ONLY reference course ids that appear in the provided catalog. Never invent course ids or titles.
- If the catalog has no courses relevant to the goal, return an empty courses array and explain in the summary.
- Do not select more courses than necessary — a tight, well-sequenced path is better than a bloated one. Aim for 3-8 courses unless the goal clearly warrants more.
- Output ONLY valid JSON, no markdown fences, no surrounding prose.

OUTPUT FORMAT:
{
  "title": "Learning Path Title",
  "description": "3-5 sentences: who it's for, the concrete outcome, the skills/themes drawn from the selected courses, and the learning arc.",
  "summary": "1-2 sentence note on the overall design / any gaps in the catalog.",
  "courses": [
    {"course_id": 123, "title": "Course Title", "is_required": true, "rationale": "Why this course, here, in one sentence."}
  ]
}`

// GenerateLearningPath takes a goal/topic and the approved course catalog, asks
// the LLM to assemble a path (title + description + sequenced course
// selection with rationale), and returns the suggestion for user review.
// It does NOT persist anything — the frontend creates the path + adds courses
// only after the user accepts.
// POST /api/learning-paths/generate  { goal: string }
func (h *Handler) GenerateLearningPath(c *gin.Context) {
	var body struct {
		Goal string `json:"goal"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	goal := strings.TrimSpace(body.Goal)
	if goal == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A goal or topic is required"})
		return
	}

	// Fetch the approved+published course catalog.
	courses, err := h.Queries.GetPublishedCourses(c.Request.Context())
	if err != nil {
		log.Printf("[lp-generate] fetch courses: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load course catalog"})
		return
	}
	if len(courses) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"title":       "",
			"description": "",
			"summary":     "There are no approved courses in the catalog yet. Create and approve some courses first, then try again.",
			"courses":     []any{},
		})
		return
	}

	// Build a compact catalog context. Include ids so the model can reference them.
	var catalogLines []string
	for _, c2 := range courses {
		desc := strings.ReplaceAll(c2.Description, "\n", " ")
		if len(desc) > 200 {
			desc = desc[:200] + "..."
		}
		catalogLines = append(catalogLines, fmt.Sprintf("- [id=%d] %s — %s", c2.ID, c2.Title, desc))
	}
	catalogCtx := strings.Join(catalogLines, "\n")

	userPrompt := fmt.Sprintf(`Author's goal or topic for the learning path:
"%s"

Available course catalog (%d courses):
%s

Assemble a learning path that achieves the goal using ONLY courses from the catalog above.`,
		goal, len(courses), catalogCtx)

	resp, err := h.LLM.Chat(learningPathGenPrompt, userPrompt)
	if err != nil {
		log.Printf("[lp-generate] llm: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate learning path"})
		return
	}

	jsonStr := stripMarkdownFences(resp)
	var result struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Summary     string `json:"summary"`
		Courses     []struct {
			CourseID   int64  `json:"course_id"`
			Title      string `json:"title"`
			IsRequired bool   `json:"is_required"`
			Rationale  string `json:"rationale"`
		} `json:"courses"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		log.Printf("[lp-generate] parse: %v — raw: %s", err, resp)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse generation response"})
		return
	}

	// Validate course ids against the catalog so a hallucinated id can never
	// reach the create step. Drop anything not in the catalog.
	validIDs := make(map[int64]bool, len(courses))
	for _, c2 := range courses {
		validIDs[c2.ID] = true
	}
	cleaned := result.Courses[:0]
	for _, sel := range result.Courses {
		if validIDs[sel.CourseID] {
			cleaned = append(cleaned, sel)
		}
	}
	result.Courses = cleaned

	c.JSON(http.StatusOK, gin.H{
		"title":       result.Title,
		"description": result.Description,
		"summary":     result.Summary,
		"courses":     result.Courses,
	})
}

// RegisterRoutes adds course generation routes.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	// Light rate limiting on the LLM-costly generation routes to prevent a
	// single client from saturating the DeepSeek budget (audit item H3).
	genLimit := middlewares.RateLimitByIP(10, 5)
	r.POST("/courses/generate", genLimit, middlewares.WrapRequireRole(h.GenerateCourse, "content creator"))
	r.POST("courses/discover", genLimit, middlewares.WrapRequireRole(h.Discover, "content creator"))
	r.POST("/courses/outline", genLimit, middlewares.WrapRequireRole(h.GenerateOutline, "content creator"))
	r.POST("/courses/:id/outline/regenerate", genLimit, middlewares.WrapRequireRole(h.RegenerateOutline, "content creator"))
	r.GET("/courses/generate/active", h.ListActiveJobs)
	r.POST("/courses/generate/:id/cancel", middlewares.WrapRequireRole(h.CancelGeneration, "content creator"))
	r.GET("/courses/generate/:id", h.GetJobStatus)
	r.GET("/courses/generate/:id/stream", h.StreamJob)

	// Learning path generation (assembles a path from the approved course catalog).
	r.POST("/learning-paths/generate", middlewares.WrapRequireRole(h.GenerateLearningPath, "content creator"))
	r.GET("/courses", h.ListCourses)
	r.GET("/courses/:id", h.GetCourse)
	r.GET("/courses/:id/preview", h.PreviewCourse)
	r.PUT("/courses/:id/edit", middlewares.WrapRequireRole(h.EditCourse, "content creator"))
	r.PUT("/courses/:id/settings", middlewares.WrapRequireRole(h.UpdateSettings, "content creator"))
	r.PUT("/courses/:id/meta", middlewares.WrapRequireRole(h.UpdateCourseMetaHandler, "content creator"))

	// Staged course builder: per-module management.
	r.POST("/courses/:id/modules", middlewares.WrapRequireRole(h.CreateModuleHandler, "content creator"))
	r.PUT("/courses/:id/modules/reorder", middlewares.WrapRequireRole(h.ReorderModulesHandler, "content creator"))
	r.POST("/courses/:id/modules/:moduleId/generate", middlewares.WrapRequireRole(h.GenerateModule, "content creator"))
	r.PUT("/courses/:id/modules/:moduleId", middlewares.WrapRequireRole(h.UpdateModuleHandler, "content creator"))
	r.DELETE("/courses/:id/modules/:moduleId", middlewares.WrapRequireRole(h.DeleteModuleHandler, "content creator"))
	r.DELETE("/courses/:id/modules", middlewares.WrapRequireRole(h.DeleteModulesHandler, "content creator"))

	r.POST("/items/:itemId/ai-edit", middlewares.WrapRequireRole(h.AiEditItem, "content creator"))
	r.PUT("/items/:itemId", middlewares.WrapRequireRole(h.UpdateItemData, "content creator"))
	r.DELETE("/items/:itemId", middlewares.WrapRequireRole(h.DeleteItem, "content creator"))
}

// EditCourse modifies an existing course based on AI-driven edit instructions.
func (h *Handler) EditCourse(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid course ID"})
		return
	}

	var body struct {
		Instructions string `json:"instructions"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.Instructions) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Instructions are required"})
		return
	}

	// Fetch existing course structure
	course, err := h.Queries.GetCourseByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}
	modules, _ := h.Queries.GetModulesByCourse(c.Request.Context(), id)
	items, _ := h.Queries.GetCourseItemsByCourse(c.Request.Context(), id)

	// Build current course JSON
	itemMap := make(map[int64][]gin.H)
	for _, item := range items {
		mid := item.ModuleID.Int64
		itemMap[mid] = append(itemMap[mid], gin.H{
			"item_type":  item.ItemType,
			"sort_order": item.SortOrder,
			"data":       json.RawMessage(item.Data),
		})
	}
	modulesJSON := make([]gin.H, 0)
	for _, m := range modules {
		modItems := itemMap[m.ID]
		if modItems == nil {
			modItems = []gin.H{}
		}
		modulesJSON = append(modulesJSON, gin.H{
			"title":       m.Title,
			"description": m.Description,
			"items":       modItems,
		})
	}
	currentJSON, _ := json.Marshal(gin.H{
		"title":       course.Title,
		"description": course.Description,
		"modules":     modulesJSON,
	})

	// Build edit prompt
	editPrompt := fmt.Sprintf(`Here is an existing course in JSON format:

%s

Edit instructions: %s

Apply these edits and return the FULL modified course JSON (not just the changes). Follow the same structure.`, string(currentJSON), body.Instructions)

	log.Printf("[ai] editing course %d: %s", id, body.Instructions)
	response, err := h.LLM.Chat(editCourseSystemPrompt, editPrompt)
	if err != nil {
		log.Printf("[ai] edit llm: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI edit failed"})
		return
	}

	// Parse modified course
	jsonStr := stripMarkdownFences(response)
	var gen genCourse
	if err := json.Unmarshal([]byte(jsonStr), &gen); err != nil {
		log.Printf("[ai] parse edit json: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse AI response"})
		return
	}

	// Begin transaction for the delete+recreate operation
	tx, err := h.Pool.Begin(c.Request.Context())
	if err != nil {
		log.Printf("[ai] begin tx: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback(c.Request.Context())
	txQueries := h.Queries.WithTx(tx)

	// Update course title/description within tx
	_, err = txQueries.UpdateCourseMeta(c.Request.Context(), database.UpdateCourseMetaParams{
		ID:          id,
		Title:       gen.Title,
		Description: gen.Description,
	})
	if err != nil {
		log.Printf("[ai] update course: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update course"})
		return
	}

	// Replace modules and items within tx
	txQueries.DeleteCourseItems(c.Request.Context(), id)
	txQueries.DeleteCourseModules(c.Request.Context(), id)

	modulesResult := make([]gin.H, 0)
	for mi, mod := range gen.Modules {
		module, err := txQueries.CreateModule(c.Request.Context(), database.CreateModuleParams{
			CourseID:    id,
			Title:       mod.Title,
			Description: mod.Description,
			SortOrder:   int32(mi),
		})
		if err != nil {
			log.Printf("[ai] create module: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create module"})
			return
		}
		itemsResult := make([]gin.H, 0)
		for ii, item := range mod.Items {
			itemData := ensureIRTParams(item.Data, item.Type)
			ci, err := txQueries.CreateCourseItem(c.Request.Context(), database.CreateCourseItemParams{
				CourseID:  id,
				ModuleID:  pgtype.Int8{Int64: module.ID, Valid: true},
				ItemType:  item.Type,
				SortOrder: int32(ii),
				Data:      itemData,
			})
			if err != nil {
				log.Printf("[ai] create item: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create item"})
				return
			}
			itemsResult = append(itemsResult, gin.H{
				"id":         ci.ID,
				"item_type":  ci.ItemType,
				"sort_order": ci.SortOrder,
				"data":       json.RawMessage(ci.Data),
			})
		}
		modulesResult = append(modulesResult, gin.H{
			"id":          module.ID,
			"title":       module.Title,
			"description": module.Description,
			"sort_order":  module.SortOrder,
			"items":       itemsResult,
		})
	}

	// Commit transaction
	if err := tx.Commit(c.Request.Context()); err != nil {
		log.Printf("[ai] commit tx: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit changes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          id,
		"title":       gen.Title,
		"description": gen.Description,
		"status":      course.Status,
		"modules":     modulesResult,
	})
}

// AiEditItem returns an AI-suggested edit for a single course item.
// This does NOT persist to the database — the frontend shows a preview
// and the user must accept it via UpdateItemData.
func (h *Handler) AiEditItem(c *gin.Context) {
	itemID, err := strconv.ParseInt(c.Param("itemId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	var body struct {
		Instructions string `json:"instructions"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.Instructions) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Instructions are required"})
		return
	}

	// Fetch the current item
	item, err := h.Queries.GetCourseItemByID(c.Request.Context(), itemID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}

	// Build the item JSON for the LLM
	currentItem := gin.H{
		"item_type": item.ItemType,
		"data":      json.RawMessage(item.Data),
	}
	currentJSON, _ := json.Marshal(currentItem)

	editPrompt := fmt.Sprintf(`Here is a course item in JSON format:

%s

Edit instructions: %s

Apply these edits and return ONLY the modified JSON for this single item.`, string(currentJSON), body.Instructions)

	log.Printf("[ai] editing item %d: %s", itemID, body.Instructions)
	response, err := h.LLM.Chat(editItemSystemPrompt, editPrompt)
	if err != nil {
		log.Printf("[ai] edit item llm: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI edit failed"})
		return
	}

	jsonStr := stripMarkdownFences(response)
	var suggested genItem
	if err := json.Unmarshal([]byte(jsonStr), &suggested); err != nil {
		log.Printf("[ai] parse edit item json: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse AI response"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"item_id":        item.ID,
		"item_type":      item.ItemType,
		"current_data":   json.RawMessage(item.Data),
		"suggested_type": suggested.Type,
		"suggested_data": suggested.Data,
	})
}

// UpdateItemData persists a modified item's data to the database.
func (h *Handler) UpdateItemData(c *gin.Context) {
	itemID, err := strconv.ParseInt(c.Param("itemId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	var body struct {
		Data json.RawMessage `json:"data"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || len(body.Data) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data is required"})
		return
	}

	// Verify item exists
	item, err := h.Queries.GetCourseItemByID(c.Request.Context(), itemID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}

	itemData := ensureIRTParams(body.Data, item.ItemType)
	updated, err := h.Queries.UpdateCourseItemData(c.Request.Context(), database.UpdateCourseItemDataParams{
		ID:   itemID,
		Data: itemData,
	})
	if err != nil {
		log.Printf("[ai] update item data: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update item"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         updated.ID,
		"item_type":  updated.ItemType,
		"sort_order": updated.SortOrder,
		"data":       json.RawMessage(updated.Data),
	})
}

// DeleteItem removes a course item from the database and renumbers its siblings.
func (h *Handler) DeleteItem(c *gin.Context) {
	itemID, err := strconv.ParseInt(c.Param("itemId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	// Fetch the item so we know which module to renumber
	item, err := h.Queries.GetCourseItemByID(c.Request.Context(), itemID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}

	if err := h.Queries.DeleteCourseItemByID(c.Request.Context(), itemID); err != nil {
		log.Printf("[ai] delete item: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete item"})
		return
	}

	// Renumber remaining items in the same module so sort_order stays contiguous
	if item.ModuleID.Valid {
		siblings, _ := h.Queries.GetCourseItemsByModule(c.Request.Context(), item.ModuleID)
		for i, sib := range siblings {
			if sib.SortOrder != int32(i) {
				h.Queries.UpdateCourseItemModule(c.Request.Context(), database.UpdateCourseItemModuleParams{
					ID:        sib.ID,
					ModuleID:  sib.ModuleID,
					SortOrder: int32(i),
				})
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"id": itemID, "deleted": true})
}

// ---- helpers ----

func stripMarkdownFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
	}
	if strings.HasSuffix(s, "```") {
		s = strings.TrimSuffix(s, "```")
	}
	return strings.TrimSpace(s)
}

// parsePlanMarkdown extracts a course plan from Step 1's Markdown output.
//
// Expected format:
//
//	# Course Title: [Title]
//	[Description paragraph(s)]
//
//	## Module 1: [Module Title]
//	**Description:** [Description]
//
//	## Module 2: [Module Title]
//	**Description:** [Description]
func parsePlanMarkdown(md string) plan {
	result := plan{}

	// Split on "## Module" to find module boundaries.
	// Everything before the first "## Module" is the course header.
	moduleRe := regexp.MustCompile(`(?m)^## Module \d+:\s*`)
	locs := moduleRe.FindAllStringIndex(md, -1)

	var headerSection string
	var moduleSections []string

	if len(locs) == 0 {
		// Fallback: treat the whole thing as header
		headerSection = md
	} else {
		headerSection = md[:locs[0][0]]
		for i, loc := range locs {
			end := len(md)
			if i+1 < len(locs) {
				end = locs[i+1][0]
			}
			moduleSections = append(moduleSections, md[loc[0]:end])
		}
	}

	// Parse course title from header
	titleRe := regexp.MustCompile(`(?m)^#\s+(?:Course Title:\s*)?(.+)$`)
	if match := titleRe.FindStringSubmatch(headerSection); match != nil {
		result.Title = strings.TrimSpace(match[1])
	}

	// Parse course description: everything in header that's not the title line
	descLines := strings.Split(headerSection, "\n")
	var descBuilder strings.Builder
	for _, line := range descLines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		descBuilder.WriteString(trimmed)
		descBuilder.WriteString("\n")
	}
	result.Description = strings.TrimSpace(descBuilder.String())

	// Parse each module
	for _, sec := range moduleSections {
		pm := planModule{}

		// Extract module title from the "## Module N: Title" line
		modTitleRe := regexp.MustCompile(`^## Module \d+:\s*(.+)$`)
		lines := strings.Split(sec, "\n")
		for _, line := range lines {
			if match := modTitleRe.FindStringSubmatch(strings.TrimSpace(line)); match != nil {
				pm.Title = strings.TrimSpace(match[1])
				break
			}
		}

		// Extract description from **Description:** line
		descRe := regexp.MustCompile(`\*\*Description:\*\*\s*(.+)`)
		for _, line := range lines {
			if match := descRe.FindStringSubmatch(line); match != nil {
				pm.Description = strings.TrimSpace(match[1])
				break
			}
		}

		// Also try without bold markers (some LLMs drop the **)
		if pm.Description == "" {
			plainDescRe := regexp.MustCompile(`Description:\s*(.+)`)
			for _, line := range lines {
				if match := plainDescRe.FindStringSubmatch(line); match != nil {
					pm.Description = strings.TrimSpace(match[1])
					break
				}
			}
		}

		if pm.Title != "" {
			result.Modules = append(result.Modules, pm)
		}
	}

	return result
}

func float64ToFloat32(in []float64) []float32 {
	out := make([]float32, len(in))
	for i, v := range in {
		out[i] = float32(v)
	}
	return out
}
