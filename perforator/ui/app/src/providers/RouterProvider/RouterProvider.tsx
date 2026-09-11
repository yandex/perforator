import React from 'react';

import { RouterProvider as BaseRouterProvider } from 'react-router-dom';

import type { PagePublicProps } from 'src/components/Page/Page';
import { buildProfileRum } from 'src/utils/buildProfileRum';

import { getRouter } from './router';


export interface RouterProviderProps {
    pageProps: PagePublicProps;
}

export const RouterProvider: React.FC<RouterProviderProps> = ({ pageProps }: RouterProviderProps) => {
    const { embed } = pageProps;
    const router = React.useMemo(() => getRouter({ embed }), [embed]);

    // Start tracking before BuildProfile's passive effect creates the task.
    React.useLayoutEffect(() => {
        buildProfileRum.navigate(router.state.location);
        return router.subscribe(state => buildProfileRum.navigate(state.location));
    }, [router]);

    return <BaseRouterProvider router={router} />;
};
