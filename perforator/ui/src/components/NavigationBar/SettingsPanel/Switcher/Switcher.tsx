import React from 'react';

import { RadioButton } from '@gravity-ui/uikit';


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
        <RadioButton.Option key={value} value={value}>
            {title}
        </RadioButton.Option>
    ));
    return (
        <RadioButton
            value={props.value}
            onUpdate={props.onUpdate}
        >
            {items}
        </RadioButton>
    );
};
