import React from 'react';

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

import { ToasterComponent, ToasterProvider, useThemeType } from '@gravity-ui/uikit';
import { toaster } from '@gravity-ui/uikit/toaster-singleton';

import { EMBED_PARAM } from 'src/const/query';
import { uiFactory } from 'src/factory';
import { RouterProvider } from 'src/providers/RouterProvider/RouterProvider';
import { ThemeProvider } from 'src/providers/ThemeProvider/ThemeProvider';
import { UserSettingsProvider } from 'src/providers/UserSettingsProvider/UserSettingsProvider';

import type { PagePublicProps } from '../Page/Page';

import './App.scss';


const queryClient = new QueryClient({
    defaultOptions: {
        queries: {
            retry: 1,
            refetchOnWindowFocus: false,
        },
    },
});

const AppImpl: React.FC<{}> = () => {
    const theme = useThemeType();
    const external = uiFactory().initializeExternal({ theme });
    const searchParams = new URLSearchParams(window.location.search);
    const embed = searchParams.get(EMBED_PARAM) === '1';
    const pageProps: PagePublicProps = {
        embed,
    };
    return <>
        <RouterProvider pageProps={pageProps} />
        {external}
    </>;
};

export const App: React.FC<{}> = () => {
    return (
        <QueryClientProvider client={queryClient}>
            <UserSettingsProvider>
                <ThemeProvider>
                    <ToasterProvider toaster={toaster}>
                        <AppImpl />
                        <ToasterComponent/>
                    </ToasterProvider>
                </ThemeProvider>
            </UserSettingsProvider>
        </QueryClientProvider>
    );
};
