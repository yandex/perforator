import React from 'react';

import type { UserSettings } from './UserSettings';


export interface UserSettingsContextProps {
    userSettings: UserSettings;
    setUserSettings: (userSettings: UserSettings) => void;
}

export const UserSettingsContext = React.createContext<UserSettingsContextProps | undefined>(undefined);

export const useUserSettings = () => {
    const value = React.useContext(UserSettingsContext);
    if (value === undefined) {
        throw new Error('useUserSettings must be used within UserSettingsProvider');
    }
    return value;
};
