import React, { useCallback } from 'react';

import { AxiosError } from 'axios';


type UseFetchArgs<D> = {
    url: string;
    extractData: (res: Response) => Promise<D>;
    onFinishDataLoading?: () => void;
    onStartRequest?: () => void;
}
export function useFetchResult<D>(args: UseFetchArgs<D>) {
    const getData = useCallback(async ({ signal }: {signal: AbortSignal}) => {
        args.onStartRequest?.();
        const fetchingStart = performance.now();
        const res = await fetch(args.url, { signal });
        const fetchingFinish = performance.now();

        // eslint-disable-next-line no-console
        console.log('Fetched data in', fetchingFinish - fetchingStart, 'ms');
        const extracted = await args.extractData(res);
        args?.onFinishDataLoading?.();
        return extracted;
    }, [args.url, args.extractData, args.onFinishDataLoading, args.onStartRequest]);

    return useAsyncResult<D>({ getData });
}

type UseAsyncArgs<D> = {
    getData: (args: { signal: AbortSignal }) => Promise<D>;
    clearPrevResult?: boolean;
}

export function useAsyncResult<D>({ getData, clearPrevResult }: UseAsyncArgs<D>) {
    const [data, setData] = React.useState<D | undefined>();
    const [error, setError] = React.useState<Error | undefined>();

    const getDataWithCatch = async ({ signal }: {signal: AbortSignal}) => {
        try {
            if (clearPrevResult) {
                setData(undefined);
            }
            const res = await getData({ signal });
            setData(res);
        } catch (e) {
            if (e instanceof AxiosError && e.code === 'ERR_CANCELED') {
                return;
            }
            if (e instanceof Error && e.name === 'AbortError') {
                return;
            }

            setError(e as Error);
        }
    };

    const loading = !data;

    React.useEffect(() => {
        const controller = new AbortController();
        getDataWithCatch({ signal: controller.signal });

        return () => controller.abort();
    }, [getData]);

    return { data, error, loading, fetch: getDataWithCatch };
}
