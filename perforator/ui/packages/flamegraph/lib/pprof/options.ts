export interface PprofRenderOptions {
    maxDepth: number;
    minWeight: number;
    showFileNames: boolean;
    showLineNumbers: boolean;
}

export const defaultPprofOptions: Readonly<PprofRenderOptions> = {
    maxDepth: 256,
    minWeight: 1e-10,
    showFileNames: true,
    showLineNumbers: false,
};

export const pprofQueryKeys = ['maxDepth', 'minWeight', 'showFileNames', 'lineNumbers'] as const;

export function validatePprofOptions(options: PprofRenderOptions): void {
    if (!Number.isInteger(options.maxDepth) || options.maxDepth < 0 || options.maxDepth > 0xffffffff) {
        throw new Error('Maximum depth must be an unsigned 32-bit integer (0 means unlimited).');
    }
    if (!Number.isFinite(options.minWeight) || options.minWeight < 0 || options.minWeight > 1) {
        throw new Error('Minimum weight must be between 0 and 1.');
    }
}

export function readPprofOptions(query: URLSearchParams): PprofRenderOptions {
    const number = (key: 'maxDepth' | 'minWeight') => {
        const value = query.get(key);
        if (value === null) {
            return defaultPprofOptions[key];
        }
        if (!value.trim()) {
            throw new Error(`${key} must be a number.`);
        }
        return Number(value);
    };
    const boolean = (key: string, fallback: boolean) => {
        const value = query.get(key);
        if (value === null) {
            return fallback;
        }
        if (value !== 'true' && value !== 'false') {
            throw new Error(`${key} must be true or false.`);
        }
        return value === 'true';
    };
    const options = {
        maxDepth: number('maxDepth'),
        minWeight: number('minWeight'),
        showFileNames: boolean('showFileNames', defaultPprofOptions.showFileNames),
        showLineNumbers: boolean('lineNumbers', defaultPprofOptions.showLineNumbers),
    };
    validatePprofOptions(options);
    return options;
}

export function writePprofOptions(query: URLSearchParams, options: PprofRenderOptions): URLSearchParams {
    for (const [key, value] of [
        ['maxDepth', options.maxDepth], ['minWeight', options.minWeight],
        ['showFileNames', options.showFileNames], ['lineNumbers', options.showLineNumbers],
    ] as const) {
        const defaultValue = key === 'lineNumbers' ? defaultPprofOptions.showLineNumbers : defaultPprofOptions[key];
        if (value === defaultValue) {
            query.delete(key);
        } else {
            query.set(key, String(value));
        }
    }
    return query;
}

export function pprofOptionsKey(options: PprofRenderOptions): string {
    return [options.maxDepth, options.minWeight, options.showFileNames, options.showLineNumbers].join(':');
}
