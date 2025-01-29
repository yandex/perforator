import type { Theme } from '@gravity-ui/uikit';

import { LocalStorageKey } from 'src/const/localStorage';


const THEME_PARAM = '_theme';

export type ShortenMode = 'true' | 'false' | 'hover';

export interface UserSettings {
    monospace: 'default' | 'system';
    theme: Theme;
    shortenFrameTexts: ShortenMode;
}

const getUserSettingsFromLocalStorage = (): any => {
    try {
        return JSON.parse(localStorage.getItem(LocalStorageKey.UserSettings) || '{}');
    } catch (err: any) {
        console.error('Failed to get user settings from local storage:', err);
        return {};
    }
};

export const initialUserSettings = (): UserSettings => {
    const searchParams = new URLSearchParams(window.location.search);
    const userSettings = getUserSettingsFromLocalStorage();
    const theme = (
        searchParams.get(THEME_PARAM)
        || userSettings['theme']
        || 'system'
    );
    return {
        shortenFrameTexts: 'false',
        monospace: 'default',
        ...userSettings,
        theme,
    };
};
