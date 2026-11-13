import { describe, expect, it } from '@jest/globals';

import type { FlamegraphOptions } from 'src/generated/perforator/proto/perforator/perforator';
import type { ProfileTaskQuery } from 'src/models/Task';

import { getRenderFlamegraph } from './getRenderFlamegraph';


const flamegraphOptions: FlamegraphOptions = { MaxDepth: 256, MinWeight: 1e-10, ShowLineNumbers: true };

describe('getRenderFlamegraph', () => {
    it.each<{
        description: string;
        query: Partial<ProfileTaskQuery>;
        wasmEnabled: boolean;
        optIn: boolean;
        expected: ReturnType<typeof getRenderFlamegraph>;
    }>([
        { description: 'JSON by default', query: {}, wasmEnabled: true, optIn: false,
            expected: { JSONFlamegraph: flamegraphOptions } },
        { description: 'raw with opt-in', query: {}, wasmEnabled: true, optIn: true,
            expected: { RawProfile: {} } },
        { description: 'JSON when WASM is disabled despite opt-in', query: {}, wasmEnabled: false, optIn: true,
            expected: { JSONFlamegraph: flamegraphOptions } },
        { description: 'explicit raw when WASM is disabled', query: { format: 'raw' }, wasmEnabled: false, optIn: true,
            expected: { RawProfile: flamegraphOptions } },
        { description: 'explicit raw without opt-in', query: { format: 'raw' }, wasmEnabled: true, optIn: false,
            expected: { RawProfile: flamegraphOptions } },
        { description: 'explicit text despite opt-in', query: { format: 'text' }, wasmEnabled: true, optIn: true,
            expected: { TextProfile: flamegraphOptions } },
        { description: 'explicit JSON fallback despite opt-in', query: { format: 'json' }, wasmEnabled: true, optIn: true,
            expected: { JSONFlamegraph: flamegraphOptions } },
        { description: 'JSON for diffs despite opt-in', query: { diffSelector: '{service="other"}' }, wasmEnabled: true, optIn: true,
            expected: { JSONFlamegraph: flamegraphOptions } },
        { description: 'legacy rawProfile takes precedence over JSON', query: { rawProfile: 'true', format: 'json' }, wasmEnabled: false, optIn: false,
            expected: { RawProfile: flamegraphOptions } },
    ])('should select $description', ({ query, wasmEnabled, optIn, expected }) => {
        const result = getRenderFlamegraph({ from: '', to: '', maxProfiles: 1, ...query }, flamegraphOptions, {
            wasmEnabled,
            mergeIntoPprofByDefault: optIn,
        });

        expect(result).toEqual(expected);
    });
});
