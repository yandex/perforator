import React from 'react';

import { useSearchParams } from 'react-router-dom';

import type { Coordinate } from '@perforator/flamegraph';

/** adds or modifies when value is truthy, deletes query if falsy.
 * Record key is query key, record value is new query value */
export function modifyQuery<T extends string = string>(query: URLSearchParams, q: Partial<Record<T, string | false>>) {
    for (const [field, value] of Object.entries<string | false>(q as Record<T, string | false>)) {

        if (value) {
            query.set(field, value);
        } else {
            query.delete(field);
        }
    }

    return query;
}

export type SetStateFromQuery<T extends string = string> = (q: Partial<Record<T, string | false>>) => void;
export type GetStateFromQuery<T extends string = string> = (name: T, defaultValue?: string) => string | undefined;

export const getStateFromQueryParams: <T extends string = string>(params: URLSearchParams) => GetStateFromQuery<T> = (params) => (name, defaultValue) => {
    if (params.has(name)) {
        return decodeURIComponent(params.get(name)!);
    } else {
        return defaultValue;
    }
};

export function useTypedQuery<T extends string>(): [GetStateFromQuery<T>, SetStateFromQuery<T>] {
    const [searchParams, setSearchParams] = useSearchParams();

    const getStateFromQuery: GetStateFromQuery<T> = React.useMemo(() => getStateFromQueryParams<T>(searchParams), [searchParams]);

    const updateStateInQuery = React.useCallback((q: Partial<Record<T, string | false>>) => {
        setSearchParams(newQuery => modifyQuery(newQuery, q));
    }, [setSearchParams]);

    return [
        getStateFromQuery,
        updateStateInQuery,
    ] as const;
}

export function stringifyStacks(stacks: Coordinate[]) {
    const res = [];
    for (const stack of stacks) {
        res.push(`${stack[0]},${stack[1]}`);
    }

    return res.join(';');
}

export function parseStacks(str: string) {

    if (str === '') {
        return [];
    }
    return str.split(';').map(p => {
        const [level, index] = p.split(',');
        return ([Number(level), Number(index)] as Coordinate);
    });
}

