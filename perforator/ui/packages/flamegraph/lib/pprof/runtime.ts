/* eslint-disable camelcase, new-cap -- These names are fixed by the native C and WASI ABIs. */
import { parseChunked } from '@discoveryjs/json-ext';

import type { ProfileData } from '../models/Profile';

import { supportsPprofWasm, unsupportedPprofWasm } from './capabilities';
import type { PprofRenderOptions } from './options';
import { validatePprofOptions } from './options';
import type { PprofRenderResult } from './protocol';


type Pointer = number;
interface NativeExports {
    memory: WebAssembly.Memory;
    __wasm_call_ctors: () => void;
    malloc: (size: number) => Pointer;
    free: (pointer: Pointer) => void;
    PerforatorProfileParsePProf: (data: Pointer, size: number, result: Pointer) => Pointer;
    PerforatorProfileDispose: (profile: Pointer) => void;
    PerforatorWasmRender: (profile: Pointer, depth: number, weight: number, files: number, lines: number, result: Pointer) => Pointer;
    PerforatorStringData: (string: Pointer) => Pointer;
    PerforatorStringSize: (string: Pointer) => number;
    PerforatorStringDispose: (string: Pointer) => void;
    PerforatorErrorString: (error: Pointer) => Pointer;
    PerforatorErrorDispose: (error: Pointer) => void;
}

function offset(pointer: Pointer): number {
    // WASM i32 results are signed in JS; addresses above 2 GiB are unsigned.
    // eslint-disable-next-line no-bitwise -- Preserve the unsigned WASM32 address bits.
    return pointer >>> 0;
}

/** The only OS services used by this standalone, single-threaded module. */
function runtimeImports(memory: () => WebAssembly.Memory): WebAssembly.Imports {
    const view = () => new DataView(memory().buffer);
    return {
        env: {
            // All views below are recreated after native calls, including growth.
            emscripten_notify_memory_growth: () => {},
        },
        wasi_snapshot_preview1: {
            clock_time_get: (clock: number, _precision: bigint, result: Pointer) => {
                if (clock !== 0 && clock !== 1) {
                    return 28; // EINVAL
                }
                const ns = clock === 0 ? BigInt(Date.now()) * 1_000_000n : BigInt(Math.trunc(performance.now() * 1_000_000));
                view().setBigUint64(offset(result), ns, true);
                return 0;
            },
            environ_sizes_get: (count: Pointer, size: Pointer) => {
                view().setUint32(offset(count), 0, true);
                view().setUint32(offset(size), 0, true);
                return 0;
            },
            environ_get: () => 0, // The browser module has an empty environment.
            fd_write: (fd: number, vectors: Pointer, count: number, written: Pointer) => {
                if (fd !== 1 && fd !== 2) {
                    return 8; // EBADF: no files are opened by this module.
                }
                let total = 0;
                const decoder = new TextDecoder();
                for (let i = 0; i < offset(count); ++i) {
                    const item = offset(vectors) + i * 8;
                    const start = view().getUint32(item, true);
                    const size = view().getUint32(item + 4, true);
                    const text = decoder.decode(new Uint8Array(memory().buffer, start, size));
                    console.info('[Perforator WASM]', text);
                    total += size;
                }
                view().setUint32(offset(written), total, true);
                return 0;
            },
            fd_close: () => 8, // EBADF
            fd_seek: () => 70, // ESPIPE: stdout/stderr are not seekable.
            proc_exit: (code: number) => { throw new Error(`The native profile engine exited with code ${code}.`); },
        },
    };
}

export class PprofRuntime {
    static async create(url: string): Promise<PprofRuntime> {
        if (!supportsPprofWasm()) {
            throw new Error(unsupportedPprofWasm);
        }
        const response = await fetch(url);
        if (!response.ok) {
            throw new Error(`Could not load the profile engine: HTTP ${response.status}.`);
        }
        return PprofRuntime.createFromBytes(await response.arrayBuffer());
    }

