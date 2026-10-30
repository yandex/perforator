import { afterEach, describe, expect, it, jest } from '@jest/globals';

import type * as Runtime from './runtime';


jest.mock('@discoveryjs/json-ext', () => ({ parseChunked: jest.fn() }));

const { PprofRuntime } = require('./runtime') as typeof Runtime;


describe('PprofRuntime initialization', () => {
    const originalFetch = globalThis.fetch;
    afterEach(() => {
        globalThis.fetch = originalFetch;
        jest.restoreAllMocks();
    });

    it('reports allocation failure before parsing and leaves the runtime reusable', async () => {
        const memory = new WebAssembly.Memory({ initial: 1 });
        const native = {
            memory,
            // eslint-disable-next-line camelcase -- Native export name.
            __wasm_call_ctors: jest.fn(),
            malloc: jest.fn<() => number>().mockReturnValueOnce(0).mockReturnValue(8),
            free: jest.fn(),
            PerforatorProfileParsePProf: jest.fn<() => number>().mockReturnValue(0),
        };
        jest.spyOn(WebAssembly, 'compile').mockResolvedValue({} as WebAssembly.Module);
        jest.spyOn(WebAssembly.Module, 'imports').mockReturnValue([]);
        jest.spyOn(WebAssembly, 'instantiate').mockResolvedValue({ exports: native } as unknown as WebAssembly.Instance);
        const runtime = await PprofRuntime.createFromBytes(new ArrayBuffer(0));
        expect(() => runtime.load(new ArrayBuffer(1))).toThrow('The profile engine could not allocate memory.');
        expect(native.PerforatorProfileParsePProf).not.toHaveBeenCalled();
        expect(() => runtime.load(new ArrayBuffer(1))).not.toThrow();
        expect(native.PerforatorProfileParsePProf).toHaveBeenCalledTimes(1);
    });

    it('reports an invalid module as a compile failure rather than an unsupported browser', async () => {
        globalThis.fetch = jest.fn<typeof fetch>().mockResolvedValue({
            ok: true,
            arrayBuffer: async () => new ArrayBuffer(0),
        } as Response);
        jest.spyOn(WebAssembly, 'compile').mockRejectedValue(new Error('invalid module'));
        await expect(PprofRuntime.create('/profile.wasm')).rejects.toThrow('Could not compile the profile engine: Error: invalid module');
    });
});
