import { describe, expect, it } from '@jest/globals';

import { shorten } from './shorten';
import { TEXT_SHORTENERS } from './shorteners';
import type { TextShortenerTestCase } from './TextShortener';


const COMMON_TESTS = [
    {
        input: 'root',
        expected: 'root',
    },
    {
        input: 'function @/main.cpp',
        expected: 'function',
    },
    {
        input: 'worker (container)',
        expected: 'worker (container)',
    },
];

const runTests = (testCases: Optional<TextShortenerTestCase[]>): void => {
    (testCases || []).forEach(({ input, expected }) => {
        it(`should shorten ${input} to ${expected}`, () => {
            expect(shorten(input)).toBe(expected);
        });
    });
};

describe('shorten frame name', () => {
    runTests(COMMON_TESTS);
    TEXT_SHORTENERS.forEach(({ testCases }) => runTests(testCases));
});
