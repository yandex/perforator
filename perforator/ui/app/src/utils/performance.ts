export interface Memory {
    totalJSHeapSize: number;
    usedJSHeapSize: number;
    jsHeapSizeLimit: number;
}

export function measureBrowserMemory(): Memory | null {
    const memory = performance?.memory;
    return memory ?
        { totalJSHeapSize: memory.totalJSHeapSize, usedJSHeapSize: memory.usedJSHeapSize, jsHeapSizeLimit: memory.jsHeapSizeLimit }
        : null;
}
