import React, { useMemo, useState } from 'react';

import { useNavigate } from 'react-router-dom';

import type {  FlamegraphProps, QueryKeys, UserSettings } from '@perforator/flamegraph';
import { calculateTopForTable, Flamegraph, prerenderColors, SideBySide, TopTable, useLeftHeavyProfile } from '@perforator/flamegraph';

import { Loader, useThemeType } from '@gravity-ui/uikit';
import { Tabs } from '@gravity-ui/uikit/legacy';
import { createSuccessToast } from '../../utils/toaster';

import {  useTypedQuery } from '../../query-utils';
import { SettingsPopup } from '../SettingsPopup/SettingsPopup';

import './Visualisation.css';
import { cn } from '../../utils/cn';

export type Tab = 'flame' | 'top' | 'sbs';
export interface VisualisationProps
    extends Pick<FlamegraphProps, 'profileData' | 'theme'> {
    loading: boolean;
}

const b = cn('vis')

export const Visualisation: React.FC<VisualisationProps> = ({ profileData, ...props }) => {
    const navigate = useNavigate();
    const [getQuery, setQuery] = useTypedQuery<'tab' | QueryKeys>();
    const tab: Tab = getQuery('tab', 'flame') as Tab;
    const isLeftHeavy = getQuery('leftHeavy', 'false') === 'true';
    const setIsLeftHeavy = React.useCallback((value: boolean) => {
        setQuery({ 'leftHeavy': value ? 'true' : 'false' });
    }, [setQuery]);
    const isTopTab = tab === 'top';
    const [isFirstTopRender, setIsFirstTopRender] = React.useState(isTopTab);
    React.useEffect(() => {
        setIsFirstTopRender(isFirstTopRender || isTopTab);
    }, [isFirstTopRender, isTopTab]);
    const theme = useThemeType()

    const prerenderedNewData = React.useMemo(() => {
        if (profileData) {
            return prerenderColors(profileData, { theme });
        }
        return null;
    }, [profileData, theme]);

     const isDiff = useMemo(() => Boolean(profileData?.rows?.[0][0].baseEventCount), [profileData])
    const newProfileData = useLeftHeavyProfile(prerenderedNewData, isLeftHeavy, getQuery, setQuery);


    const topData = React.useMemo(() => {
        return profileData && isFirstTopRender && newProfileData
            ? calculateTopForTable(
                  newProfileData.rows,
                  profileData.stringTable.length,
                  { rootCoords: [0, 0], omitted: [], keepCoords: null }
              )
            : null;
    }, [profileData, isFirstTopRender]);

    const [userSettings, setUserSettings] = useState<UserSettings>(localStorage.getItem('userSettings') ? JSON.parse(localStorage.getItem('userSettings')!) : {
        monospace: 'default',
        numTemplating: 'exponent',
        reverseFlameByDefault: true,
        shortenFrameTexts: 'false',
        theme: 'system'
    });

    const handleUserSettings = React.useCallback((settings: UserSettings) => {
        setUserSettings(settings);
        try {
            localStorage.setItem('userSettings', JSON.stringify(settings));
        } catch (e) {
            console.error(e);
        }
    }, []);

    const flamegraphProps: FlamegraphProps = {
        goToDefinitionHref: () => '',
        profileData: newProfileData,
        getState: getQuery,
        isDiff,
        setState: setQuery,
        onSuccess: createSuccessToast,
        userSettings,
        isLeftHeavy,
        onChangeLeftHeavy: setIsLeftHeavy,
        ...props
    };

    let content: React.JSX.Element | undefined;

    if (props.loading) {
        content = <Loader />;
    } else {
        if (tab === 'flame') {
            content = <Flamegraph {...flamegraphProps} />;
        }
        if (tab === 'top' && topData && profileData) {
            const topTableProps = {
                goToDefinitionHref: () => '',
                topData,
                profileData,
                userSettings,
                navigate,
                getState: getQuery,
                setState: setQuery,
                ...props
            };
            content = <TopTable {...topTableProps} />;
        }

        if(tab === 'sbs') {
            content = <SideBySide navigate={navigate} {...flamegraphProps} />;
        }
    }

    return (
        <div className={b({sbs: tab === 'sbs'})}>
            <div className={b('header')}>
                <Tabs
                    className={'vis_tabs'}
                    activeTab={tab}
                    items={[
                        { id: 'flame', title: 'Flamegraph' },
                        { id: 'top', title: 'Top' },
                        { id: 'sbs', title: 'Side by side' }
                    ]}
                    onSelectTab={(newTab) => {
                        setQuery({ tab: newTab });
                    }}
                />
                <SettingsPopup settings={userSettings} onSettingsChange={handleUserSettings} />
            </div>
            {content}
        </div>
    );
};
