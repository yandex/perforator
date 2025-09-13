import React from 'react';

import {
    QueryLanguageEditor,
    type QueryLanguageEditorProps,
} from 'src/components/QueryLanguageEditor';

import './SelectorInput.scss';


export interface SelectorInputProps extends Omit<QueryLanguageEditorProps, 'height'> {}

export const SelectorInput: React.FC<SelectorInputProps> = props => (
    <div className="selector-input">
        <QueryLanguageEditor
            height="18px"
            {...props}
        />
    </div>
);

