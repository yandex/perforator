import * as fc from 'fast-check';

import { describe, expect, it } from '@jest/globals';

import { cloneRows, forEachNode, profileRowsArbitrary } from './__test-helpers__/profileRowsArbitrary';
import type { FormatNode, ProfileData } from './models/Profile';
import { __test__, calculateTopForTable } from './top';


const rows: ProfileData['rows'] = [
    [
        { parentIndex: -1, textId: 0, eventCount: 100, sampleCount: 100, baseEventCount: 100, baseSampleCount: 100 },
    ],
    [
        { parentIndex: 0, textId: 1, eventCount: 50, sampleCount: 50, baseEventCount: 25, baseSampleCount: 25 },
        { parentIndex: 0, textId: 2, eventCount: 50, sampleCount: 50, baseEventCount: 75, baseSampleCount: 75 },
    ],
    [
        { parentIndex: 0, textId: 3, eventCount: 50, sampleCount: 50, baseEventCount: 25, baseSampleCount: 25 },
        { parentIndex: 1, textId: 2, eventCount: 49, sampleCount: 49, baseEventCount: 72, baseSampleCount: 72 },
    ],
    [
        { parentIndex: 1, textId: 4, eventCount: 0, sampleCount: 0, baseEventCount:2, baseSampleCount: 2 },
    ],
];

const profileData = { rows, stringTable: ['all', 'a', 'b', 'c', 'd', 'cycles', 'function'], meta: { version: 1, eventType: 5, frameType: 6 } };

