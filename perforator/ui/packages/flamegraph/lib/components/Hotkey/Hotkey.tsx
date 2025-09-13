import React from 'react';

import { Hotkey as GravityHotkey, HotkeyProps as GravityHotkeyProps } from '@gravity-ui/uikit';

import './Hotkey.css';


export interface HotkeyProps extends Pick<GravityHotkeyProps, 'view' | 'value'> {
}

export const Hotkey: React.FC<HotkeyProps> = props => (
    <GravityHotkey
        view={'light'}
        className="hotkey"
        {...props}
    />
);
