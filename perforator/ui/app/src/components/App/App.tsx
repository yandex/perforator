import React from 'react';

import { ToasterComponent, ToasterProvider, useThemeType } from '@gravity-ui/uikit';
import { toaster } from '@gravity-ui/uikit/toaster-singleton';

import { uiFactory } from 'src/factory';
import { RouterProvider } from 'src/providers/RouterProvider/RouterProvider';
import { ThemeProvider } from 'src/providers/ThemeProvider/ThemeProvider';
import { UserSettingsProvider } from 'src/providers/UserSettingsProvider/UserSettingsProvider';

import type { PageProps } from '../Page/Page';

import './App.scss';


const AppImpl: React.FC<{}> = () => {
    const theme = useThemeType();
    const external = uiFactory().initializeExternal({ theme });
    const searchParams = new URLSearchParams(window.location.search);
    const embed = searchParams.get('embed') === '1';
    const pageProps: PageProps = {
        embed,
    };
    return <>
        <RouterProvider pageProps={pageProps} />
        {external}
    </>;
};

export const App: React.FC<{}> = () => {
    return (
        <UserSettingsProvider>
            <ThemeProvider>
                <ToasterProvider toaster={toaster}>
                    <AppImpl />
                    <ToasterComponent/>
                </ToasterProvider>
            </ThemeProvider>
        </UserSettingsProvider>
    );
};
