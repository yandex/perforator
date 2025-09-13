import * as React from 'react'
import { useCallback, useEffect, useState } from 'react';

import { Magnifier } from '@gravity-ui/icons';
import { Checkbox, Dialog, Icon, TextInput } from '@gravity-ui/uikit';

import { useRegexError } from './useRegexError';

import './RegexpDialog.css';


interface RegexpDialogProps {
    showDialog: boolean;
     onCloseDialog: () => void;
     initialSearch?: string | null;
     initialExact?: boolean | null;
     onSearchUpdate: (str: string, exactMatch?: boolean) => void;
}

export function RegexpDialog({ showDialog, onCloseDialog, onSearchUpdate, initialExact, initialSearch }: RegexpDialogProps) {
    const [searchQuery, setSearchQuery] = useState(initialSearch ?? '');
    const [exact, setExact] = useState(initialExact ?? false);
    const controlRef = React.useRef<null | HTMLInputElement>(null)
    const regexError = useRegexError(searchQuery);

    const handleKeyDown = useCallback((e: KeyboardEvent) => {
        if (e.key === 'Enter' && !regexError) {
            onSearchUpdate(searchQuery, exact);
            e.preventDefault()
        }
    }, [exact, onSearchUpdate, regexError, searchQuery]);

    const handleApply = () => {
        if (regexError) {
            return;
        }

        onSearchUpdate(searchQuery, exact);

    };

    useEffect(() => {
        document.addEventListener('keydown', handleKeyDown);
        return () => document.removeEventListener('keydown', handleKeyDown);
    }, [handleKeyDown]);

    const handleSearchUpdate = (str: string) => {
        setSearchQuery(str);
    };
    return (
        <Dialog initialFocus={controlRef} className="regexp-dialog__dialog" size="l" open={showDialog} onClose={onCloseDialog}>
            <Dialog.Header insertBefore={<Icon className="regexp-dialog__header-icon" data={Magnifier}/>} caption="Search"/>
            <Dialog.Body>
                <TextInput
                    note={'Regular expressions are supported'}
                    autoFocus={true}
                    controlRef={controlRef}
                    value={searchQuery}
                    onUpdate={handleSearchUpdate}
                    error={Boolean(regexError)}
                    errorMessage={regexError} />
                <Checkbox title="Disable regex parsing, literal mode" checked={exact} onUpdate={setExact}>Exact match</Checkbox>
            </Dialog.Body>
            <Dialog.Footer
                onClickButtonCancel={onCloseDialog}
                textButtonCancel="Cancel"
                propsButtonApply={{ disabled: Boolean(regexError) }}
                onClickButtonApply={handleApply}
                textButtonApply={'Search'}
            />
        </Dialog>
    );
}
