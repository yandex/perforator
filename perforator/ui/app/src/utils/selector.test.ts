import { describe, expect, it } from '@jest/globals';

import { cutIdFromSelector, cutTimeFromSelector, insertStatementIntoSelector, parseTimestampFromSelector, validateSelectorContainsOnlyService } from './selector';


const selector = '{service="perforator.perforator-proxy-prod",timestamp>="2024-08-26T09:56:12.624Z", timestamp<="2024-08-27T09:56:12.625Z"}';


describe('well-known field parser', () => {
    it('should parse timestamp', () => {
        expect(parseTimestampFromSelector(selector)).toEqual({
            from: '2024-08-26T09:56:12.624Z',
            to: '2024-08-27T09:56:12.625Z',
        });
    });
});

describe('insertStatementIntoSelector', () => {
    it('should insert statement into selector', () => {
        expect(insertStatementIntoSelector(selector, 'smbhElseSel="smbhElse"')).toEqual('{service="perforator.perforator-proxy-prod",timestamp>="2024-08-26T09:56:12.624Z", timestamp<="2024-08-27T09:56:12.625Z",smbhElseSel="smbhElse"}');
    });
    it('should insert statement into selector with trailing comma', () => {
        expect(insertStatementIntoSelector('{service="perforator.perforator-proxy-prod",}', 'smbhElseSel="smbhElse"')).toEqual('{service="perforator.perforator-proxy-prod",smbhElseSel="smbhElse"}');
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

describe('cutIdFromSelector', () => {
    it('should cut id from the end', () => {
        const s = '{service="perforator.perforator-proxy-prod",id="profile-id"}';

        expect(cutIdFromSelector(s)).toEqual('{service="perforator.perforator-proxy-prod"}');
    });

    it('should cut id from the middle', () => {
        const s = '{service="perforator.perforator-proxy-prod", id = "profile-id",event_type="cpu"}';

        expect(cutIdFromSelector(s)).toEqual('{service="perforator.perforator-proxy-prod",event_type="cpu"}');
    });

    it('should cut id from the beginning', () => {
        const s = '{id = "profile-id",event_type="cpu",service="perforator.perforator-proxy-prod"}';

        expect(cutIdFromSelector(s)).toEqual('{event_type="cpu",service="perforator.perforator-proxy-prod"}');
    });

    it('should cut id when it is the only condition', () => {
        const s = '{id="profile-id"}';

        expect(cutIdFromSelector(s)).toEqual('{}');
    });

    it('should not cut node_id or pod_id', () => {
        const s = '{node_id="node-id",pod_id="pod-id"}';

        expect(cutIdFromSelector(s)).toEqual(s);
    });

    it('should not split conditions by comma inside quoted values', () => {
        const s = '{id="profile-id",service="perforator,proxy",event_type="cpu"}';

        expect(cutIdFromSelector(s)).toEqual('{service="perforator,proxy",event_type="cpu"}');
    });
});

describe('validateSelectorContainsOnlyService', () => {
    it('should be correct for selector without trailing comma', () => {
        const s = '{service="perforator.perforator-proxy-prod"}';

        expect(validateSelectorContainsOnlyService(s)).toBe(true);
    });
    it('should be correct for selector with trailing comma', () => {
        const s = '{service="perforator.perforator-proxy-prod" , }';

        expect(validateSelectorContainsOnlyService(s)).toBe(true);
    });

    it('should fail on selector without service', () => {
        const s = '{timestamp>="2024-08-26T09:56:12.624Z", timestamp<="2024-08-27T09:56:12.625Z"}';

        expect(validateSelectorContainsOnlyService(s)).toBe(false);
    });

    it('should fail on selector with other fields', () => {
        const s = '{service="perforator.perforator-proxy-prod",timestamp>="2024-08-26T09:56:12.624Z", timestamp<="2024-08-27T09:56:12.625Z",otherField="otherField"}';

        expect(validateSelectorContainsOnlyService(s)).toBe(false);
    });
});
