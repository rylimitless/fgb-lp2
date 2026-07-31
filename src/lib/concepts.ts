/**
 * Adaptive concept grouping helpers.
 *
 * The generator renders each testable concept in multiple question-type
 * variants (mc / tf / fb / ...) that share a `question_group_id`. These helpers
 * cluster those variants so the UI can present "N unique questions" instead of
 * a misleading raw item count, and collapse variant sets into reviewable blocks.
 *
 * Items without a question_group_id (content, legacy items, AI-edit output) are
 * treated as singletons — each is its own one-item group.
 */

export type AnyItem = {
    id: number | string;
    item_type: string;
    question_group_id?: string | null;
    [key: string]: unknown;
};

/**
 * A concept block: either a single standalone item (content, or an ungrouped
 * question) or a set of type-variants of the same concept.
 */
export type ConceptBlock<T extends AnyItem = AnyItem> = {
    /** Stable key (the question_group_id, or "item-<id>" for singletons). */
    key: string;
    /** True when this block holds >1 variant of one concept. */
    isVariantSet: boolean;
    /** Member items, in their original sort order. */
    items: T[];
};

/** Stable per-item key for items that aren't part of a variant group. */
function singletonKey(item: AnyItem): string {
    return `item-${item.id}`;
}

/** The concept key for an item — its group id, or its own id when ungrouped. */
export function conceptKey(item: AnyItem): string {
    return item.question_group_id || singletonKey(item);
}

/**
 * Counts distinct concepts across a list of items. Content items are excluded
 * (they're teaching material, not assessable concepts). Variants sharing a
 * group id count as one; ungrouped questions each count as one.
 */
export function uniqueConceptCount<T extends AnyItem>(items: T[]): number {
    const seen = new Set<string>();
    for (const item of items) {
        if (item.item_type === "content") continue;
        seen.add(conceptKey(item));
    }
    return seen.size;
}

/**
 * Groups items into concept blocks, preserving the order in which each group
 * FIRST appears. Content items and ungrouped questions become singleton
 * blocks; grouped variants cluster into one block each. This is what the
 * "all at once" preview iterates over so a concept's variants stay together.
 */
export function groupByConcept<T extends AnyItem>(items: T[]): ConceptBlock<T>[] {
    const order: string[] = [];
    const byKey = new Map<string, T[]>();
    for (const item of items) {
        const key = conceptKey(item);
        const bucket = byKey.get(key);
        if (bucket) {
            bucket.push(item);
        } else {
            byKey.set(key, [item]);
            order.push(key);
        }
    }
    return order.map((key) => {
        const blockItems = byKey.get(key)!;
        return { key, isVariantSet: blockItems.length > 1, items: blockItems };
    });
}
