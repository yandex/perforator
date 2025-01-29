import type { SelectorCondition } from 'src/utils/selector';


export const QUERY_LANGUAGE_ID = 'query-language';

export const TOKEN_FIELD_TO_MONACO_TYPE: {[key in keyof SelectorCondition]: string} = {
    field: 'string.key.json',
    operator: 'keyword',
    value: 'string.value.json',
};

export const makeMonacoTokenType = (name: string): string => `${name}.${QUERY_LANGUAGE_ID}`;

export const MONACO_TYPE_TO_TOKEN_FIELD = Object.fromEntries(
    Object.entries(TOKEN_FIELD_TO_MONACO_TYPE)
        .map(([key, value]) => [makeMonacoTokenType(value), key as keyof SelectorCondition]),
);
