import React from 'react';

import { ThemeProvider as GravityThemeProvider } from '@gravity-ui/uikit';

import { useUserSettings } from '../UserSettingsProvider';


export interface ThemeProviderProps {
    children?: React.ReactNode;
}

export const ThemeProvider: React.FC<ThemeProviderProps> = props => {
    const { userSettings } = useUserSettings();
    return (
        <GravityThemeProvider theme={userSettings.theme}>
            {props.children}
        </GravityThemeProvider>
    );
};
