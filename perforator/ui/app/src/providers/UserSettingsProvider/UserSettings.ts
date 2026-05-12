import type { Theme } from '@gravity-ui/uikit';

import { LocalStorageKey } from 'src/const/localStorage';
import { THEME_PARAM } from 'src/const/query';


export type ShortenMode = 'true' | 'false' | 'hover';

export type NumTemplatingFormat = 'exponent' | 'hugenum';

export type PythonPrettifyLevel = 'off' | 'mixed' | 'python-only';

export interface UserSettings {
    monospace: 'default' | 'system';
    numTemplating: NumTemplatingFormat;
    theme: Theme;
    shortenFrameTexts: ShortenMode;
    reverseFlameByDefault: boolean;
    pythonPrettifyLevel: PythonPrettifyLevel;
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
        reverseFlameByDefault: true,
        numTemplating: 'hugenum',
        pythonPrettifyLevel: 'off',
        ...userSettings,
        theme,
    };
};
