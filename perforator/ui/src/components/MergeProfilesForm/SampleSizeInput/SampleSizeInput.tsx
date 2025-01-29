import React from 'react';

import { Select } from '@gravity-ui/uikit';

import './SampleSizeInput.scss';


const SAMPLE_SIZES = [1, 2, 5, 10, 20, 50, 100];

export interface SampleSizeInputProps {
    value: number;
    onUpdate: (value: number) => void;
}

export const SampleSizeInput: React.FC<SampleSizeInputProps> = props => {
    const options = SAMPLE_SIZES.map(size => ({
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
