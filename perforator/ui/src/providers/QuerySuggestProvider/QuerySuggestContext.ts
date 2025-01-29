import React from 'react';

import type { QueryFields } from './fields';


export interface QuerySuggestContextProps {
    fields: QueryFields;
}

export const QuerySuggestContext = React.createContext<Optional<QuerySuggestContextProps>>(undefined);

export const useQuerySuggestContext = () => {
    const value = React.useContext(QuerySuggestContext);
    if (value === undefined) {
        throw new Error('useQuerySuggest must be used within QuerySuggestProvider');
    }
    return value;
};
