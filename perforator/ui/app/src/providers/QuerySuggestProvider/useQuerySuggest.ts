import React from 'react';

import { AxiosError } from 'axios';

import { apiClient } from 'src/utils/api';
import { useDebounce } from 'src/utils/debounce';
import {
    makeSelectorFromConditions,
    SELECTOR_CONDITION_KEYS,
    type SelectorCondition,
} from 'src/utils/selector';
import { createErrorToast } from 'src/utils/toaster';

import { useQuerySuggestContext } from './QuerySuggestContext';


export type SuggestToken = SelectorCondition;
export type Suggestion = string;
export type Suggestions = Optional<Suggestion[]>;

type SuggestHandler = (state: SuggestState) => Promise<Suggestions>;

export interface SuggestState {
    tokens: SuggestToken[];
    currentToken: Optional<SuggestToken>;
    key: keyof SuggestToken;
}

const fetchSuggestions = (state: SuggestState, options: {abortController: AbortController}) => (
    async (): Promise<Optional<string[]>> => {
        const { currentToken } = state;
        const selector = makeSelectorFromConditions(
            state.tokens.filter(token => (
                token !== currentToken
                && SELECTOR_CONDITION_KEYS.every(key => token[key] !== undefined)
            )),
        );
        try {
            const response = await apiClient.getSuggestions({
                'Field': currentToken?.field ?? '',
                'Regex': currentToken?.value ?? '',
                'Selector': selector,
                'Paginated.Offset': 0,
                'Paginated.Limit': 100,
            }, { signal: options.abortController.signal });
            return (
                response?.data?.SuggestSupported
                    ? (response?.data?.Suggestions ?? []).map(suggestion => suggestion.Value)
                    : undefined
            );
        } catch (error: unknown) {
            let content: string | undefined;
            if (error instanceof AxiosError) {
                if (error.code === 'ERR_CANCELED') {
                    return undefined;
                }
                content = error.message;
            }
            createErrorToast(
                error,
                { name: 'list-suggestions', title: 'Failed to load suggestions', content },
            );
        }
        return undefined;
    }
);

export const useQuerySuggest = () => {
    const abortControllerRef = React.useRef(new AbortController());
    React.useEffect(() => {
        return () => abortControllerRef.current.abort();
    }, []);
    const querySuggestContext = useQuerySuggestContext();
    const { fields } = querySuggestContext;

    const debounce = useDebounce();

    const findField = (value: Optional<string>) => fields.get(value ?? '');

    const suggestFields: SuggestHandler = React.useCallback(async state => {
        // we should ignore only fields with perfect matches
        const usedFields = new Set(
            state.tokens
                .filter(token => token.operator === '=' && token.field !== state.currentToken?.field)
                .map(token => token.field),
        );
        return (
            [...fields.keys()]
                .filter(field => !usedFields.has(field))
                .filter(field => field.includes(state.currentToken?.field ?? ''))
        );
    }, [fields]);

    const suggestOperators: SuggestHandler = React.useCallback(async state => (
        findField(state.currentToken?.field)?.operators
            ?.filter(field => field.includes(state.currentToken?.operator ?? ''))
    ), [fields]);

    const suggestValues: SuggestHandler = React.useCallback(async state => {
        if (!abortControllerRef.current.signal.aborted) {
            abortControllerRef.current.abort();
        }
        abortControllerRef.current = new AbortController();
        const fetchValues = fetchSuggestions(state, { abortController: abortControllerRef.current });
        return await debounce(async () => await fetchValues());
    }, [fields]);

    const suggests: {[key in keyof SuggestToken]: SuggestHandler} = React.useMemo(() => ({
        field: suggestFields,
        operator: suggestOperators,
        value: suggestValues,
    }), [
        suggestFields,
        suggestOperators,
        suggestValues,
    ]);

    const handleQuerySuggest = React.useCallback(async (state: SuggestState) => (
        (await (suggests[state.key] || (() => undefined))(state))
            ?.filter(value => value)
    ), [suggests]);

    return {
        handleQuerySuggest,
    };
};