    static async createFromBytes(bytes: ArrayBuffer): Promise<PprofRuntime> {
        if (!supportsPprofWasm()) {
            throw new Error(unsupportedPprofWasm);
        }
        let module: WebAssembly.Module;
        try {
            module = await WebAssembly.compile(bytes);
        } catch (error) {
            throw new Error(`Could not compile the profile engine: ${String(error)}`);
        }
        const imports = runtimeImports(() => native.memory);
        for (const item of WebAssembly.Module.imports(module)) {
            if (item.kind !== 'function' || typeof imports[item.module]?.[item.name] !== 'function') {
                throw new Error(`Unsupported profile engine import: ${item.module}.${item.name}.`);
            }
        }
        const instance = await WebAssembly.instantiate(module, imports);
        const native = instance.exports as unknown as NativeExports;
        native.__wasm_call_ctors();
        return new PprofRuntime(native);
    }

    private profile: Pointer = 0;
    private readonly native: NativeExports;

    private constructor(native: NativeExports) {
        this.native = native;
    }

    load(source: ArrayBuffer): void {
        if (this.profile !== 0) {
            throw new Error('A profile is already loaded.');
        }
        const input = this.allocate(Math.max(1, source.byteLength));
        let result = 0;
        try {
            result = this.allocate(4);
            new Uint8Array(this.native.memory.buffer, offset(input), source.byteLength).set(new Uint8Array(source));
            this.check(this.native.PerforatorProfileParsePProf(input, source.byteLength, result));
            this.profile = new DataView(this.native.memory.buffer).getUint32(offset(result), true);
        } finally {
            this.native.free(result);
            this.native.free(input);
        }
    }

    async render(options: PprofRenderOptions): Promise<PprofRenderResult> {
        validatePprofOptions(options);
        if (this.profile === 0) {
            throw new Error('No profile is loaded.');
        }
        const result = this.allocate(4);
        let json = 0;
        try {
            const started = performance.now();
            this.check(this.native.PerforatorWasmRender(this.profile, options.maxDepth, options.minWeight,
                Number(options.showFileNames), Number(options.showLineNumbers), result));
            json = new DataView(this.native.memory.buffer).getUint32(offset(result), true);
            const renderMs = performance.now() - started;
            const start = offset(this.native.PerforatorStringData(json));
            const jsonBytes = offset(this.native.PerforatorStringSize(json));
            const memory = this.native.memory;
            // The native output stays alive, and no other native call runs until
            // parsing finishes. Chunks avoid materializing a giant JS string.
            function* chunks() {
                for (let position = 0; position < jsonBytes; position += 64 * 1024) {
                    yield new Uint8Array(memory.buffer, start + position, Math.min(64 * 1024, jsonBytes - position));
                }
            }
            const parseStarted = performance.now();
            const profile: ProfileData = await parseChunked(chunks());
            profile.rows = profile.rows.filter(Boolean);
            return {
                profile,
                metrics: { profileParseMs: 0, renderMs, jsonParseMs: performance.now() - parseStarted,
                    jsonBytes, wasmMemoryBytes: memory.buffer.byteLength },
            };
        } finally {
            if (json !== 0) {
                this.native.PerforatorStringDispose(json);
            }
            this.native.free(result);
        }
    }

    dispose(): void {
        if (this.profile !== 0) {
            this.native.PerforatorProfileDispose(this.profile);
            this.profile = 0;
        }
    }

    private allocate(size: number): Pointer {
        if (size > 0xffffffff) {
            throw new Error('The profile exceeds the WASM32 address space.');
        }
        const pointer = this.native.malloc(size);
        if (pointer === 0) {
            throw new Error('The profile engine could not allocate memory.');
        }
        return pointer;
    }

    private check(error: Pointer): void {
        if (error === 0) {
            return;
        }
        try {
            const start = offset(this.native.PerforatorErrorString(error));
            const bytes = new Uint8Array(this.native.memory.buffer);
            const end = bytes.indexOf(0, start);
            throw new Error(new TextDecoder().decode(bytes.subarray(start, end < 0 ? undefined : end)));
        } finally {
            this.native.PerforatorErrorDispose(error);
        }
    }

}