describe('top', () => {
    const getTopData = () => calculateTopForTable(cloneRows(profileData.rows), profileData.stringTable.length).sort((a, b) => b.textId - a.textId);

    it('should work for example data', () => {
        const topData = getTopData();

        expect(topData).toMatchSnapshot();
    });
    it('there should be no function with more total events than root', () => {
        const topData = getTopData();
        const topEventCount = profileData.rows[0][0].eventCount;

        for (const topEntry of topData) {
            expect(topEntry['all.eventCount']).toBeLessThanOrEqual(topEventCount);
        }
    });
    it('there should be no self time on root', () => {
        const topData = getTopData();
        const rootTopData = topData.find((topEntry) => topEntry.textId === 0)!;

        expect(rootTopData['self.eventCount']).toBe(0);
    });
    it('recursive functions should be correctly accounted in total time', () => {
        const topData = getTopData();
        const topEntry = topData.find((entry) => entry.textId === 2)!;

        expect(topEntry['all.eventCount']).toBe(50);
    });
    it('should fall back to root for invalid root coordinates', () => {
        expect(() => calculateTopForTable(
            profileData.rows,
            profileData.stringTable.length,
            { rootCoords: [8, 1], omitted: [], keepCoords: null },
        )).not.toThrow();
    });

    it.each([
        {
            scenario: 'adjacent symbol ids with different inline flags',
            first: { textId: 1, inlined: true },
            second: { textId: 2, inlined: false },
        },
        {
            scenario: 'textId/file boundary',
            first: { kind: 0, file: 0, textId: 2, inlined: true },
            second: { kind: 0, file: 1, textId: 1, inlined: false },
        },
        {
            scenario: 'file/kind boundary',
            first: { kind: 0, file: 2, textId: 2, inlined: true },
            second: { kind: 1, file: 0, textId: 1, inlined: false },
        },
    ])('should keep distinct functions separate: $scenario', ({ first, second }) => {
        const distinctRows: ProfileData['rows'] = [
            [{ parentIndex: -1, textId: 0, eventCount: 100, sampleCount: 10 }],
            [
                { parentIndex: 0, ...first, eventCount: 30, sampleCount: 3 },
                { parentIndex: 0, ...second, eventCount: 70, sampleCount: 7 },
            ],
        ];

        const topData = calculateTopForTable(distinctRows, 3);

        expect(topData).toHaveLength(3);
        expect(topData.find((entry) => entry.textId === first.textId)).toMatchObject({
            ...first,
            'self.eventCount': 30,
            'all.eventCount': 30,
            'self.sampleCount': 3,
            'all.sampleCount': 3,
        });
        expect(topData.find((entry) => entry.textId === second.textId)).toMatchObject({
            ...second,
            'self.eventCount': 70,
            'all.eventCount': 70,
            'self.sampleCount': 7,
            'all.sampleCount': 7,
        });
    });

    it('should keep a focused child whose parent has an adjacent symbol id', () => {
        const focusedRows: ProfileData['rows'] = [
            [{ parentIndex: -1, textId: 0, eventCount: 100, sampleCount: 10 }],
            [{ parentIndex: 0, textId: 1, inlined: true, eventCount: 100, sampleCount: 10 }],
            [{ parentIndex: 0, textId: 2, inlined: false, eventCount: 70, sampleCount: 7 }],
        ];

        const topData = calculateTopForTable(focusedRows, 3, {
            rootCoords: [2, 0], omitted: [], keepCoords: null,
        });

        expect(topData).toHaveLength(1);
        expect(topData[0]).toMatchObject({
            textId: 2,
            inlined: false,
            'self.eventCount': 70,
            'all.eventCount': 70,
            'self.sampleCount': 7,
            'all.sampleCount': 7,
        });
    });

    it('should keep inline and non-inline occurrences of one symbol separate', () => {
        const inlineRows: ProfileData['rows'] = [
            [{ parentIndex: -1, textId: 0, eventCount: 100, sampleCount: 10 }],
            [
                { parentIndex: 0, textId: 1, inlined: true, eventCount: 30, sampleCount: 3 },
                { parentIndex: 0, textId: 1, inlined: false, eventCount: 70, sampleCount: 7 },
            ],
        ];

        const topData = calculateTopForTable(inlineRows, 2);
        const symbolEntries = topData.filter((entry) => entry.textId === 1);

        expect(symbolEntries).toHaveLength(2);
        expect(symbolEntries.find((entry) => entry.inlined)).toMatchObject({
            'self.eventCount': 30,
            'all.eventCount': 30,
        });
        expect(symbolEntries.find((entry) => !entry.inlined)).toMatchObject({
            'self.eventCount': 70,
            'all.eventCount': 70,
        });
    });

    it('should populate exact self counts for generated rows', () => {
        fc.assert(fc.property(profileRowsArbitrary(), ({ rows: generatedRows }) => {
            const rowsWithSelf = cloneRows(generatedRows);

            __test__.populateWithSelfEventCount(rowsWithSelf);

            forEachNode(rowsWithSelf, (node, h, i) => {
                const children = getChildren(rowsWithSelf, h, i);
                const expectedSelfEventCount = node.eventCount - sum(children, 'eventCount');
                const expectedSelfSampleCount = node.sampleCount - sum(children, 'sampleCount');
                const expectedBaseSelfEventCount = (node.baseEventCount ?? 0) - sum(children, 'baseEventCount');
                const expectedBaseSelfSampleCount = (node.baseSampleCount ?? 0) - sum(children, 'baseSampleCount');

                expect(node.selfEventCount).toBe(expectedSelfEventCount);
                expect(node.selfSampleCount).toBe(expectedSelfSampleCount);
                expect(node.baseSelfEventCount).toBe(expectedBaseSelfEventCount);
                expect(node.baseSelfSampleCount).toBe(expectedBaseSelfSampleCount);
                expect(node.selfEventCount).toBeGreaterThanOrEqual(0);
                expect(node.selfSampleCount).toBeGreaterThanOrEqual(0);
                expect(node.baseSelfEventCount).toBeGreaterThanOrEqual(0);
                expect(node.baseSelfSampleCount).toBeGreaterThanOrEqual(0);
            });
        }));
    });
    it('should keep generated top totals within root totals and clean temporary children sets', () => {
        fc.assert(fc.property(profileRowsArbitrary(), ({ rows: generatedRows, stringTable }) => {
            const topRows = cloneRows(generatedRows);
            const topData = calculateTopForTable(topRows, stringTable.length);
            const rootNode = topRows[0][0];

            for (const topEntry of topData) {
                expect(topEntry['all.eventCount']).toBeLessThanOrEqual(rootNode.eventCount);
                expect(topEntry['all.sampleCount']).toBeLessThanOrEqual(rootNode.sampleCount);
                expect(topEntry).not.toHaveProperty('shortestPath');
            }
            forEachNode(topRows, (node) => {
                expect(node.childrenIndices).toBeUndefined();
            });
        }));
    });
    it('should identify only strict prefixes as subpaths', () => {
        fc.assert(fc.property(
            fc.array(fc.nat(20), { maxLength: 20 }),
            fc.array(fc.nat(20), { minLength: 1, maxLength: 20 }),
            (prefix, suffix) => {
                expect(__test__.isSubpath(prefix.concat(suffix), prefix)).toBe(true);
                expect(__test__.isSubpath(prefix, prefix)).toBe(false);
                expect(__test__.isSubpath(prefix, prefix.concat(suffix))).toBe(false);
            },
        ));

        fc.assert(fc.property(
            fc.nat(20),
            fc.nat(20),
            fc.array(fc.nat(20), { maxLength: 20 }),
            (first, otherFirst, tail) => {
                fc.pre(first !== otherFirst);

                expect(__test__.isSubpath([first].concat(tail), [otherFirst])).toBe(false);
            },
        ));
    });
    it('should use the shallowest recursive occurrence for total time', () => {
        const recursiveRows: ProfileData['rows'] = [
            [
                { parentIndex: -1, textId: 0, eventCount: 100, sampleCount: 100 },
            ],
            [
                { parentIndex: 0, textId: 1, eventCount: 80, sampleCount: 80 },
                { parentIndex: 0, textId: 2, eventCount: 20, sampleCount: 20 },
            ],
            [
                { parentIndex: 0, textId: 1, eventCount: 60, sampleCount: 60 },
            ],
            [
                { parentIndex: 0, textId: 1, eventCount: 40, sampleCount: 40 },
            ],
        ];
        const topData = calculateTopForTable(recursiveRows, 3);
        const recursiveTopEntry = topData.find((entry) => entry.textId === 1)!;

        expect(recursiveTopEntry['all.eventCount']).toBe(80);
        expect(recursiveTopEntry['all.sampleCount']).toBe(80);
    });
});

function getChildren(rowsToRead: ProfileData['rows'], h: number, i: number) {
    return rowsToRead[h + 1]?.filter((node) => node.parentIndex === i) ?? [];
}

function sum(nodes: FormatNode[], field: 'eventCount' | 'sampleCount' | 'baseEventCount' | 'baseSampleCount') {
    return nodes.reduce((total, node) => total + (node[field] ?? 0), 0);
}
