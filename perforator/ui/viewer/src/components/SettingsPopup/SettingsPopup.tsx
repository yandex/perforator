import React, { useRef, useState } from 'react';
import { Text, Button, SegmentedRadioGroup, Popup } from '@gravity-ui/uikit';
import { Gear } from '@gravity-ui/icons';
import type { UserSettings, ShortenMode, NumTemplatingFormat } from '@perforator/flamegraph';

import i18n from './i18n';

import './SettingsPopup.css';
import { cn } from '../../utils/cn';

export interface SettingsPopupProps {
    settings: UserSettings;
    onSettingsChange: (settings: UserSettings) => void;
}

const b = cn('settings-popup');

export const SettingsPopup: React.FC<SettingsPopupProps> = ({ settings, onSettingsChange }) => {
    const [open, setOpen] = useState(false);
    const popupAnchorRef = useRef(null);
    const handleMonospaceChange = (value: string) => {
        onSettingsChange({
            ...settings,
            monospace: value as 'default' | 'system',
        });
    };

    const handleNumTemplatingChange = (value: string) => {
        onSettingsChange({
            ...settings,
            numTemplating: value as NumTemplatingFormat,
        });
    };

    const handleShortenFrameTextsChange = (value: string) => {
        onSettingsChange({
            ...settings,
            shortenFrameTexts: value as ShortenMode,
        });
    };

    const content = (
        <div className={b()}>
            <div className={b('section')}>
                <Text variant={"subheader-1"} className={b('label')}>{i18n('monospaceFont')}</Text>
                <SegmentedRadioGroup
                    name="monospace"
                    value={settings.monospace}
                    onUpdate={handleMonospaceChange}
                    size="m"
                >
                    <SegmentedRadioGroup.Option value="default">{i18n('defaultFont')}</SegmentedRadioGroup.Option>
                    <SegmentedRadioGroup.Option value="system">{i18n('systemFont')}</SegmentedRadioGroup.Option>
                </SegmentedRadioGroup>
            </div>

            <div className={b('section')}>
                <Text variant={"subheader-1"} className={b('label')}>{i18n('numberFormat')}</Text>
                <SegmentedRadioGroup
                    name="numTemplating"
                    value={settings.numTemplating}
                    onUpdate={handleNumTemplatingChange}
                    size="m"
                >
                    <SegmentedRadioGroup.Option value="exponent">{i18n('exponentFormat')}</SegmentedRadioGroup.Option>
                    <SegmentedRadioGroup.Option value="hugenum">{i18n('hugeNumFormat')}</SegmentedRadioGroup.Option>
                </SegmentedRadioGroup>
            </div>

            <div className={b('section')}>
                <Text variant={"subheader-1"} className={b('label')}>{i18n('shortenFrameTexts')}</Text>
                <SegmentedRadioGroup
                    name="shortenFrameTexts"
                    value={settings.shortenFrameTexts}
                    onUpdate={handleShortenFrameTextsChange}
                    size="m"
                >
                    <SegmentedRadioGroup.Option value="true">{i18n('shortenTrue')}</SegmentedRadioGroup.Option>
                    <SegmentedRadioGroup.Option value="false">{i18n('shortenFalse')}</SegmentedRadioGroup.Option>
                    <SegmentedRadioGroup.Option value="hover">{i18n('shortenHover')}</SegmentedRadioGroup.Option>
                </SegmentedRadioGroup>
            </div>
        </div>
    );

    return (
        <>
            <Button ref={popupAnchorRef} view="flat" size="m" onClick={() => setOpen(!open)}>
                <Button.Icon>
                    <Gear />
                </Button.Icon>
            </Button>
            <Popup
                placement="bottom-end"
                anchorRef={popupAnchorRef}
                open={open}
                onOpenChange={setOpen}
            >
                {content}
            </Popup>
        </>
    );
};
