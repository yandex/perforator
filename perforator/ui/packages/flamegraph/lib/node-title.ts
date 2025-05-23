import type { FormatNode } from './models/Profile';


export type ReadString = (id?: number) => string;

export type StringModifier = (s: string) => string;

export function getNodeTitleFull(readString: ReadString, shorten: StringModifier, shouldShorten: boolean, node: Pick<FormatNode, 'kind' | 'textId' | 'file' | 'inlined'>): string {
    const kind = readString(node.kind);
    let nodeTitle: string | undefined;
    const nodeText = readString(node.textId);
    if (shouldShorten) {
        nodeTitle = shorten(nodeText);
    } else {
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
