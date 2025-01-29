import React from 'react';

import { RouterProvider as BaseRouterProvider } from 'react-router-dom';

import type { PageProps } from 'src/components/Page/Page';

import { getRouter } from './router';


export interface RouterProviderProps {
    pageProps: PageProps;
}

export const RouterProvider: React.FC<RouterProviderProps> = props => {
    return <BaseRouterProvider router={getRouter(props.pageProps)} />;
};
