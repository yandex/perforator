import React from 'react';

import { RouterProvider as BaseRouterProvider } from 'react-router-dom';

import type { PagePublicProps } from 'src/components/Page/Page';

import { getRouter } from './router';


export interface RouterProviderProps {
    pageProps: PagePublicProps;
}

export const RouterProvider: React.FC<RouterProviderProps> = ({ pageProps }: RouterProviderProps) => {
    return <BaseRouterProvider router={getRouter(pageProps)} />;
};
