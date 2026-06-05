import React from 'react';

import { ThemeProvider as GravityThemeProvider } from '@gravity-ui/uikit';

import { useUserSettings } from '../UserSettingsProvider';


export interface ThemeProviderProps {
    children?: React.ReactNode;
}

export const ThemeProvider: React.FC<ThemeProviderProps> = ({ children }: ThemeProviderProps) => {
    const { userSettings } = useUserSettings();
    return (
        <GravityThemeProvider theme={userSettings.theme}>
            {children}
        </GravityThemeProvider>
    );
};
