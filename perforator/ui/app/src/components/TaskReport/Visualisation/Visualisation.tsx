import React from 'react';

import { useNavigate } from 'react-router-dom';

import type { Coordinate, FlamegraphProps, QueryKeys, TopTableProps } from '@perforator/flamegraph';
import { calculateTopForTable as calculateTopForTableOriginal, createLeftHeavy as createLeftHeavyOriginal, Flamegraph, inverseLeftHeavy as inverseLeftHeavyOriginal, SideBySide, TopTable } from '@perforator/flamegraph';

import { Loader } from '@gravity-ui/uikit';
import { Tabs } from '@gravity-ui/uikit/legacy';

import { Beta } from 'src/components/Beta/Beta';
import { useFullscreen } from 'src/components/Fullscreen/FullscreenContext';
import { uiFactory } from 'src/factory';
import { withMeasureTime } from 'src/utils/logging';
import { measureBrowserMemory } from 'src/utils/performance';
import { parseStacks, stringifyStacks, useTypedQuery } from 'src/utils/query';
import { createSuccessToast } from 'src/utils/toaster';

import type { Tab } from '../TaskFlamegraph/TaskFlamegraph';

import './Visualisation.css';


const calculateTopForTable = withMeasureTime(calculateTopForTableOriginal, 'calculateTopForTable', (ms) => uiFactory().rum()?.sendDelta?.('calculateTopForTable', ms));
const createLeftHeavy = withMeasureTime(createLeftHeavyOriginal, 'createLeftHeavy', (ms) => uiFactory().rum()?.sendDelta?.('createLeftHeavy', ms));
const inverseLeftHeavy = withMeasureTime(inverseLeftHeavyOriginal, 'inverseLeftHeavy', (ms) => uiFactory().rum()?.sendDelta?.('inverseLeftHeavy', ms));


type FlamegraphSizes = 'xs' | 's' | 'm' | 'l' | 'xl' | 'xxl';

const enum SizeThresholds {
    XS = 10_000,
    S = 50_000,
    M = 100_000,
    L = 500_000,
    XL = 1_000_000,
}

function getFlamegraphSize(n: number): FlamegraphSizes {
    if (n < SizeThresholds.XS) {
        return 'xs';
    }
    else if (n < SizeThresholds.S) {
        return 's';
    }
    else if (n < SizeThresholds.M) {
        return 'm';
    }
    else if (n < SizeThresholds.L) {
        return 'l';
    }
    else if (n < SizeThresholds.XL) {
        return 'xl';
    }

    return 'xxl';
}

export interface VisualisationProps extends Pick<FlamegraphProps,
'profileData'
 | 'isDiff'
 | 'theme'
 | 'userSettings'
 > {
    loading: boolean;
}

const RUM_BASE_NAME = 'flamegraph-render';

export const Visualisation: React.FC<VisualisationProps> = ({ profileData, ...props }) => {
    const navigate = useNavigate();
    const [getQuery, setQuery] = useTypedQuery<'tab' | QueryKeys>();
    const tab: Tab = getQuery('tab', 'flame') as Tab;
    const isTopTab = tab === 'top' || tab === 'sbs';
    const [isFirstTopRender, setIsFirstTopRender] = React.useState(isTopTab);
    React.useEffect(() => {
        setIsFirstTopRender(isFirstTopRender || isTopTab);
    }, [isFirstTopRender, isTopTab]);
    const { setEnabled } = useFullscreen();
    const isLeftHeavy = getQuery('leftHeavy', 'false') === 'true';
    const setIsLeftHeavy = React.useCallback((value: boolean) => {
        setQuery({ 'leftHeavy': value ? 'true' : 'false' });
    }, [setQuery]);


    const rowsRef = React.useRef(profileData?.rows);
    // HACK using memo for ref modification
    // otherwise would need useLayoutEffect + force the rerender
    React.useMemo(() => {
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

    React.useEffect(() => {
        if (tab === 'sbs') {
            setEnabled(true);
        }
        else {
            setEnabled(false);
        }
    }, []);

    const topData = React.useMemo(() => {
        return profileData && isFirstTopRender && rowsRef.current ? calculateTopForTable(rowsRef.current, profileData.stringTable.length, { rootCoords: [0, 0], omitted: [] }) : null;
    }, [profileData, isFirstTopRender]);

    const totalFrames = profileData?.rows.reduce((a, row) => a + row.length, 0);

    React.useEffect(() => {
        if (totalFrames !== undefined) {
            uiFactory().rum()?.logInt?.(`${RUM_BASE_NAME}-total-frames`, totalFrames);
        }
    }, [totalFrames]);

    let content: React.JSX.Element | undefined;

    if (props.loading) {
        content = <Loader />;
    } else {
        const flamegraphProps: FlamegraphProps = {
            profileData: profileData ? { rows: rowsRef.current!, meta: profileData?.meta, stringTable: profileData?.stringTable } : null,
            getState: getQuery,
            setState: setQuery,
            onFinishRendering: (opts) => {
                const size = getFlamegraphSize(totalFrames ?? 0);
                uiFactory().rum()?.finishDataRendering?.('task-flamegraph');
                const memory = measureBrowserMemory();
                function sendWithMetric(metricId: string) {
                    if (opts?.delta && opts?.textNodesCount) {
                        const additional = { textNodesCount: opts.textNodesCount, exceededLimit: opts.exceededLimit,
                            ...(memory ? memory : {}),
                        };
                        uiFactory().rum()?.sendDelta?.(metricId, opts.delta, { additional });
                        uiFactory().rum()?.logInt?.(`${metricId}-nodes`, opts.textNodesCount);
                        if (memory) {
                            uiFactory().rum()?.logMemory?.(metricId, memory);
                        }
                    }
                }
                sendWithMetric(RUM_BASE_NAME);
                sendWithMetric(RUM_BASE_NAME + '-' + size);
            },
            onSuccess: createSuccessToast,
            goToDefinitionHref: uiFactory().goToDefinitionHref,
            isLeftHeavy,
            onChangeLeftHeavy: setIsLeftHeavy,
            ...props,
        };
        const topTableProps: TopTableProps | null = topData && profileData ? {
            topData,
            profileData,
            navigate,
            getState: getQuery,
            setState: setQuery,
            onFinishRendering: () => {
                uiFactory().rum()?.finishDataRendering?.('top-table');
                const memory = measureBrowserMemory();
                if (memory) {
                    uiFactory().rum()?.logMemory?.('top-table', memory);
                }
            },
            goToDefinitionHref: uiFactory().goToDefinitionHref,
            ...props,
        } : null;

        if (tab === 'flame' ) {
            content = <Flamegraph
                {...flamegraphProps}
            />;
        }
        if (tab === 'top' && topTableProps) {
            content = <TopTable
                {...topTableProps}
            />;
        }
        if ( tab === 'sbs' && topTableProps) {
            content = <SideBySide
                {...flamegraphProps}
                navigate={navigate}
            />;
        }
    }

    return <React.Fragment>
        <Tabs
            className={'visualisation_tabs'}
            activeTab={tab}
            items={[
                { id: 'flame', title: 'Flamegraph' },
                { id: 'top', title: 'Top' },
                { id: 'sbs', title: <>Side by side <Beta/></> },
            ]}
            onSelectTab={(newTab: Tab) => {
                setQuery({ tab: newTab });
                if (newTab === 'sbs') {
                    setEnabled(true);
                }
                else {
                    setEnabled(false);
                }
            }}
        />
        {content}
    </React.Fragment>;
};
