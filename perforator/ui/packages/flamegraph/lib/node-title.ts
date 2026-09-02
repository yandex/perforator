import type { FormatNode } from './models/Profile';


export type ReadString = (id?: number) => string;

export type StringModifier = (s: string) => string;

type ModifierCaches = WeakMap<StringModifier, Map<number, string>>;

const SHORTENED_TEXT_CACHES = new WeakMap<ReadString, ModifierCaches>();

function getShortenedText(readString: ReadString, shorten: StringModifier, textId: number): string {
    let modifierCaches = SHORTENED_TEXT_CACHES.get(readString);
    if (modifierCaches === undefined) {
        modifierCaches = new WeakMap();
        SHORTENED_TEXT_CACHES.set(readString, modifierCaches);
    }
    let cache = modifierCaches.get(shorten);
    if (cache === undefined) {
        cache = new Map();
        modifierCaches.set(shorten, cache);
    }
    const cached = cache.get(textId);
    if (cached !== undefined) {
        return cached;
    }
    const shortened = shorten(readString(textId));
    cache.set(textId, shortened);
    return shortened;
}

export function getNodeTitleFull(readString: ReadString, shorten: StringModifier, shouldShorten: boolean, node: Pick<FormatNode, 'kind' | 'textId' | 'file' | 'inlined'>): string {
    const kind = readString(node.kind);
    let nodeTitle: string | undefined;
    if (shouldShorten) {
        nodeTitle = getShortenedText(readString, shorten, node.textId);
    } else {
        const nodeText = readString(node.textId);
        nodeTitle = nodeText + ' ' + readString(node.file);
    }
    if (kind !== '') {
        nodeTitle += ` (${kind})`;
    }
    if (node.inlined) {
        nodeTitle += ' (inlined)';
    }
    return nodeTitle;
}
