import React from 'react';

import { useNavigate } from 'react-router-dom';

import type { FlamegraphProps, QueryKeys, TopTableProps } from '@perforator/flamegraph';
import {
    calculateTopForTable as calculateTopForTableOriginal,
    Flamegraph,
    SideBySide,
    TopTable,
    useLeftHeavyProfile,
} from '@perforator/flamegraph';

import { Loader } from '@gravity-ui/uikit';
import { Tabs } from '@gravity-ui/uikit/legacy';

import { Beta } from 'src/components/Beta/Beta';
import { ErrorBoundary } from 'src/components/ErrorBoundary/ErrorBoundary';
import { useFullscreen } from 'src/components/Fullscreen/FullscreenContext';
import { uiFactory } from 'src/factory';
import { boolToString } from 'src/utils/bool';
import { withMeasureTime } from 'src/utils/logging';
import { measureBrowserMemory } from 'src/utils/performance';
import { useTypedQuery } from 'src/utils/query';
import { createSuccessToast } from 'src/utils/toaster';

import type { Tab } from '../TaskFlamegraph/TaskFlamegraph';

import './Visualisation.css';


const calculateTopForTable = withMeasureTime(calculateTopForTableOriginal, 'calculateTopForTable', (ms) => uiFactory().rum()?.sendDelta?.('calculateTopForTable', ms));

const leftHeavyProfileOptions = {
    onCreateLeftHeavyMeasure: (ms: number) => uiFactory().rum()?.sendDelta?.('createLeftHeavy', ms),
    onInverseLeftHeavyMeasure: (ms: number) => uiFactory().rum()?.sendDelta?.('inverseLeftHeavy', ms),
};


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
 | 'disableHoverPopup'
 | 'onFrameClick'
 | 'onFrameAltClick'
 | 'onContextClick'
 | 'onContextItemClick'
 | 'onResetOmitted'
 | 'onSearch'
 | 'onKeepOnlyFound'
 | 'onSearchReset'
 | 'setOffsetterRef'
 | 'onChangeLeftHeavy'
 | 'showLineNumbers'
 | 'setShowLineNumbers'
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
        props.onChangeLeftHeavy?.(value);
        setQuery({ 'leftHeavy':  boolToString(value) });
    }, [setQuery, props.onChangeLeftHeavy]);

    React.useEffect(() => {
        if (!PerformanceObserver.supportedEntryTypes.includes('event')) {
            return () => {};
        }
        const observer = new PerformanceObserver((entries) => {
            const entry = entries.getEntriesByName('click')?.[0];
            if (!entry) {return;}
            // @ts-ignore
            if (entry?.target?.nodeName === 'CANVAS') {
                uiFactory?.().rum?.()?.sendDelta?.('flamegraph-rerender-full', entry.duration);
                uiFactory?.().rum?.()?.logInt?.('flamegraph-rerender-full-ev', entry.duration);
            }
        });
        // no durationThreshold in types yet
        //@ts-expect-error
        observer.observe({ type: 'event', durationThreshold: 16 });
        return () => {observer?.disconnect?.();};
    }, []);

    const newProfileData = useLeftHeavyProfile(profileData, isLeftHeavy, getQuery, setQuery, leftHeavyProfileOptions);
    React.useEffect(() => {
        if (tab === 'sbs') {
            setEnabled(true);
        }
    }, []);

    const topData = React.useMemo(() => {
        return newProfileData && isFirstTopRender && newProfileData.rows
            ? calculateTopForTable(
                newProfileData.rows,
                newProfileData.stringTable.length,
                { rootCoords: [0, 0], omitted: [], keepCoords: null },
            )
            : null;
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
            profileData: newProfileData,
            getState: getQuery,
            setState: setQuery,
            useSelfAsScrollParent: true,
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
            ...props,
            onChangeLeftHeavy: setIsLeftHeavy,
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
            content = <ErrorBoundary>
                <Flamegraph
                    {...flamegraphProps}
                />
            </ErrorBoundary>;
        }
        if (tab === 'top' && topTableProps) {
            content = <TopTable
                {...topTableProps}
            />;
        }
        if ( tab === 'sbs' && topTableProps) {
            content = <ErrorBoundary>
                <SideBySide
                    {...flamegraphProps}
                    navigate={navigate}
                />
            </ErrorBoundary>;
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
            }}
        />
        {content}
    </React.Fragment>;
};
