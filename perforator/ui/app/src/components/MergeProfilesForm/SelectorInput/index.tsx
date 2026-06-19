import type { QueryInput, QueryInputRenderer } from '../QueryInput';

import { SelectorInput } from './SelectorInput';


const renderSelectorInput: QueryInputRenderer = (query, setQuery, setTableSelector) => {
    const doSetQuery = (selector: Optional<string>) => {
        setQuery(currentQuery => ({
            ...currentQuery,
            selector: selector || '',
        }));
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
            }} />
    );
};

export const SELECTOR_QUERY_INPUT: QueryInput = {
    name: 'Selector',
    queryField: 'selector',
    render: renderSelectorInput,
};
