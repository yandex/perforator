import { describe, expect, it } from '@jest/globals';

import { getFrameCoordinateFromQuery } from './query-utils';
import type { QueryKeys } from './renderer';


describe('query utils', () => {
    it('should parse frame coordinates by numeric prefix', () => {
        const query: Partial<Record<QueryKeys, string>> = {
            frameDepth: '1',
            framePos: '3(',
        };

        expect(getFrameCoordinateFromQuery((key, defaultValue) => query[key] ?? defaultValue)).toEqual([1, 3]);
    });
});
