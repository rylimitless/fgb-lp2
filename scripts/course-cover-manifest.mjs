#!/usr/bin/env node
/**
 * Generate a course-cover manifest for Higgsfield.
 *
 * Usage:
 *   node scripts/course-cover-manifest.mjs http://localhost:5555/api/courses
 *
 * If the API is unavailable, paste course JSON via stdin:
 *   type courses.json | node scripts/course-cover-manifest.mjs
 */

const SOURCE = process.argv[2];

function slugify(title) {
  return String(title || '')
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-|-$/g, '');
}

function themeFor(title) {
  const t = String(title || '').toLowerCase();
  if (/cyber|security|fraud|infosec|information/.test(t)) return 'vigilance';
  if (/compliance|regulator|policy|aml|privacy|data protection/.test(t)) return 'documents and continuity';
  if (/risk|credit|portfolio/.test(t)) return 'data and pattern';
  if (/leader|executive|management|team/.test(t)) return 'trust and exchange';
  if (/customer|service|relationship|experience|retail/.test(t)) return 'trust and exchange';
  if (/banking|foundation|introduction|fundamental/.test(t)) return 'growth';
  return 'documents and continuity';
}

async function readInput() {
  if (SOURCE) {
    const res = await fetch(SOURCE, { headers: { Accept: 'application/json' } });
    if (!res.ok) throw new Error(`HTTP ${res.status} from ${SOURCE}`);
    return res.json();
  }
  const chunks = [];
  for await (const chunk of process.stdin) chunks.push(chunk);
  return JSON.parse(Buffer.concat(chunks).toString('utf8'));
}

const style = `Editorial photography, FGB Academy brand: refined, premium Caribbean / Jamaican financial-services aesthetic. Cinematic single-source light from the upper-left, soft gold-warm fill. Palette restricted to deep navy (#00548e), warm gold (#d6c47e), cream, and graphite. Shallow depth of field. No text, no logos. AVOID beach, steel drums, vacation cliches.`;

const courses = await readInput();
const rows = (Array.isArray(courses) ? courses : [])
  .filter((c) => c.status === 'published' || c.approved || c.title)
  .map((c) => {
    const slug = slugify(c.title);
    const theme = themeFor(c.title);
    return {
      title: c.title,
      slug,
      path: `static/brand/modules/${slug}.png`,
      theme,
      prompt: `${style} 16:9 course thumbnail for "${c.title}". Theme: ${theme}. Abstract macro or professional Jamaican banking scene appropriate to the theme, left side safe for text overlay, premium SaaS learning card.`
    };
  });

console.log(JSON.stringify(rows, null, 2));