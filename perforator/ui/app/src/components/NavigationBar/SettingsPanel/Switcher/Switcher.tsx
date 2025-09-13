import React from 'react';

import { SegmentedRadioGroup } from '@gravity-ui/uikit';


export interface SwitcherOption {
    value: string;
    title: string;
}

export interface SwitcherProps {
    value: string;
    onUpdate: (value: string) => void;
    options: SwitcherOption[];
}

export const Switcher: React.FC<SwitcherProps> = (props) => {
    const items = props.options.map(({ value, title }) => (
        <SegmentedRadioGroup.Option key={value} value={value}>
            {title}
        </SegmentedRadioGroup.Option>
    ));
    return (
        <SegmentedRadioGroup
            value={props.value}
            onUpdate={props.onUpdate}
        >
            {items}
        </SegmentedRadioGroup>
    );
};
