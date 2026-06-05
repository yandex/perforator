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

export const Switcher: React.FC<SwitcherProps> = ({ value, onUpdate, options }: SwitcherProps) => {
    const items = options.map(({ value: optionValue, title }) => (
        <SegmentedRadioGroup.Option key={optionValue} value={optionValue}>
            {title}
        </SegmentedRadioGroup.Option>
    ));
    return (
        <SegmentedRadioGroup
            value={value}
            onUpdate={onUpdate}
        >
            {items}
        </SegmentedRadioGroup>
    );
};
