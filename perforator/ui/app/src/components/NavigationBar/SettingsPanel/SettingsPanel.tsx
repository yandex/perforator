import React from 'react';

import { Palette } from '@gravity-ui/icons';
import { Settings } from '@gravity-ui/navigation';
import { Switch, type Theme } from '@gravity-ui/uikit';

import { useUserSettings } from 'src/providers/UserSettingsProvider';
import type { NumTemplatingFormat, PythonPrettifyLevel, ShortenMode } from 'src/providers/UserSettingsProvider/UserSettings';

import i18n from './i18n';
import { Switcher } from './Switcher/Switcher';


export interface SettingsPanelProps {}

export const SettingsPanel: React.FC<SettingsPanelProps> = () => {
    const { userSettings, setUserSettings } = useUserSettings();
    return (
        <Settings>
            <Settings.Page id="appearance" title={i18n('appearance')} icon={{ data: Palette }}>
                <Settings.Section title={i18n('appearance')}>
                    <Settings.Item title={i18n('interfaceTheme')}>
                        <Switcher
                            value={userSettings.theme}
                            onUpdate={theme => setUserSettings({ ...userSettings, theme: (theme as Theme) })}
                            options={[
                                { value: 'light', title: i18n('light') },
                                { value: 'dark', title: i18n('dark') },
                                { value: 'system', title: i18n('system') },
                            ]}
                        />
                    </Settings.Item>
                    <Settings.Item title={i18n('fullFunctionNames')}>
                        <Switcher
                            value={userSettings.shortenFrameTexts}
                            onUpdate={shorten => setUserSettings({ ...userSettings, shortenFrameTexts: (shorten as ShortenMode) })}
                            options={[
                                { value: 'false', title: i18n('always') },
                                { value: 'hover', title: i18n('onHover') },
                                { value: 'true', title: i18n('never') },
                            ]}
                        />
                    </Settings.Item>
                    <Settings.Item title={i18n('compactNumbers')}>
                        <Switcher
                            value={userSettings.numTemplating}
                            onUpdate={numTemplating => setUserSettings({ ...userSettings, numTemplating: numTemplating as NumTemplatingFormat })}
                            options={[
                                { value: 'exponent', title: i18n('exponent') },
                                { value: 'hugenum', title: i18n('metricPrefix') },
                            ]}
                        />
                    </Settings.Item>
                    <Settings.Item title={i18n('monospaceFont')}>
                        <Switch
                            checked={userSettings.monospace === 'system'}
                            onUpdate={checked => setUserSettings({ ...userSettings, monospace: checked ? 'system' : 'default' })}
                        />
                    </Settings.Item>
                    <Settings.Item title={i18n('reverseFlamegraph')}>
                        <Switch
                            checked={userSettings.reverseFlameByDefault}
                            onUpdate={checked => setUserSettings({ ...userSettings, reverseFlameByDefault: checked })}
                        />
                    </Settings.Item>
                </Settings.Section>
                <Settings.Section title={i18n('experimental')}>
                    <Settings.Item
                        description={i18n('pythonPrettificationDescription')}
                        title={i18n('pythonPrettification')}
                    >
                        <Switcher
                            value={userSettings.pythonPrettifyLevel}
                            onUpdate={level => setUserSettings({ ...userSettings, pythonPrettifyLevel: level as PythonPrettifyLevel })}
                            options={[
                                { value: 'off', title: i18n('off') },
                                { value: 'mixed', title: i18n('mixed') },
                                { value: 'python-only', title: i18n('pythonOnly') },
                            ]}
                        />
                    </Settings.Item>
                </Settings.Section>
            </Settings.Page>
        </Settings>
    );
};
