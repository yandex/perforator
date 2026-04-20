import type { QueryKeys } from '@perforator/flamegraph';


const WELL_KNOWN_QUERY_PARAMS: string[] = [
    'flamegraphQuery',
    'exactMatch',
    'caseInsensitive',
    'flamegraphReverse',
    'tab',
    'flamegraphExclude',
    'leftHeavy',
    'keepOnlyFound',
    'topQuery',
    'flameBase',
] satisfies QueryKeys[];

export function preserveWellKnownQueryParams(searchParams: URLSearchParams): URLSearchParams {
    const preserved = new URLSearchParams();
    searchParams.forEach((value, key) => {
        if (WELL_KNOWN_QUERY_PARAMS.includes(key)) {
            preserved.set(key, value);
        }
    });
    return preserved;
}
