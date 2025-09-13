import React from 'react';

import { AxiosError } from 'axios';


type UseFetchArgs<D> = {
    url: string;
    extractData: (res: Response) => Promise<D>;
    onFinishDataLoading?: () => void;
    onStartRequest?: () => void;
}
export function useFetchResult<D>(args: UseFetchArgs<D>) {
    const [data, setData] = React.useState<D | undefined>();
    const [error, setError] = React.useState<Error | undefined>();

    const getData = async ({ signal }: {signal: AbortSignal}) => {
        const fetchingStart = performance.now();
        const res = await fetch(args.url, { signal });
        const fetchingFinish = performance.now();

        // eslint-disable-next-line no-console
        console.log('Fetched data in', fetchingFinish - fetchingStart, 'ms');
        const extracted = await args.extractData(res);
        setData(extracted);
        args?.onFinishDataLoading?.();
    };

    const getDataWithCatch = async ({ signal }: {signal: AbortSignal}) => {
        try {
            await getData({ signal });
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
    }, [args.url]);

    return { data, error, loading, fetch: getDataWithCatch };
}
