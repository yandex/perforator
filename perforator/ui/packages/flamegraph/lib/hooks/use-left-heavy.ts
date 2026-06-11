import React from 'react';

import { createLeftHeavy, inverseLeftHeavy } from '../left-heavy';
import type { ProfileData } from '../models/Profile';
import type { GetStateFromQuery, SetStateFromQuery } from '../query-utils';
import { parseStacks, stringifyStacks } from '../query-utils';
import type { Coordinate, QueryKeys } from '../renderer';


export type UseLeftHeavyProfileOptions = {
    onCreateLeftHeavyMeasure?: (ms: number) => void;
    onInverseLeftHeavyMeasure?: (ms: number) => void;
};

function measure<T>(fn: () => T, onMeasure?: (ms: number) => void): T {
    const start = performance.now();
    const result = fn();
    onMeasure?.(performance.now() - start);
    return result;
}

export function useLeftHeavyProfile(
    profileData: ProfileData | null,
    leftHeavy: boolean,
    getState: GetStateFromQuery<QueryKeys>,
    setState: SetStateFromQuery<QueryKeys>,
    options: UseLeftHeavyProfileOptions = {},
): ProfileData | null {
    const rowsRef = React.useRef(profileData?.rows);
    const prevProfileRowsRef = React.useRef(profileData?.rows);
    const firstRenderRef = React.useRef(true);
    return React.useMemo(() => {
        const omitted = parseStacks(getState('omittedIndexes') ?? '');

        const currentRootH = parseInt(getState('frameDepth') ?? '0');
        const currentRootI = parseInt(getState('framePos') ?? '0');

        if (profileData?.rows !== prevProfileRowsRef.current) {
            rowsRef.current = profileData?.rows;
            prevProfileRowsRef.current = profileData?.rows;
            firstRenderRef.current = true;
        }


        const remappedOmitted: Coordinate[] = [];
        let remappedRoot: Coordinate | null = null;

        function findOmittedIndex(h: number, i: number) {
            for (let j = 0; j < omitted.length; j++) {
                if (omitted[j][0] === h && omitted[j][1] === i) {
                    return j;
                }
            }

            return -1;
        }

        const coordsMapper = (hmap: number, oldI: number, newI: number) => {
            if (firstRenderRef.current) {
                return;
            }

            if (hmap === currentRootH && oldI === currentRootI) {
                remappedRoot = [hmap, newI];
            }

            const idx = findOmittedIndex(hmap, oldI);

            if (idx !== -1) {
                remappedOmitted[idx] = [hmap, newI];
            }
        };

        if (profileData?.rows && leftHeavy) {
            const rows = measure(
                () => createLeftHeavy(rowsRef.current ?? profileData.rows, 'eventCount', coordsMapper),
                options.onCreateLeftHeavyMeasure,
            );
            rowsRef.current = rows;
        } else if (profileData?.rows && !leftHeavy) {
            const rows = measure(
                () => inverseLeftHeavy(rowsRef.current ?? profileData.rows, profileData.stringTable, coordsMapper),
                options.onInverseLeftHeavyMeasure,
            );
            rowsRef.current = rows;
        }

        if (firstRenderRef.current && profileData?.rows) {
            firstRenderRef.current = false;
        }

        const hasRemappedOmitted = remappedOmitted.length > 0;

        if (remappedRoot) {
            setState({
                framePos: String(remappedRoot[1]),
            });
        }

        if (hasRemappedOmitted) {
            setState({ omittedIndexes: stringifyStacks(remappedOmitted) });
        }

        return profileData ? ({
            rows: rowsRef.current,
            meta: profileData.meta,
            stringTable: profileData.stringTable,
        } as ProfileData) : null;
    }, [profileData, leftHeavy, options.onCreateLeftHeavyMeasure, options.onInverseLeftHeavyMeasure]);
}
