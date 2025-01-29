import React from 'react';

import axios from 'axios';

import { useThemeType } from '@gravity-ui/uikit';

import { uiFactory } from 'src/factory';
import type { ProfileData } from 'src/models/Profile';
import { useUserSettings } from 'src/providers/UserSettingsProvider/UserSettingsContext.ts';
import { withMeasureTime } from 'src/utils/logging';

import { Flamegraph } from '../Flamegraph/Flamegraph';
import { prerenderColors as prerenderColorsOriginal } from '../Flamegraph/utils/colors';


const prerenderColors = withMeasureTime(prerenderColorsOriginal);


export type SupportedRenderFormats = 'Flamegraph' | 'JSONFlamegraph'

export interface TaskFlamegraphProps {
    url: string;
    isDiff: boolean;
    format?: SupportedRenderFormats;
}

export const TaskFlamegraph: React.FC<TaskFlamegraphProps> = (props) => {
    const isMounted = React.useRef(false);
    const theme = useThemeType();
    const { userSettings } = useUserSettings();


    const [newData, setNewData] = React.useState<ProfileData | undefined>();

    const getProfileData = async () => {
        const fetchingStart = performance.now();
        const data = (
            await axios.get(props.url, {
                headers: {
                    'Accept-encoding': 'gzip',
                },
            })
        )?.data;
        const fetchingFinish = performance.now();

        // eslint-disable-next-line no-console
        console.log('Fetched data in', fetchingFinish - fetchingStart, 'ms');

        if (props.format === 'JSONFlamegraph') {
            setNewData(data);
        } else if (props.format === 'Flamegraph') {
            setNewData(uiFactory()?.parseLegacyFormat?.(data));
        }
        uiFactory().rum()?.finishDataLoading?.('task-flamegraph');
    };

    const prerenderedNewData = React.useMemo(() => {
        uiFactory().rum()?.startDataRendering?.('task-flamegraph', '', false);
        if (newData) {
            return prerenderColors(newData, { theme });
        }
        return null;
    }, [newData, theme]);

    const loading = !prerenderedNewData;

    React.useEffect(() => {
        if (!isMounted.current) {
            uiFactory().rum()?.makeSpaSubPage?.('task-flamegraph');
            isMounted.current = true;
            getProfileData();
        }
    });

    return (
        <Flamegraph
            isDiff={props.isDiff}
            theme={theme}
            userSettings={userSettings}
            newData={prerenderedNewData}
            loading={loading}
        />
    );
};
