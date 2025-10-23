import React, { useMemo } from 'react';

import { useNavigate } from 'react-router-dom';

import type { Coordinate, FlamegraphProps, QueryKeys } from '@perforator/flamegraph';
import { calculateTopForTable, Flamegraph, prerenderColors, SideBySide, TopTable, createLeftHeavy, inverseLeftHeavy } from '@perforator/flamegraph';

import { Loader, useThemeType } from '@gravity-ui/uikit';
import { Tabs } from '@gravity-ui/uikit/legacy';
import { createSuccessToast } from '../../utils/toaster';

import { parseStacks, stringifyStacks, useTypedQuery } from '../../query-utils';

import './Visualisation.css';
import { cn } from '../../utils/cn';

export type Tab = 'flame' | 'top' | 'sbs';
export interface VisualisationProps
    extends Pick<FlamegraphProps, 'profileData' | 'theme'> {
    loading: boolean;
}

const b = cn('vis')

export const Visualisation: React.FC<VisualisationProps> = ({ profileData, ...props }) => {
    const navigate = useNavigate();
    const [getQuery, setQuery] = useTypedQuery<'tab' | QueryKeys>();
    const tab: Tab = getQuery('tab', 'flame') as Tab;
    const isLeftHeavy = getQuery('leftHeavy', 'false') === 'true';
    const setIsLeftHeavy = React.useCallback((value: boolean) => {
        setQuery({ 'leftHeavy': value ? 'true' : 'false' });
    }, [setQuery]);
    const isTopTab = tab === 'top';
    const [isFirstTopRender, setIsFirstTopRender] = React.useState(isTopTab);
    React.useEffect(() => {
        setIsFirstTopRender(isFirstTopRender || isTopTab);
    }, [isFirstTopRender, isTopTab]);
    const theme = useThemeType();
    const rowsRef = React.useRef(profileData?.rows);

        // HACK using memo for ref modification
    // otherwise would need useLayoutEffect + force the rerender
    React.useMemo(() => {
        if (profileData) {prerenderColors(profileData, { theme });}
        const currentRootH = parseInt(getQuery('frameDepth') ?? '0');
        const currentRootI = parseInt(getQuery('framePos') ?? '0');
        const omittedIndices = parseStacks(getQuery('omittedIndexes') ?? '');
        const newOmittedIndices: Coordinate[] = [];
        function findSatisfiesOmittedIndex (h: number, i: number) {
            for (let j = 0; j < omittedIndices.length; j++) {
                if (omittedIndices[j][0] === h && omittedIndices[j][1] === i) {
                    return j;
                }
            }
            return -1;
        }
        const coordsMapper = (hmap: number, oldI: number, newI: number) => {
            if (hmap === currentRootH && oldI === currentRootI) {
                setQuery({ framePos: String(newI) });
            }
            const idx = findSatisfiesOmittedIndex(hmap, oldI);
            if (idx !== -1) {
                newOmittedIndices[idx] = [hmap, newI];
            }
        };
        if (profileData?.rows && isLeftHeavy) {
            const rows = createLeftHeavy(rowsRef.current ?? profileData.rows, 'eventCount', coordsMapper);
            rowsRef.current = rows;
            if (newOmittedIndices && newOmittedIndices.length > 0) {
                setQuery({ omittedIndexes: stringifyStacks(newOmittedIndices) });
            }
            return;
        } else if (profileData?.rows && !isLeftHeavy) {
            const rows = inverseLeftHeavy(rowsRef.current ?? profileData.rows, profileData.stringTable, coordsMapper);
            rowsRef.current = rows;
            if (newOmittedIndices && newOmittedIndices.length > 0) {
                setQuery({ omittedIndexes: stringifyStacks(newOmittedIndices) });
            }
            return;
        }
    }, [profileData?.rows, isLeftHeavy, props.loading]);

    const isDiff = useMemo(() => Boolean(profileData?.rows?.[0][0].baseEventCount), [profileData])

    const topData = React.useMemo(() => {
        return profileData && isFirstTopRender && rowsRef.current
            ? calculateTopForTable(
                  rowsRef.current,
                  profileData.stringTable.length,
                  { rootCoords: [0, 0], omitted: [] }
              )
            : null;
    }, [profileData, isFirstTopRender]);

    const userSettings  = {
        monospace: 'default',
        numTemplating: 'exponent',
        reverseFlameByDefault: true,
        shortenFrameTexts: 'false',
        theme: 'system'
    } as const;

    const flamegraphProps: FlamegraphProps = {
        goToDefinitionHref: () => '',
        profileData: profileData ? { rows: rowsRef.current!, meta: profileData?.meta, stringTable: profileData?.stringTable } : null,
        getState: getQuery,
        isDiff,
        setState: setQuery,
        onSuccess: createSuccessToast,
        userSettings,
        isLeftHeavy,
        onChangeLeftHeavy: setIsLeftHeavy,
        ...props
    };

    let content: React.JSX.Element | undefined;

    if (props.loading) {
        content = <Loader />;
    } else {
        if (tab === 'flame') {
            content = <Flamegraph {...flamegraphProps} />;
        }
        if (tab === 'top' && topData && profileData) {
            const topTableProps = {
                goToDefinitionHref: () => '',
                topData,
                profileData,
                userSettings,
                navigate,
                getState: getQuery,
                setState: setQuery,
                ...props
            };
            content = <TopTable {...topTableProps} />;
        }

        if(tab === 'sbs') {
            content = <SideBySide navigate={navigate} {...flamegraphProps} />;
        }
    }

    return (
        <div className={b({sbs: tab === 'sbs'})}>
            <Tabs
                className={'vis_tabs'}
                activeTab={tab}
                items={[
                    { id: 'flame', title: 'Flamegraph' },
                    { id: 'top', title: 'Top' },
                    { id: 'sbs', title: 'Side by side' }
                ]}
                onSelectTab={(newTab) => {
                    setQuery({ tab: newTab });
                }}
            />
            {content}
        </div>
    );
};
