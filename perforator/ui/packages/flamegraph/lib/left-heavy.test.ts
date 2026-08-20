import { fc, it } from '@fast-check/jest';
import { describe, expect } from '@jest/globals';

import { cloneRows, profileRowsArbitrary } from './__test-helpers__/profileRowsArbitrary';
import { createLeftHeavy, inverseLeftHeavy, validateIsLeftHeavy } from './left-heavy';
import type { FormatNode, ProfileData } from './models/Profile';


const rows: ProfileData['rows'] = [
    [
        { parentIndex: -1, textId: 0, eventCount: 125, sampleCount: 125 },
    ],
    [
        { parentIndex: 0, textId: 1, eventCount: 50, sampleCount: 50 },
        { parentIndex: 0, textId: 2, eventCount: 75, sampleCount: 75 },
    ],
    [
        { parentIndex: 0, textId: 2, eventCount: 49, sampleCount: 49 },
        { parentIndex: 1, textId: 3, eventCount: 50, sampleCount: 50 },
    ],
];

const profileData = { rows, stringTable: ['all', 'a', 'b', 'c', 'd', 'cycles', 'function'], meta: { version: 1, eventType: 5, frameType: 6 } };
const sortByEventCount = (a: FormatNode, b: FormatNode) => b.eventCount - a.eventCount;

describe('left-heavy', () => {
    it('should work for example data', () => {
        const leftHeavy = createLeftHeavy(JSON.parse(JSON.stringify((profileData.rows))));
        expect(leftHeavy).toMatchInlineSnapshot(`
[
  [
    {
      "eventCount": 125,
      "parentIndex": -1,
      "sampleCount": 125,
      "textId": 0,
    },
  ],
  [
    {
      "eventCount": 75,
      "parentIndex": 0,
      "sampleCount": 75,
      "textId": 2,
    },
    {
      "eventCount": 50,
      "parentIndex": 0,
      "sampleCount": 50,
      "textId": 1,
    },
  ],
  [
    {
      "eventCount": 50,
      "parentIndex": 0,
      "sampleCount": 50,
      "textId": 3,
    },
    {
      "eventCount": 49,
      "parentIndex": 1,
      "sampleCount": 49,
      "textId": 2,
    },
  ],
]
`);
    });
    it('should be inversable', () => {
        const leftHeavy = createLeftHeavy(JSON.parse(JSON.stringify((profileData.rows))));
        const inversedLeftHeavy = inverseLeftHeavy(leftHeavy, profileData.stringTable);
        expect(inversedLeftHeavy).toEqual(profileData.rows);
    });
    it('should roundtrip generated canonical rows through left-heavy and inverse left-heavy', () => {
        fc.assert(fc.property(
            profileRowsArbitrary({ canonicalStringOrder: true, includeBaseCounts: false }),
            ({ rows: generatedRows, stringTable }) => {
                const originalRows = cloneRows(generatedRows);
                const leftHeavy = createLeftHeavy(cloneRows(generatedRows));
                const inversedLeftHeavy = inverseLeftHeavy(leftHeavy, stringTable);

                expect(inversedLeftHeavy).toEqual(originalRows);
            },
        ));
    });
    it('should be idempotent for generated rows', () => {
        fc.assert(fc.property(profileRowsArbitrary({ includeBaseCounts: false }), ({ rows: generatedRows }) => {
            const once = createLeftHeavy(cloneRows(generatedRows));
            const twice = createLeftHeavy(cloneRows(once));

            expect(twice).toEqual(once);
        }));
    });
    it('should produce rows that validate as left-heavy by event count', () => {
        fc.assert(fc.property(profileRowsArbitrary({ includeBaseCounts: false }), ({ rows: generatedRows }) => {
            const leftHeavy = createLeftHeavy(cloneRows(generatedRows));

            expect(validateIsLeftHeavy(leftHeavy, sortByEventCount)).toBe(true);
        }));
    });
    it('should preserve row sizes, per-level sums, and valid parent child totals', () => {
        fc.assert(fc.property(profileRowsArbitrary({ includeBaseCounts: false }), ({ rows: generatedRows }) => {
            const originalSummary = getRowsSummary(generatedRows);
            const leftHeavy = createLeftHeavy(cloneRows(generatedRows));

            expect(getRowsSummary(leftHeavy)).toEqual(originalSummary);
            expect(hasValidParentIndexes(leftHeavy)).toBe(true);
            expect(hasValidChildTotals(leftHeavy)).toBe(true);
        }));
    });
    it('should report old and new coordinates for reordered internal nodes', () => {
        const visitorCalls: Array<[number, number, number]> = [];
        const internalRows: ProfileData['rows'] = [
            [
                { parentIndex: -1, textId: 0, eventCount: 30, sampleCount: 30 },
            ],
            [
                { parentIndex: 0, textId: 1, eventCount: 10, sampleCount: 10 },
                { parentIndex: 0, textId: 2, eventCount: 20, sampleCount: 20 },
            ],
            [
                { parentIndex: 0, textId: 3, eventCount: 10, sampleCount: 10 },
                { parentIndex: 1, textId: 4, eventCount: 20, sampleCount: 20 },
            ],
        ];

        createLeftHeavy(internalRows, 'eventCount', (h, oldIndex, newIndex) => {
            visitorCalls.push([h, oldIndex, newIndex]);
        });

        expect(visitorCalls).toEqual(expect.arrayContaining([
            [1, 1, 0],
            [1, 0, 1],
        ]));
    });
});

function getRowsSummary(rowsToSummarize: ProfileData['rows']) {
    return rowsToSummarize.map((row) => ({
        length: row.length,
        eventCount: row.reduce((sum, node) => sum + node.eventCount, 0),
        sampleCount: row.reduce((sum, node) => sum + node.sampleCount, 0),
    }));
}

function hasValidParentIndexes(rowsToValidate: ProfileData['rows']) {
    return rowsToValidate.every((row, h) => row.every((node) => {
        if (h === 0) {
            return node.parentIndex === -1;
        }

        return node.parentIndex >= 0 && node.parentIndex < rowsToValidate[h - 1].length;
    }));
}

function hasValidChildTotals(rowsToValidate: ProfileData['rows']) {
    for (let h = 0; h < rowsToValidate.length - 1; h++) {
        const parentRow = rowsToValidate[h];
        const childRow = rowsToValidate[h + 1];
        const eventCounts = new Array<number>(parentRow.length).fill(0);
        const sampleCounts = new Array<number>(parentRow.length).fill(0);

        for (let i = 0; i < childRow.length; i++) {
            const child = childRow[i];
            eventCounts[child.parentIndex] += child.eventCount;
            sampleCounts[child.parentIndex] += child.sampleCount;
        }
        for (let i = 0; i < parentRow.length; i++) {
            if (eventCounts[i] > parentRow[i].eventCount || sampleCounts[i] > parentRow[i].sampleCount) {
                return false;
            }
        }
    }
    return true;
}
