import * as React from 'react';

import { Divider } from '@gravity-ui/uikit';

import type { DenselyPackedCoordinates } from '../../densely-packed';
import { useSearchPattern } from '../../hooks/use-search-pattern';
import { getFrameCoordinateFromQuery, normalizeFrameCoordinate, parseStacks, resetInvalidFrameCoordinate } from '../../query-utils';
import { search as outerSearch } from '../../search';
import { calculateTopForTable } from '../../top';
import { cn } from '../../utils/cn';
import { Flamegraph, type FlamegraphProps } from '../Flamegraph/Flamegraph';
import { TopTable, type TopTableProps } from '../TopTable/TopTable';

import './SideBySide.css';


export type SideBySideProps = FlamegraphProps & Pick<TopTableProps, 'navigate'>

const b = cn('visualisation_sbs');

export function SideBySide(props: SideBySideProps) {
    const { profileData, getState, setState } = props;
    const [frameDepth, framePos] = React.useMemo(
        () => normalizeFrameCoordinate(profileData.rows, getFrameCoordinateFromQuery(getState)),
        [getState, profileData.rows],
    );

    React.useEffect(() => {
        resetInvalidFrameCoordinate(profileData.rows, getState, setState);
    }, [getState, profileData.rows, setState]);
    const omitted = getState('omittedIndexes', '');
    const keepOnlyFound = getState('keepOnlyFound') === 'true';
    const search = getState('flamegraphQuery');
    const exactMatch = getState('exactMatch') === 'true';
    const excludeSearch = getState('flamegraphExclude');
    const caseInsensitive = getState('caseInsensitive') === 'true';

    const searchFn = React.useCallback((query: RegExp, omitQuery?: RegExp): DenselyPackedCoordinates => {
        if (!profileData.rows) {
            return [];
        }
        function readString(index: number) {
            return profileData?.stringTable[index] ?? '';
        }

        return outerSearch(readString, (str) => str, false, profileData.rows, query, omitQuery);
    }, [profileData.rows, profileData?.stringTable]);

    const { searchPattern, excludeSearchPattern } = useSearchPattern(search, excludeSearch, exactMatch, caseInsensitive);

    const topData = React.useMemo(() => {
        let keepCoords: DenselyPackedCoordinates | null = null;
        if (keepOnlyFound && search) {

            keepCoords = searchFn(searchPattern, excludeSearchPattern);
        }
        return profileData ? calculateTopForTable(profileData.rows, profileData.stringTable.length, { rootCoords: [frameDepth, framePos] as [number, number], omitted: parseStacks(omitted), keepCoords }) : null;
    }, [keepOnlyFound, search, excludeSearchPattern, searchPattern, profileData, frameDepth, framePos, omitted, searchFn]);
    return <div className={b()}>
        <TopTable
            lines={100}
            disableAutoTabSwitch
            className={b('top-table')}
            {...props}
            topData={topData!}
        />
        <Divider orientation={'vertical'} />
        <Flamegraph
            useSelfAsScrollParent
            className={b('flamegraph')}
            {...props}
        />
    </div>;
}
