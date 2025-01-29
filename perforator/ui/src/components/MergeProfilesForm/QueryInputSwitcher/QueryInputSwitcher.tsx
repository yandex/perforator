import React from 'react';

import { RadioButton } from '@gravity-ui/uikit';

import { Beta } from 'src/components/Beta/Beta';

import type { QueryInput } from '../QueryInput';


export interface QueryInputSwitcherProps {
    value: string;
    inputs: QueryInput[];
    onUpdate: (input: string) => void;
}

export const QueryInputSwitcher: React.FC<QueryInputSwitcherProps> = props => {
    const options = React.useMemo(() => (
        props.inputs.map(input => (
            <RadioButton.Option key={input.name} value={input.name}>
                {input.name}
                {input.beta && <Beta />}
            </RadioButton.Option>
        ))
    ), [props.inputs]);

    if (options.length <= 1) {
        return <div />;  // to not mess up the flexbox layout
    }

    return (
        <RadioButton
            value={props.value}
            onUpdate={props.onUpdate}
        >
            {options}
        </RadioButton>
    );
};
