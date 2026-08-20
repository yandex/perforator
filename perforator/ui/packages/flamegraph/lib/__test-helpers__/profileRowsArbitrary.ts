import * as fc from 'fast-check';

import type { FormatNode, ProfileData } from '../models/Profile';


export interface ProfileRowsArbitraryOptions {
    maxDepth?: number;
    maxBreadth?: number;
    maxNodes?: number;
    stringTableMinLength?: number;
    stringTableMaxLength?: number;
    canonicalStringOrder?: boolean;
    includeBaseCounts?: boolean;
}

export interface GeneratedProfileRows {
    rows: ProfileData['rows'];
    stringTable: ProfileData['stringTable'];
}

const DEFAULT_MAX_DEPTH = 6;
const DEFAULT_MAX_BREADTH = 5;
const DEFAULT_MAX_NODES = 128;
const DEFAULT_STRING_TABLE_MIN_LENGTH = 8;
const DEFAULT_STRING_TABLE_MAX_LENGTH = 32;
const SELF_COUNT_LIMIT = 50;

type MutableNode = FormatNode & {
    selfEventSeed: number;
    selfSampleSeed: number;
    baseSelfEventSeed?: number;
    baseSelfSampleSeed?: number;
};

export function cloneRows(rows: ProfileData['rows']): ProfileData['rows'] {
    return JSON.parse(JSON.stringify(rows));
}

export function forEachNode(rows: ProfileData['rows'], visitor: (node: FormatNode, h: number, i: number) => void) {
    for (let h = 0; h < rows.length; h++) {
        for (let i = 0; i < rows[h].length; i++) {
            visitor(rows[h][i], h, i);
        }
    }
}

export function profileRowsArbitrary(options: ProfileRowsArbitraryOptions = {}): fc.Arbitrary<GeneratedProfileRows> {
    const maxDepth = options.maxDepth ?? DEFAULT_MAX_DEPTH;
    const maxBreadth = options.maxBreadth ?? DEFAULT_MAX_BREADTH;
    const maxNodes = options.maxNodes ?? DEFAULT_MAX_NODES;
    const stringTableMinLength = options.stringTableMinLength ?? DEFAULT_STRING_TABLE_MIN_LENGTH;
    const stringTableMaxLength = options.stringTableMaxLength ?? DEFAULT_STRING_TABLE_MAX_LENGTH;

    return fc.record({
        stringTableLength: fc.integer({ min: stringTableMinLength, max: stringTableMaxLength }),
        seeds: fc.array(fc.nat(1_000_000), { minLength: 1, maxLength: maxNodes * 8 }),
    }).map(({ stringTableLength, seeds }) => {
        let cursor = 0;
        const nextSeed = () => seeds[cursor++] ?? 0;
        const nextCount = () => nextSeed() % (SELF_COUNT_LIMIT + 1);
        const nextTextId = () => nextSeed() % stringTableLength;

        const stringTable = Array.from({ length: stringTableLength }, (_, index) => `func_${String(index).padStart(3, '0')}`);
        const rows: MutableNode[][] = [[createNode(-1, 0, nextCount, options.includeBaseCounts ?? true)]];
        let nodeCount = 1;

        for (let h = 1; h <= maxDepth; h++) {
            const previousRow = rows[h - 1];
            const row: MutableNode[] = [];

            for (let parentIndex = 0; parentIndex < previousRow.length && nodeCount < maxNodes; parentIndex++) {
                const remainingNodes = maxNodes - nodeCount;
                const childCount = Math.min(nextSeed() % (maxBreadth + 1), remainingNodes);
                const textIds = options.canonicalStringOrder
                    ? createUniqueTextIds(childCount, stringTableLength, nextSeed)
                    : Array.from({ length: childCount }, nextTextId);

                if (options.canonicalStringOrder) {
                    textIds.sort((a, b) => stringTable[a].localeCompare(stringTable[b]));
                }

                for (let i = 0; i < childCount; i++) {
                    row.push(createNode(parentIndex, textIds[i], nextCount, options.includeBaseCounts ?? true));
                }
                nodeCount += childCount;
            }

            if (row.length === 0) {
                break;
            }
            rows.push(row);
        }

        populateCountsFromSelfSeeds(rows);

        return {
            rows: stripSelfSeeds(rows),
            stringTable,
        };
    });
}

function createNode(
    parentIndex: number,
    textId: number,
    nextCount: () => number,
    includeBaseCounts: boolean,
): MutableNode {
    const node: MutableNode = {
        parentIndex,
        textId,
        eventCount: 0,
        sampleCount: 0,
        selfEventSeed: nextCount(),
        selfSampleSeed: nextCount(),
    };

    if (includeBaseCounts) {
        node.baseSelfEventSeed = nextCount();
        node.baseSelfSampleSeed = nextCount();
    }

    return node;
}

function createUniqueTextIds(count: number, stringTableLength: number, nextSeed: () => number) {
    const start = nextSeed() % stringTableLength;
    return Array.from({ length: count }, (_, index) => (start + index) % stringTableLength);
}

function populateCountsFromSelfSeeds(rows: MutableNode[][]) {
    for (let h = rows.length - 1; h >= 0; h--) {
        for (let i = 0; i < rows[h].length; i++) {
            const node = rows[h][i];
            const children = rows[h + 1]?.filter((child) => child.parentIndex === i) ?? [];

            node.eventCount = node.selfEventSeed + children.reduce((sum, child) => sum + child.eventCount, 0);
            node.sampleCount = node.selfSampleSeed + children.reduce((sum, child) => sum + child.sampleCount, 0);

            if (node.baseSelfEventSeed !== undefined) {
                node.baseEventCount = node.baseSelfEventSeed + children.reduce((sum, child) => sum + (child.baseEventCount ?? 0), 0);
            }
            if (node.baseSelfSampleSeed !== undefined) {
                node.baseSampleCount = node.baseSelfSampleSeed + children.reduce((sum, child) => sum + (child.baseSampleCount ?? 0), 0);
            }
        }
    }
}

function stripSelfSeeds(rows: MutableNode[][]): ProfileData['rows'] {
    return rows.map((row) => row.map((node) => {
        const formatNode: FormatNode = {
            parentIndex: node.parentIndex,
            textId: node.textId,
            eventCount: node.eventCount,
            sampleCount: node.sampleCount,
        };

        if (node.baseEventCount !== undefined) {
            formatNode.baseEventCount = node.baseEventCount;
        }
        if (node.baseSampleCount !== undefined) {
            formatNode.baseSampleCount = node.baseSampleCount;
        }

        return formatNode;
    }));
}
