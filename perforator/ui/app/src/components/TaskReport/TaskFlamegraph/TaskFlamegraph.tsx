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

import { useTypedQuery } from '../Flamegraph/query-utils';
import { Visualisation } from '../Visualisation/Visualisation';


const prerenderColors = withMeasureTime(prerenderColorsOriginal);


export type SupportedRenderFormats = 'Flamegraph' | 'JSONFlamegraph'

export interface TaskFlamegraphProps {
    url: string;
    isDiff: boolean;
    format?: SupportedRenderFormats;
}


export type Tab = 'flame' | 'top'

export const TaskFlamegraph: React.FC<TaskFlamegraphProps> = (props) => {
    const isMounted = React.useRef(false);
    const theme = useThemeType();
    const { userSettings } = useUserSettings();


    const [profileData, setProfileData] = React.useState<ProfileData | undefined>();
    const [error, setError] = React.useState<Error | undefined>();
    const [getQuery] = useTypedQuery<QueryKeys>();
    const tab = getQuery('tab') ?? 'flame' as Tab;
    const pageName = tab === 'flame' ? 'task-flamegraph' : 'top-table';

    const getProfileData = async () => {

        const fetchingStart = performance.now();
        const req = await fetch(props.url);
        const fetchingFinish = performance.now();

        // eslint-disable-next-line no-console
        console.log('Fetched data in', fetchingFinish - fetchingStart, 'ms');
        if (props.format === 'JSONFlamegraph') {
            const data = await parseFromWebStream(req.body!);
            setProfileData({ rows: data.rows.filter(Boolean), stringTable: data.stringTable, meta: data.meta });
        } else if (props.format === 'Flamegraph') {
            const data = await req.text();
            setProfileData(uiFactory()?.parseLegacyFormat?.(data));
        }
        uiFactory().rum()?.finishDataLoading?.(pageName);
    };

    const getProfileDataWithCatch = async () => {
        try {
            await getProfileData();
        } catch (e) {
            setError(e as Error);
        }
    };

    const prerenderedNewData = React.useMemo(() => {
        if (profileData) {
            uiFactory().rum()?.startDataRendering?.(pageName, '', false);
            return prerenderColors(profileData, { theme });
        }
        return null;
    }, [profileData, theme]);


    const loading = !prerenderedNewData;

    React.useEffect(() => {
        if (!isMounted.current) {
            uiFactory().rum()?.makeSpaSubPage?.(pageName, undefined, undefined, { flamegraphFormat: props.format });
            isMounted.current = true;
            getProfileDataWithCatch();

        }
    });

    if (error) {
        return <ErrorPanel message={error.message}/>;
    }

    return <Visualisation loading={loading} isDiff={props.isDiff} theme={theme} userSettings={userSettings} profileData={prerenderedNewData} />;

};
