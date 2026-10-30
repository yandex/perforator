export function supportsPprofWasm(): boolean {
    // (module (func (try (do) (catch_all)))): native WASM exception handling.
    return typeof WebAssembly !== 'undefined' && WebAssembly.validate(new Uint8Array([
        0, 97, 115, 109, 1, 0, 0, 0,
        1, 4, 1, 96, 0, 0, 3, 2, 1, 0,
        10, 8, 1, 6, 0, 6, 64, 25, 11, 11,
    ]));
}

export const unsupportedPprofWasm = 'Local rendering requires a browser with WebAssembly exception handling support.';
