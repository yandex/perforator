import React, { useCallback, useMemo } from 'react';

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
    onLineNumbersChange: (value: boolean) => void;
    lineNumbers: boolean;
}


export type Tab = 'flame' | 'top' | 'sbs'

export const TaskFlamegraph: React.FC<TaskFlamegraphProps> = ({ url, isDiff, format, lineNumbers, onLineNumbersChange }: TaskFlamegraphProps) => {
    const isMounted = React.useRef(false);
    const theme = useThemeType();
    const { userSettings } = useUserSettings();

    const [getQuery] = useTypedQuery<QueryKeys>();
    const tab = getQuery('tab') ?? 'flame' as Tab;
    const pageName = tab === 'flame' ? 'task-flamegraph' : 'top-table';

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
    }, [pageName, url]);

    const onStartRequest = useCallback(() => {
        if (!isMounted.current) {
            uiFactory().rum()?.makeSpaSubPage?.(pageName, undefined, undefined, { flamegraphFormat: format });
            isMounted.current = true;
        }
    }, [format]);

    const { data: profileData, error } = useFetchResult<ProfileData>({ url: url, extractData: extractData,
        onFinishDataLoading: onFinishDataLoading,
        onStartRequest: onStartRequest,
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

    return (
        <Visualisation
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
