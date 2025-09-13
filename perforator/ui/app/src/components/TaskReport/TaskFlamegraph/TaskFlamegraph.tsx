import React from 'react';

import { parseFromWebStream } from '@discoveryjs/json-ext';
import type { QueryKeys } from '@perforator/flamegraph';
import { prerenderColors as prerenderColorsOriginal } from '@perforator/flamegraph';

import { useThemeType } from '@gravity-ui/uikit';

import { ErrorPanel } from 'src/components/ErrorPanel/ErrorPanel';
import { uiFactory } from 'src/factory';
import type { ProfileData } from 'src/models/Profile';
import { useUserSettings } from 'src/providers/UserSettingsProvider/UserSettingsContext.ts';
import { withMeasureTime } from 'src/utils/logging';
import { useTypedQuery } from 'src/utils/query';

import { Visualisation } from '../Visualisation/Visualisation';

import { useFetchResult } from './useFetchResult';


export type SupportedRenderFormats = 'Flamegraph' | 'JSONFlamegraph'

export interface TaskFlamegraphProps {
    url: string;
    isDiff: boolean;
    format?: SupportedRenderFormats;
}


export type Tab = 'flame' | 'top' | 'sbs'

export const TaskFlamegraph: React.FC<TaskFlamegraphProps> = (props) => {
    const isMounted = React.useRef(false);
    const theme = useThemeType();
    const { userSettings } = useUserSettings();

    const [getQuery] = useTypedQuery<QueryKeys>();
    const tab = getQuery('tab') ?? 'flame' as Tab;
    const pageName = tab === 'flame' ? 'task-flamegraph' : 'top-table';

    const { data: profileData, error } = useFetchResult<ProfileData>({ url: props.url, extractData: async(req) => {
        if (props.format === 'JSONFlamegraph') {
            const data = await parseFromWebStream(req.body!);
            return ({ rows: data.rows.filter(Boolean), stringTable: data.stringTable, meta: data.meta });
        } else if (props.format === 'Flamegraph') {
            const data = await req.text();
            return (uiFactory()?.parseLegacyFormat?.(data)!);
        } else {
            return { rows: [], stringTable: [], meta: {} };
        }
    },
    onFinishDataLoading: () => uiFactory().rum()?.finishDataLoading?.(pageName),
    onStartRequest: () => {
        if (!isMounted.current) {
            uiFactory().rum()?.makeSpaSubPage?.(pageName, undefined, undefined, { flamegraphFormat: props.format });
            isMounted.current = true;
        }
    },
    });

    const prerenderedNewData = React.useMemo(() => {
        if (profileData) {
            uiFactory().rum()?.startDataRendering?.(pageName, '', false);
            const framesCount = profileData?.rows?.reduce((acc, row) => acc + row.length, 0);

            const prerenderColors = withMeasureTime(prerenderColorsOriginal, 'prerenderColors', (ms) => uiFactory().rum()?.sendDelta?.('prerenderColors', ms, { additional: { framesCount } }));

            return prerenderColors(profileData, { theme });
        }
        return null;
    }, [profileData, theme]);


    const loading = !prerenderedNewData;


    if (error) {
        return <ErrorPanel message={error.message}/>;
    }

    return <Visualisation loading={loading} isDiff={props.isDiff} theme={theme} userSettings={userSettings} profileData={prerenderedNewData} />;

};
