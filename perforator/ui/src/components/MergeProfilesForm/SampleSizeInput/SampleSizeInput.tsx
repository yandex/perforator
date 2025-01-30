import React from 'react';

import { Select } from '@gravity-ui/uikit';

import { uiFactory } from 'src/factory';

import './SampleSizeInput.scss';


export interface SampleSizeInputProps {
    value: number;
    onUpdate: (value: number) => void;
}

export const SampleSizeInput: React.FC<SampleSizeInputProps> = props => {
    const options = uiFactory().sampleSizes().map(size => ({
        content: size,
        value: size.toString(),
    }));
    return (
        <div className="sample-size-input">
            <span className="sample-size-input__caption">Sample size</span>
            <Select
                className="sample-size-input__select"
                value={[props.value.toString()]}
                options={options}
                onUpdate={values => props.onUpdate(Number(values[0]))}
            />
        </div>
    );
};
