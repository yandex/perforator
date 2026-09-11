import React from 'react';

import { AxiosError } from 'axios';
import { useLocation, useNavigate, useSearchParams } from 'react-router-dom';

import { Loader } from '@gravity-ui/uikit';

import { ErrorPanel } from 'src/components/ErrorPanel/ErrorPanel';
import type { ProfileTaskQuery } from 'src/models/Task';
import { useUserSettings } from 'src/providers/UserSettingsProvider';
import { buildProfileRum } from 'src/utils/buildProfileRum';
import {
    defaultProfileTaskQuery,
    startProfileTask,
} from 'src/utils/profileTask';
import { preserveWellKnownQueryParams } from 'src/utils/profileTask/preserveWellKnown';


const setupQuery = (searchParams: URLSearchParams): ProfileTaskQuery => {
    const query = defaultProfileTaskQuery();
    searchParams.forEach((value, key) => {
        (query as any)[key] = value ?? query[key as keyof ProfileTaskQuery];
    });
    return query;
};

export interface BuildProfileProps {}

export const BuildProfile: React.FC<BuildProfileProps> = () => {
    const [error, setError] = React.useState<string | undefined>(undefined);
    const { userSettings: { pythonPrettifyLevel } } = useUserSettings();
    const [searchParams] = useSearchParams();
    const { key } = useLocation();
    const navigate = useNavigate();
    const request = React.useRef<{ key: string; result: Promise<string> } | undefined>(undefined);

    React.useEffect(() => {
        let active = true;
        const attempt = buildProfileRum.forBuild(key);
        // Reuse task creation during Strict Mode's effect replay, but ignore
        // completions after leaving /build or starting a different build.
        if (request.current?.key !== key) {
            setError(undefined);
            request.current = { key, result: startProfileTask(setupQuery(searchParams), { pythonPrettifyLevel }) };
        }
        request.current.result.then(taskId => {
            if (!active) {
                return;
            }
            if (attempt) {
                attempt.taskId = taskId;
            }
            const q = preserveWellKnownQueryParams(searchParams);
            navigate(`/task/${taskId}?${q.toString()}`, { replace: true });
        }).catch(e => {
            if (active) {
                attempt?.finish('error');
                setError(e instanceof AxiosError ? e.message : (e as Error)?.message ?? 'Unknown error');
            }
        });
        return () => { active = false; };
    }, [key, navigate, searchParams, pythonPrettifyLevel]);

    return error ? <ErrorPanel message={error} /> : <Loader />;
};
