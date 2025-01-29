import React from 'react';

import { getQueryFields, type QueryFields } from './fields';
import { QuerySuggestContext, type QuerySuggestContextProps } from './QuerySuggestContext';


export interface QuerySuggestProviderProps {
    children?: React.ReactNode;
}

export const QuerySuggestProvider: React.FC<QuerySuggestProviderProps> = props => {
    const [fields, setFields] = React.useState<QueryFields>(new Map());

    const setFieldsAsync = React.useCallback(async () => {
        const fieldsList = await getQueryFields();
        setFields(new Map(fieldsList.map(field => [field.field, field])));
    }, []);

    React.useEffect(() => { setFieldsAsync(); }, [setFieldsAsync]);

    const value: QuerySuggestContextProps = { fields };
    return (
        <QuerySuggestContext.Provider value={value}>
            {props.children}
        </QuerySuggestContext.Provider>
    );
};
