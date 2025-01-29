import React from 'react';

import { Button, TextInput } from '@gravity-ui/uikit';

import { cn } from 'src/utils/cn';

import './CustomRangeInput.scss';


const b = cn('time-interval-selector');

export interface CustomRangeInputProps {
    selected?: boolean;
    onUpdate?: (value: string) => void;
}

export const CustomRangeInput: React.FC<CustomRangeInputProps> = props => {
    const [value, setValue] = React.useState<string>('');
    const [active, setActive] = React.useState(false);
    const [selected, setSelected] = React.useState(props.selected);

    React.useEffect(() => {
        if (props.selected !== selected) {
            setSelected(props.selected);
            if (!props.selected) {
                setValue('');
                props.onUpdate?.('');
            }
        }
    }, [props.selected, selected]);

    const handleKeyDown = React.useCallback((ev: React.KeyboardEvent<HTMLInputElement>) => {
        if (ev.key === 'Enter') {
            ev.preventDefault();
            const newValue = ev.currentTarget.value;
            if (newValue) {
                props.onUpdate?.(value);
                setActive(false);
                setSelected(true);
            }
        }
    }, [props.onUpdate, setActive, setSelected, value]);

    const renderInput = React.useCallback(() => (
        <TextInput
            className={b('custom-range-input')}
            size="s"
            value={value}
            autoFocus={active}
            onUpdate={setValue}
            onKeyDown={handleKeyDown}
            placeholder="6h"
        />
    ), [active, handleKeyDown, setValue, value]);

    const renderButton = React.useCallback(() => (
        <Button
            size="s"
            view="flat"
            selected={true}
            onClick={() => {
                setActive(true);
                setSelected(false);
            }}
        >
            {value}
        </Button>
    ), [setActive, setSelected, value]);

    return (
        <>
            {(!selected || active) ? renderInput() : renderButton()}
        </>
    );
};
