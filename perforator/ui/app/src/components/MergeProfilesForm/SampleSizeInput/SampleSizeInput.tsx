import React from 'react';

import { Check, Plus, Xmark } from '@gravity-ui/icons';
import { Button, Icon, NumberInput, Popup, Select } from '@gravity-ui/uikit';

import { LocalStorageKey } from 'src/const/localStorage';
import { uiFactory } from 'src/factory';

import './SampleSizeInput.scss';


const ADD_OPTION_VALUE = '__add_option__';

const getCustomSampleSizes = (): number[] => {
    try {
        const stored = localStorage.getItem(LocalStorageKey.CustomSampleSizes);
        if (stored) {
            const parsed = JSON.parse(stored);
            if (Array.isArray(parsed)) {
                return parsed.filter((n): n is number => typeof n === 'number' && n > 0).sort((a, b) => a - b);
            }
        }
    } catch {}
    return [];
};

const saveCustomSampleSizes = (sizes: number[]): void => {
    localStorage.setItem(LocalStorageKey.CustomSampleSizes, JSON.stringify(sizes));
};

export interface SampleSizeInputProps {
    value: number;
    onUpdate: (value: number) => void;
}

export const SampleSizeInput: React.FC<SampleSizeInputProps> = ({ value, onUpdate }: SampleSizeInputProps) => {
    const [customSizes, setCustomSizes] = React.useState<number[]>(getCustomSampleSizes);
    const [popupOpen, setPopupOpen] = React.useState(false);
    const [inputValue, setInputValue] = React.useState<number | null>(null);
    const anchorRef = React.useRef<HTMLDivElement>(null);

    const defaultSizes = uiFactory().sampleSizes();
    const allSizes = new Set([...defaultSizes, ...customSizes]);

    const addCustomOption = {
        content: (
            <span className="sample-size-input__add-option">
                <Icon data={Plus} size={14} />
                <span>Add option</span>
            </span>
        ),
        value: ADD_OPTION_VALUE,
    };

    const customOptions = customSizes.map(size => ({
        content:  (
            <span className="sample-size-input__custom-option">
                <span>{size}</span>
                <Button
                    size="xs"
                    view="flat"
                    className="sample-size-input__delete-btn"
                    onClick={(e) => {
                        e.stopPropagation();
                        handleDeleteCustomSize(size);
                    }}
                >
                    <Icon data={Xmark} size={12} />
                </Button>
            </span>
        ),
        value: size.toString(),
    }));

    const defaultOptions = defaultSizes.map(size => ({
        content: size.toString(),
        value: size.toString(),
    }));

    const options = [...defaultOptions, ...customOptions, addCustomOption];


    const handleDeleteCustomSize = (size: number) => {
        const newCustomSizes = customSizes.filter(s => s !== size);
        setCustomSizes(newCustomSizes);
        saveCustomSampleSizes(newCustomSizes);

        // If the deleted size was selected, switch to default
        if (value === size) {
            onUpdate(uiFactory().defaultSampleSize());
        }
    };

    const handleSelectUpdate = (values: string[]) => {
        const selected = values[0];
        if (selected === ADD_OPTION_VALUE) {
            setPopupOpen(true);
            return;
        }
        onUpdate(Number(selected));
    };

    const handleAddOption = () => {
        const numValue = inputValue ?? 0;
        if (numValue > 0 && !allSizes.has(numValue)) {
            const newCustomSizes = [...customSizes, numValue].sort((a, b) => a - b);
            setCustomSizes(newCustomSizes);
            saveCustomSampleSizes(newCustomSizes);
            onUpdate(numValue);
        }
        setInputValue(null);
        setPopupOpen(false);
    };

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter') {
            e.preventDefault();
            handleAddOption();
        }
    };

    return (
        <div className="sample-size-input">
            <span className="sample-size-input__caption">Profile count</span>
            <div className="sample-size-input__select-wrapper">
                <Select
                    className="sample-size-input__select"
                    value={[value.toString()]}
                    options={options}
                    onUpdate={handleSelectUpdate}
                />
                <div ref={anchorRef} className="sample-size-input__anchor" />
            </div>
            <Popup
                anchorElement={anchorRef.current}
                open={popupOpen}
                onOpenChange={setPopupOpen}
                placement="bottom-start"
            >
                <div className="sample-size-input__popup">
                    <NumberInput
                        value={inputValue}
                        onUpdate={setInputValue}
                        onKeyDown={handleKeyDown}
                        placeholder="Enter value"
                        autoFocus
                    />
                    <Button
                        size="s"
                        view="action"
                        onClick={handleAddOption}
                        disabled={!inputValue || inputValue <= 0}
                    >
                        <Icon data={Check} size={14} />
                    </Button>
                </div>
            </Popup>
        </div>
    );
};
