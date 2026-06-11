import type { ProfileData } from './models/Profile';
import type { Coordinate, QueryKeys } from './renderer';

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

export function isValidFrameCoordinate(rows: ProfileData['rows'], [h, i]: Coordinate) {
    return Number.isInteger(h) && Number.isInteger(i) && h >= 0 && i >= 0 && Boolean(rows[h]?.[i]);
}

export function getFrameCoordinateFromQuery(getState: GetStateFromQuery<QueryKeys>): Coordinate {
    return [
        parseInt(getState('frameDepth', '0')!, 10),
        parseInt(getState('framePos', '0')!, 10),
    ];
}

export function normalizeFrameCoordinate(rows: ProfileData['rows'], coordinate: Coordinate): Coordinate {
    return isValidFrameCoordinate(rows, coordinate) ? coordinate : [0, 0];
}

export function resetInvalidFrameCoordinate(rows: ProfileData['rows'], getState: GetStateFromQuery<QueryKeys>, setState: SetStateFromQuery<QueryKeys>) {
    const coordinate = getFrameCoordinateFromQuery(getState);
    if (isValidFrameCoordinate(rows, coordinate)) {
        return coordinate;
    }

    if (getState('frameDepth') !== undefined || getState('framePos') !== undefined) {
        setState({
            frameDepth: false,
            framePos: false,
        });
    }

    return [0, 0] as Coordinate;
}
