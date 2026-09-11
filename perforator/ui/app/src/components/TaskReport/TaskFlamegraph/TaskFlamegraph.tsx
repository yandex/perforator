import React, { useCallback, useMemo } from 'react';

import { useParams } from 'react-router-dom';

import { parseFromWebStream } from '@discoveryjs/json-ext';
import type { QueryKeys } from '@perforator/flamegraph';
import { prerenderColors as prerenderColorsOriginal } from '@perforator/flamegraph';

import { useThemeType } from '@gravity-ui/uikit';

import { ErrorPanel } from 'src/components/ErrorPanel/ErrorPanel';
import { uiFactory } from 'src/factory';
import type { ProfileData } from 'src/models/Profile';
import { useUserSettings } from 'src/providers/UserSettingsProvider/UserSettingsContext.ts';
import { buildProfileRum } from 'src/utils/buildProfileRum';
import { withMeasureTime } from 'src/utils/logging';
import { useTypedQuery } from 'src/utils/query';

import { Visualisation } from '../Visualisation/Visualisation';

import { useFetchResult } from './useFetchResult';


export type SupportedRenderFormats = 'Flamegraph' | 'JSONFlamegraph'

export interface TaskFlamegraphProps {
    url: string;
    isDiff: boolean;
    format?: SupportedRenderFormats;
    onLineNumbersChange: (value: boolean) => void;
    lineNumbers: boolean;
}


export type Tab = 'flame' | 'top' | 'sbs'

export const TaskFlamegraph: React.FC<TaskFlamegraphProps> = ({ url, isDiff, format, lineNumbers, onLineNumbersChange }: TaskFlamegraphProps) => {
    const { taskId } = useParams();
    const attempt = buildProfileRum.forTask(taskId);
    const theme = useThemeType();
    const { userSettings } = useUserSettings();

    const [getQuery] = useTypedQuery<QueryKeys>();
    const tab = getQuery('tab') ?? 'flame' as Tab;
    const pageName = tab === 'flame' ? 'task-flamegraph' : 'top-table';
    const renderingPage = React.useRef<string | undefined>(undefined);

    const extractData = useMemo(() => {
        return async (req: Response) => {
            if (format === 'JSONFlamegraph') {
                const data = await parseFromWebStream(req.body!);
                const rows = data.rows.filter(Boolean);
                return ({ rows, stringTable: data.stringTable, meta: data.meta });
            } else if (format === 'Flamegraph') {
                const data = await req.text();
                return (uiFactory()?.parseLegacyFormat?.(data)!);
            } else {
                return { rows: [], stringTable: [], meta: {} };
            }
        };
    }, [format]);

    const onFinishDataLoading = useCallback(() => {
        uiFactory().rum()?.finishDataLoading?.(pageName);
        uiFactory().rum()?.sendResTiming?.(url);
        uiFactory().rum()?.startDataRendering?.(pageName, '', false);
        renderingPage.current = pageName;
    }, [pageName, url]);

    const onStartRequest = useCallback(() => {
        renderingPage.current = undefined;
        uiFactory().rum()?.makeSpaSubPage?.(pageName, undefined, undefined, false, { flamegraphFormat: format });
    }, [pageName, format]);

    const { data: profileData, error } = useFetchResult<ProfileData>({ url: url, extractData: extractData,
        onFinishDataLoading: onFinishDataLoading,
        onStartRequest: onStartRequest,
    });

    React.useEffect(() => {
        if (error) {
            attempt?.finish('error');
        }
    }, [attempt, error]);

    const prerenderedNewData = React.useMemo(() => {
        if (profileData) {
            const framesCount = profileData?.rows?.reduce((acc, row) => acc + row.length, 0);

            const prerenderColors = withMeasureTime(prerenderColorsOriginal, 'prerenderColors', (ms) => uiFactory().rum()?.sendDelta?.('prerenderColors', ms, { additional: { framesCount } }));

            return prerenderColors(profileData, { theme });
        }
        return null;
    }, [profileData, theme]);


    const loading = !prerenderedNewData;

    const onFinishRendering = useCallback(() => {
        if (renderingPage.current) {
            uiFactory().rum()?.finishDataRendering?.(renderingPage.current);
            renderingPage.current = undefined;
            attempt?.finish();
        }
    }, [attempt]);


    if (error) {
        return <ErrorPanel message={error.message}/>;
    }

    return (
        <Visualisation
            onFinishRendering={onFinishRendering}
            loading={loading}
            isDiff={isDiff}
            theme={theme}
            userSettings={userSettings}
            profileData={prerenderedNewData}
            showLineNumbers={lineNumbers}
            setShowLineNumbers={onLineNumbersChange}
        />
    );


};
