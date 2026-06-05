import React from 'react';

import { SegmentedRadioGroup } from '@gravity-ui/uikit';

import { Beta } from 'src/components/Beta/Beta';

import type { QueryInput } from '../QueryInput';


export interface QueryInputSwitcherProps {
    value: string;
    inputs: QueryInput[];
    onUpdate: (input: string) => void;
}

export const QueryInputSwitcher: React.FC<QueryInputSwitcherProps> = ({ value, inputs, onUpdate }: QueryInputSwitcherProps) => {
    const options = React.useMemo(() => (
        inputs.map(input => (
            <SegmentedRadioGroup.Option key={input.name} value={input.name}>
                <span>
                    {input.name}
                    {input.beta && <Beta />}
                </span>
            </SegmentedRadioGroup.Option>
        ))
    ), [inputs]);

    if (options.length <= 1) {
        return <div />;  // to not mess up the flexbox layout
    }

    return (
        <SegmentedRadioGroup
            value={value}
            onUpdate={onUpdate}
        >
            {options}
        </SegmentedRadioGroup>
    );
};
