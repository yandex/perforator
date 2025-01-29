import React from 'react';

import { Palette } from '@gravity-ui/icons';
import { Settings } from '@gravity-ui/navigation';
import { Switch, type Theme } from '@gravity-ui/uikit';

import { useUserSettings } from 'src/providers/UserSettingsProvider';
import type { ShortenMode } from 'src/providers/UserSettingsProvider/UserSettings';

import { Switcher } from './Switcher/Switcher';


export interface SettingsPanelProps {}

export const SettingsPanel: React.FC<SettingsPanelProps> = () => {
    const { userSettings, setUserSettings } = useUserSettings();
    return (
        <Settings>
            <Settings.Page id="appearance" title="Appearance" icon={{ data: Palette }}>
                <Settings.Section title="Appearance">
                    <Settings.Item title="Interface theme">
                        <Switcher
                            value={userSettings.theme}
                            onUpdate={theme => setUserSettings({ ...userSettings, theme: (theme as Theme) })}
                            options={[
                                { value: 'light', title: 'Light' },
                                { value: 'dark', title: 'Dark' },
                                { value: 'system', title: 'System' },
                            ]}
                        />
                    </Settings.Item>
                    <Settings.Item title="Show full function names">
                        <Switcher
                            value={userSettings.shortenFrameTexts}
                            onUpdate={shorten => setUserSettings({ ...userSettings, shortenFrameTexts: (shorten as ShortenMode) })}
                            options={[
                                { value: 'false', title: 'Always' },
                                { value: 'hover', title: 'On hover' },
                                { value: 'true', title: 'Never' },
                            ]}
                        />
                    </Settings.Item>
                    <Settings.Item title="Use browser monospace font for flamegraph">
                        <Switch
                            checked={userSettings.monospace === 'system'}
                            onUpdate={checked => setUserSettings({ ...userSettings, monospace: checked ? 'system' : 'default' })}
                        />
                    </Settings.Item>
                </Settings.Section>
            </Settings.Page>
        </Settings>
    );
};
