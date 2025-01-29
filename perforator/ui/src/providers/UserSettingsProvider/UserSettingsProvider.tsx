import React from 'react';

import { LocalStorageKey } from 'src/const/localStorage';

import type { UserSettings } from './UserSettings';
import { initialUserSettings } from './UserSettings';
import type { UserSettingsContextProps } from './UserSettingsContext';
import { UserSettingsContext } from './UserSettingsContext';


export interface UserSettingsProviderProps {
    children?: React.ReactNode;
}

export const UserSettingsProvider: React.FC<UserSettingsProviderProps> = props => {
    const [userSettings, setUserSettingsImpl] = React.useState(initialUserSettings());

    const setUserSettings = React.useCallback((value: UserSettings) => {
        setUserSettingsImpl(value);
        localStorage.setItem(LocalStorageKey.UserSettings, JSON.stringify(value));
    }, []);

    const value: UserSettingsContextProps = { userSettings, setUserSettings };
    return (
        <UserSettingsContext.Provider value={value}>
            {props.children}
        </UserSettingsContext.Provider>
    );
};
