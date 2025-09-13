import React, { useMemo } from 'react';

import { useNavigate } from 'react-router-dom';

import type { FlamegraphProps, QueryKeys } from '@perforator/flamegraph';
import { calculateTopForTable, Flamegraph, prerenderColors, SideBySide, TopTable } from '@perforator/flamegraph';

import { Loader, useThemeType } from '@gravity-ui/uikit';
import { Tabs } from '@gravity-ui/uikit/legacy';
import { Link } from '../Link/Link';
import { createSuccessToast } from '../../utils/toaster';

import { useTypedQuery } from '../../query-utils';

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
    const isTopTab = tab === 'top';
    const [isFirstTopRender, setIsFirstTopRender] = React.useState(isTopTab);
    React.useEffect(() => {
        setIsFirstTopRender(isFirstTopRender || isTopTab);
    }, [isFirstTopRender, isTopTab]);
    const theme = useThemeType();

    React.useLayoutEffect(() => {
        if (profileData) {prerenderColors(profileData, { theme });}
    }, [profileData, theme]);

    const isDiff = useMemo(() => Boolean(profileData?.rows?.[0][0].baseEventCount), [profileData])

    const topData = React.useMemo(() => {
        return profileData && isFirstTopRender
            ? calculateTopForTable(profileData.rows, profileData.stringTable.length)
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
        profileData,
        getState: getQuery,
        isDiff,
        setState: setQuery,
        onSuccess: createSuccessToast,
        userSettings,
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
                wrapTo={(item, node) => (
                    <Link key={item.id} href={`?tab=${item?.id}`}>
                        {node}
                    </Link>
                )}
                items={[
                    { id: 'flame', title: 'Flamegraph' },
                    { id: 'top', title: 'Top' },
                    { id: 'sbs', title: 'Side by side' }
                ]}
                onSelectTab={() => {}}
            />
            {content}
        </div>
    );
};
