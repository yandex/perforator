import { describe, expect, it } from '@jest/globals';

import {
    makeBasicTokensFromSelector,
    makeBasicTokensFromService,
    makeSelectorFromTokensString,
    parseSelectorToTokensString,
    parseTokensString,
    serializeTokens,
} from './utils';


describe('TokensInput utils', () => {
    describe('parseTokensString', () => {
        it('should parse serialized tokens', () => {
            expect(parseTokensString('[["service","=","perforator"],["cpu","=","AMD EPYC"]]')).toEqual([
                { field: 'service', operator: '=', value: 'perforator' },
                { field: 'cpu', operator: '=', value: 'AMD EPYC' },
            ]);
        });

        it('should parse empty tokens from missing data', () => {
            expect(parseTokensString(undefined)).toEqual([]);
        });
    });

    describe('serializeTokens', () => {
        it('should serialize tokens preserving spaces in values', () => {
            expect(serializeTokens([
                { field: 'cpu', operator: '=', value: 'AMD EPYC' },
            ])).toBe('[["cpu","=","AMD EPYC"]]');
        });
    });

    describe('makeSelectorFromTokensString', () => {
        it('should make selector preserving spaces in values', () => {
            expect(makeSelectorFromTokensString('[["cpu","=","AMD EPYC"]]'))
                .toBe('{cpu="AMD EPYC"}');
        });
    });

    describe('parseSelectorToTokensString', () => {
        it('should parse selector preserving spaces in values', () => {
            expect(parseSelectorToTokensString('{cpu="AMD EPYC"}'))
                .toBe('[["cpu","=","AMD EPYC"]]');
        });

        it('should parse several selector conditions with spacing around operators', () => {
            expect(parseSelectorToTokensString('{service = "perforator", cpu = "AMD EPYC", event_type!="wall"}'))
                .toBe('[["service","=","perforator"],["cpu","=","AMD EPYC"],["event_type","!=","wall"]]');
        });

        it('should parse empty selector as empty tokens', () => {
            expect(parseSelectorToTokensString('{}')).toBe('[]');
        });
    });

    describe('selector tokens round trip', () => {
        it('should preserve spaces in values when converting selector to tokens and back', () => {
            const selector = '{cpu="AMD EPYC"}';

            expect(makeSelectorFromTokensString(parseSelectorToTokensString(selector))).toBe(selector);
        });
    });

    describe('basic tokens helpers', () => {
        it('should make basic tokens from selector preserving spaces in service value', () => {
            expect(makeBasicTokensFromSelector('{service="service with spaces"}'))
                .toBe('[["service","=","service with spaces"]]');
        });

        it('should make basic tokens from service preserving spaces in service value', () => {
            expect(makeBasicTokensFromService('service with spaces'))
                .toBe('[["service","=","service with spaces"]]');
        });
    });
});
