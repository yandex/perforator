import { describe, expect, it, jest } from '@jest/globals';

import { shorten } from './shorten';
import { TEXT_SHORTENERS } from './shorteners';
import type { TextShortenerTestCase } from './TextShortener';


const COMMON_TESTS = [
    {
        input: 'root',
        expected: 'root',
    },
    {
        input: 'worker (container)',
        expected: 'worker (container)',
    },
];

const runTests = (testCases: TextShortenerTestCase[] | undefined): void => {
    (testCases || []).forEach(({ input, expected }) => {
        it(`should shorten ${input} to ${expected}`, () => {
            expect(shorten(input)).toBe(expected);
            expect(shorten(expected)).toBe(expected);
        });
    });
};

describe('shorten frame name', () => {
    runTests(COMMON_TESTS);
    TEXT_SHORTENERS.forEach(({ testCases }) => runTests(testCases));

    it('bypasses every language-specific parser for a plain C name', () => {
        const spies = TEXT_SHORTENERS.map(shortener => jest.spyOn(shortener, 'shorten'));
        try {
            expect(shorten('plain_c_function')).toBe('plain_c_function');
            spies.forEach(spy => expect(spy).not.toHaveBeenCalled());
        } finally {
            spies.forEach(spy => spy.mockRestore());
        }
    });

    it('bypasses C++ parsing while continuing to the Go shortener', () => {
        const cppShorten = jest.spyOn(TEXT_SHORTENERS[0], 'shorten');
        const goShorten = jest.spyOn(TEXT_SHORTENERS[1], 'shorten');
        try {
            expect(shorten('github.com/example/project.Function')).toBe('project.Function');
            expect(cppShorten).not.toHaveBeenCalled();
            expect(goShorten).toHaveBeenCalledTimes(1);
        } finally {
            cppShorten.mockRestore();
            goShorten.mockRestore();
        }
    });
});
