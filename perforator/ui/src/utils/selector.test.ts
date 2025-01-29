import { describe, expect, it } from '@jest/globals';

import { cutTimeFromSelector, parseTimestampFromSelector } from './selector';


const selector = '{service="perforator.perforator-proxy-prod",timestamp>="2024-08-26T09:56:12.624Z", timestamp<="2024-08-27T09:56:12.625Z"}';


describe('well-known field parser', () => {
    it('should parse timestamp', () => {
        expect(parseTimestampFromSelector(selector)).toEqual({
            from: '2024-08-26T09:56:12.624Z',
            to: '2024-08-27T09:56:12.625Z',
        });
    });
});

describe('cutTimeFromSelector', () => {
    it('should cut timestamp from the end', () => {
        const s = '{service="perforator.perforator-proxy-prod",timestamp>="2024-08-26T09:56:12.624Z", timestamp<="2024-08-27T09:56:12.625Z"}';

        expect(cutTimeFromSelector(s)).toEqual('{service="perforator.perforator-proxy-prod"}');
    });
    it('should cut timestamp from the middle', () => {
        const s = '{service="perforator.perforator-proxy-prod",timestamp>="2024-08-26T09:56:12.624Z", timestamp<="2024-08-27T09:56:12.625Z",smbhElseSel="smbhElse"}';

        expect(cutTimeFromSelector(s)).toEqual('{service="perforator.perforator-proxy-prod",smbhElseSel="smbhElse"}');
    });
    it('should cut timestamp from the beginning', () => {
        const s = '{timestamp>="2024-08-26T09:56:12.624Z", timestamp<="2024-08-27T09:56:12.625Z",service="perforator.perforator-proxy-prod"}';

        expect(cutTimeFromSelector(s)).toEqual('{service="perforator.perforator-proxy-prod"}');
    });
    it('should not cut anything for selector without timestamp', () => {
        const s = '{service="perforator.perforator-proxy-prod"}';

        expect(cutTimeFromSelector(s)).toEqual(s);
    });
});
