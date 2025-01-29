import React from 'react';

import {
    QueryLanguageEditor,
    type QueryLanguageEditorProps,
} from 'src/components/QueryLanguageEditor';

import type { QueryInput, QueryInputRenderer } from '../QueryInput';

import './SelectorInput.scss';


export interface SelectorInputProps extends Omit<QueryLanguageEditorProps, 'height'> {}

export const SelectorInput: React.FC<SelectorInputProps> = props => (
    <div className="selector-input">
        <QueryLanguageEditor
            height="28px"
            {...props}
        />
    </div>
);

const renderSelectorInput: QueryInputRenderer = (query, setQuery, setTableSelector) => {
    const doSetQuery = (selector: Optional<string>) => {
        setQuery({
            ...query,
            selector: selector || '',
        });
    };
    return (
        <SelectorInput
            selector={query.selector}
            onUpdate={doSetQuery}
            onSelectorChange={(selector: Optional<string>) => {
                doSetQuery(selector);
                if (selector) {
                    // no need to list profiles manually
                    setTableSelector?.(selector);
                }
            }}
        />
    );
};

export const SELECTOR_QUERY_INPUT: QueryInput = {
    name: 'Selector',
    queryField: 'selector',
    render: renderSelectorInput,
};
