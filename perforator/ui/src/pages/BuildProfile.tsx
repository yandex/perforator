import React from 'react';

import { AxiosError } from 'axios';
import { useNavigate, useSearchParams } from 'react-router-dom';

import { Loader } from '@gravity-ui/uikit';

import { ErrorPanel } from 'src/components/ErrorPanel/ErrorPanel';
import type { ProfileTaskQuery } from 'src/models/Task';
import {
    defaultProfileTaskQuery,
    startProfileTask,
} from 'src/utils/profileTask';


const setupQuery = (searchParams: URLSearchParams): ProfileTaskQuery => {
    const query = defaultProfileTaskQuery();
    searchParams.forEach((value, key) => {
        (query as any)[key] = value ?? query[key as keyof ProfileTaskQuery];
    });
    return query;
};

export interface BuildProfileProps {}

export const BuildProfile: React.FC<BuildProfileProps> = () => {
    const isMounted = React.useRef(false);
    const [error, setError] = React.useState<string | undefined>(undefined);

    const [searchParams] = useSearchParams();
    const navigate = useNavigate();

    const navigateToTask = React.useCallback(async () => {
        const query = setupQuery(searchParams);
        try {
            const taskId = await startProfileTask(query);
            navigate(`/task/${taskId}`, { replace: true });
        } catch (e) {
            if (e instanceof AxiosError) {
                setError(e.message);
            } else {
                setError((e as any)?.message ?? 'Unknown error');
            }
        }
    }, [navigate, searchParams]);

    React.useEffect(() => {
        if (!isMounted.current) {
            navigateToTask();
            isMounted.current = true;
        }
    }, []);

    return error ? <ErrorPanel message={error} /> : <Loader />;
};
