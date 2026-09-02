import { describe, expect, it, jest } from '@jest/globals';

import { getNodeTitleFull } from './node-title';


describe('getNodeTitleFull', () => {
    it('caches shortened text by string table, modifier, and text ID', () => {
        const strings = ['first', 'second'];
        const readString = (id?: number): string => id === undefined ? '' : strings[id];
        const shorten = jest.fn((text: string): string => `short:${text}`);
        const otherShorten = jest.fn((text: string): string => `other:${text}`);
        const first = { textId: 0 };
        const second = { textId: 1 };

        expect(getNodeTitleFull(readString, shorten, true, first)).toBe('short:first');
        expect(getNodeTitleFull(readString, shorten, true, first)).toBe('short:first');
        expect(getNodeTitleFull(readString, shorten, true, second)).toBe('short:second');
        expect(getNodeTitleFull(readString, otherShorten, true, first)).toBe('other:first');
        expect(shorten).toHaveBeenCalledTimes(2);
        expect(otherShorten).toHaveBeenCalledTimes(1);
    });

    it('does not share cached text between string tables', () => {
        const shorten = jest.fn((text: string): string => `short:${text}`);
        const firstReader = (id?: number): string => id === undefined ? '' : ['first'][id];
        const secondReader = (id?: number): string => id === undefined ? '' : ['second'][id];
        const node = { textId: 0 };

        expect(getNodeTitleFull(firstReader, shorten, true, node)).toBe('short:first');
        expect(getNodeTitleFull(secondReader, shorten, true, node)).toBe('short:second');
        expect(shorten).toHaveBeenCalledTimes(2);
    });
});
