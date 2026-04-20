import React from 'react';

import {
    QueryLanguageEditor,
    type QueryLanguageEditorProps,
    QueryLanguageHelpPopover,
} from 'src/components/QueryLanguageEditor';
import { cn } from 'src/utils/cn';

import './SelectorInput.scss';


const b = cn('selector-input');

export interface SelectorInputProps extends Omit<QueryLanguageEditorProps, 'height'> {}

export const SelectorInput: React.FC<SelectorInputProps> = props => (
    <div className={b()}>
        <QueryLanguageEditor
            height="18px"
            {...props}
        />
        <QueryLanguageHelpPopover />
    </div>
);
