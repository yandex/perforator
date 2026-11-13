import { afterEach, beforeEach, describe, expect, it, jest } from '@jest/globals';

import { LocalStorageKey } from 'src/const/localStorage';
import { uiFactory } from 'src/factory';

import { apiClient } from '../api';

import { startProfileTask } from './profileTask';


describe('startProfileTask', () => {
    afterEach(() => { jest.restoreAllMocks(); });
    beforeEach(() => {
        localStorage.clear();
        jest.spyOn(uiFactory(), 'wasmEnabled').mockReturnValue(true);
        jest.spyOn(apiClient, 'startTask').mockResolvedValue({ data: { TaskID: 'created-task' } } as never);
    });

    it('should request symbolized pprof and retain postprocessing', async () => {
        const query = { from: '', to: '', selector: '{service="test",timestamp>="2026-09-04T00:00:00Z",timestamp<="2026-09-04T01:00:00Z"}', maxProfiles: 100,
            format: 'raw' as const };
        expect(await startProfileTask(query, { pythonPrettifyLevel: 'mixed' })).toBe('created-task');
        const request = jest.mocked(apiClient.startTask).mock.calls[0][0];
        expect(request.Spec.MergeProfiles).toMatchObject({
            Format: { RawProfile: {}, Symbolize: { Symbolize: true }, Postprocessing: { MergePythonAndNativeStacks: true } },
            MaxSamples: 100, Query: { Selector: query.selector },
        });
        expect(request.Spec.MergeProfiles.Format.JSONFlamegraph).toBeUndefined();
    });

    it.each([false, true])('should pass the factory WASM flag %s to format selection', async enabled => {
        jest.spyOn(uiFactory(), 'wasmEnabled').mockReturnValue(enabled);
        await startProfileTask({ from: '', to: '', maxProfiles: 1 }, { mergeIntoPprofByDefault: true });

        const request = jest.mocked(apiClient.startTask).mock.calls[0][0];
        const format = request.Spec.MergeProfiles.Format;
        if (enabled) {
            expect(format.RawProfile).toEqual({});
            expect(format.JSONFlamegraph).toBeUndefined();
        } else {
            expect(format.JSONFlamegraph).toBeDefined();
            expect(format.RawProfile).toBeUndefined();
        }
    });

    it.each([undefined, 'json'] as const)('caches JSON line-number tasks with format %s', async format => {
        await startProfileTask({ from: '', to: '', maxProfiles: 1, format, prevTask: 'old-task' }, {});
        expect(JSON.parse(localStorage.getItem(LocalStorageKey.CachedLineNumberTasks)!)).toEqual({
            'old-task': 'created-task',
            'created-task': 'old-task',
        });
    });

    it.each([
        { format: 'raw' as const },
        { format: 'text' as const },
        { rawProfile: 'true' as const, format: 'json' as const },
        {},
    ])('does not cache non-JSON line-number tasks for %j', async query => {
        await startProfileTask({ from: '', to: '', maxProfiles: 1, prevTask: 'old-task', ...query }, { mergeIntoPprofByDefault: true });
        expect(localStorage.getItem(LocalStorageKey.CachedLineNumberTasks)).toBeNull();
    });
});
