import { useCallback, useEffect, useMemo, useState } from 'react';

import MagnifierIcon from '@gravity-ui/icons/svgs/magnifier.svg?raw';
import { Dialog, Icon, TextInput } from '@gravity-ui/uikit';

import './RegexpDialog.scss';


interface RegexpDialogProps {
    showDialog: boolean;
     onCloseDialog: () => void;
     initialSearch?: string | null;
     onSearchUpdate: (str: string) => void;
}

export function RegexpDialog({ showDialog, onCloseDialog, onSearchUpdate, initialSearch }: RegexpDialogProps) {
    const [searchQuery, setSearchQuery] = useState(initialSearch ?? '');

    const regexError = useMemo(() => {
        try {
            RegExp(searchQuery);
            return null;
        } catch (error: unknown) {
            if (typeof error === 'object' && error !== null && 'message' in error && typeof error.message === 'string') {
                return error.message;
            }
            else if (typeof error === 'string') {
                return error;
            }
            else {
                return 'Unknown error in regexp';
            }
        }
    }, [searchQuery]);

    const handleKeyDown = useCallback((e: KeyboardEvent) => {
        if (e.key === 'Enter' && !regexError) {
            onSearchUpdate(searchQuery);
        }
    }, [onSearchUpdate, regexError, searchQuery]);

    const handleApply = () => {
        if (regexError) {
            return;
        }

        onSearchUpdate(searchQuery);

    };

    useEffect(() => {
        document.addEventListener('keydown', handleKeyDown);
        return () => document.removeEventListener('keydown', handleKeyDown);
    }, [handleKeyDown]);

    const handleSearchUpdate = (str: string) => {
        setSearchQuery(str);
    };
    return (
        <Dialog className="regexp-dialog__dialog" size="l" open={showDialog} onClose={onCloseDialog}>
            <Dialog.Header insertBefore={<Icon className="regexp-dialog__header-icon" data={MagnifierIcon}/>} caption="Search"/>
            <Dialog.Body>
                <TextInput
                    note={'Regular expressions are supported'}
                    autoFocus
                    value={searchQuery}
                    onUpdate={handleSearchUpdate}
                    error={Boolean(regexError)}
                    errorMessage={regexError} />
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
