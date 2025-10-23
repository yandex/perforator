import { describe, expect, it } from '@jest/globals';

import { createLeftHeavy, inverseLeftHeavy } from './left-heavy';
import type { ProfileData } from './models/Profile';


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
});
