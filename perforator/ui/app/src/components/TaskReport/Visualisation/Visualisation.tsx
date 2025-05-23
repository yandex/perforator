import React from 'react';

import { useNavigate } from 'react-router-dom';

import type { FlamegraphProps, QueryKeys } from '@perforator/flamegraph';
import { calculateTopForTable, Flamegraph, TopTable } from '@perforator/flamegraph';

import { Loader, Tabs } from '@gravity-ui/uikit';

import { Link } from 'src/components/Link/Link';
import { uiFactory } from 'src/factory';
import { createSuccessToast } from 'src/utils/toaster';

import { useTypedQuery } from '../Flamegraph/query-utils';
import type { Tab } from '../TaskFlamegraph/TaskFlamegraph';

import './Visualisation.css';


export interface VisualisationProps extends Pick<FlamegraphProps, 'profileData' | 'isDiff' | 'theme' | 'userSettings'> {
    loading: boolean;
}

export const Visualisation: React.FC<VisualisationProps> = ({ profileData, ...props }) => {
    const navigate = useNavigate();
    const [getQuery, setQuery] = useTypedQuery<'tab' | QueryKeys>();
    const tab: Tab = getQuery('tab', 'flame') as Tab;
    const isTopTab = tab === 'top';
    const [isFirstTopRender, setIsFirstTopRender] = React.useState(isTopTab);
    React.useEffect(() => {
        setIsFirstTopRender(isFirstTopRender || isTopTab);
    }, [isFirstTopRender, isTopTab]);

    const topData = React.useMemo(() => {
        return profileData && isFirstTopRender ? calculateTopForTable(profileData.rows, profileData.stringTable.length) : null;
    }, [profileData, isFirstTopRender]);


    let content: React.JSX.Element | undefined;

    if (props.loading) {
        content = <Loader />;
    } else {
        if (tab === 'flame' ) {
            content = <Flamegraph
                profileData={profileData}
                getState={getQuery}
                setState={setQuery}
                onFinishRendering={() => uiFactory().rum()?.finishDataRendering?.('task-flamegraph')}
                onSuccess={createSuccessToast}
                goToDefinitionHref={uiFactory().goToDefinitionHref}
                {...props}
            />;
        }
        if (tab === 'top' && topData && profileData) {
            content = <TopTable
                onFinishRendering={() => uiFactory().rum()?.finishDataRendering?.('top-table')}
                goToDefinitionHref={uiFactory().goToDefinitionHref}
                topData={topData}
                profileData={profileData}
                navigate={navigate}
                getState={getQuery}
                setState={setQuery}
                {...props}
            />;
        }
    }

    return <React.Fragment>
        <Tabs
            className={'visualisation_tabs'}
            activeTab={tab}
            wrapTo={(item, node) => <Link key={item.id} href={`?tab=${item?.id}`}>{node}</Link>}
            items={[
                { id: 'flame', title: 'Flamegraph' },
                { id: 'top', title: 'Top' },
            ]}
            onSelectTab={() => {}}
        />
        {content}
    </React.Fragment>;
};
