import React from 'react';

import { Button, TextInput } from '@gravity-ui/uikit';

import { cn } from 'src/utils/cn';

import './CustomRangeInput.scss';


const b = cn('time-interval-selector');

export interface CustomRangeInputProps {
    selected?: boolean;
    onUpdate?: (value: string) => void;
}

export const CustomRangeInput: React.FC<CustomRangeInputProps> = ({ selected: selectedProp, onUpdate }: CustomRangeInputProps) => {
    const [value, setValue] = React.useState<string>('');
    const [active, setActive] = React.useState(false);
    const [selected, setSelected] = React.useState(selectedProp);

    React.useEffect(() => {
        if (selectedProp !== selected) {
            setSelected(selectedProp);
            if (!selectedProp) {
                setValue('');
                onUpdate?.('');
            }
        }
    }, [selectedProp, selected]);

    const handleKeyDown = React.useCallback((ev: React.KeyboardEvent<HTMLInputElement>) => {
        if (ev.key === 'Enter') {
            ev.preventDefault();
            const newValue = ev.currentTarget.value;
            if (newValue) {
                onUpdate?.(value);
                setActive(false);
                setSelected(true);
            }
        }
    }, [onUpdate, setActive, setSelected, value]);

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
