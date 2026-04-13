import type {
    TokenizedInputSuggestionContext,
    TokenizedInputSuggestionsItem as TokenizedSuggestionsItem,
    TokenizedInputTokenField as TokenField,
    TokenizedInputTokenOnKeyDownOptions as TokenOnKeyDownOptions,
} from '@gravity-ui/components';

import {
    QUERY_LANGUAGE_OPERATORS,
    type Suggestions,
    type SuggestState,
} from 'src/providers/QuerySuggestProvider';
import {
    makeSelectorFromConditions,
    nextSelectorConditionKey,
    parseServiceFromSelector,
    type SelectorCondition,
} from 'src/utils/selector';


export type Token = SelectorCondition;

export const parseTokensString = (data: Optional<string>): Token[] => {
    const tokensList = JSON.parse(data || '[]') as string[][];
    return tokensList.map(([field, operator, value]) => ({ field, operator, value }));
};

export const serializeTokens = (tokens: Token[]): string => {
    const tokensList = tokens.map(({ field, operator, value }) => [field, operator, value]);
    return JSON.stringify(tokensList);
};

export const makeSelectorFromTokensString = (data: Optional<string>): string => (
    makeSelectorFromConditions(parseTokensString(data))
);

export const makeBasicTokensFromSelector = (selector: string) => {
    const service = parseServiceFromSelector(selector);
    return JSON.stringify([['service', '=', service]]);
};

export const makeBasicTokensFromService = (service: string) => (
    JSON.stringify([['service', '=', service]])
);

const makeKeyAction = (key: keyof Token) => (
    ({ token, focus, onFocus, onChange, event }: TokenOnKeyDownOptions<Token>) => {
        event.preventDefault();
        if (!token.value[key]) {
            return;
        }
        const nextKey = nextSelectorConditionKey(key);
        onFocus({ ...focus, key: nextKey });
        onChange(focus.idx, { ...token.value, [nextKey]: event.key });
    }
);

export const getTokenFields = (): TokenField<Token>[] => {
    const operators = QUERY_LANGUAGE_OPERATORS;
    return [
        {
            key: 'field',
            specialKeysActions: [
                {
                    key: (e) => {
                        if (e.key === ' ') {
                            return true;
                        }
                        return operators.some(operator => operator.includes(e.key));
                    },
                    action: makeKeyAction('field'),
                },
            ],
        },
        {
            key: 'operator',
            specialKeysActions: [
                {
                    key: (e) => {
                        if (e.key.length > 1) {
                            return false;  // special keys, e.g. ArrowLeft
                        }
                        return !operators.some(operator => operator.includes(e.key));
                    },
                    action: makeKeyAction('operator'),
                },
            ],
        },
        {
            key: 'value',
        },
    ];
};

const makeSuggestItem = (key: string, value: string): TokenizedSuggestionsItem<Token> => ({
    label: value,
    search: value,
    value: {
        [key]: value,
    },
});

export const handleTokensSuggest = async (
    ctx: TokenizedInputSuggestionContext<Token>,
    handleQuerySuggest: (state: SuggestState) => Promise<Suggestions>,
) => {
    const { idx, key, tokens } = ctx;
    const currentToken = tokens[idx]?.value;

    const suggestState = {
        tokens: tokens.map(({ value }) => value),
        currentToken,
        key,
    };
    const options = await handleQuerySuggest(suggestState);

    const items = options?.map(option => ({
        ...makeSuggestItem(key, option),
        focus: {
            idx: idx + Number(key === 'value'),
            key: nextSelectorConditionKey(key),
        },
    }));

    return {
        items: items || [],
        options: {
            // `options === undefined` if suggest for the current key is not supported.
            // In that case, there is no point in showing a message about no matches.
            // But if `options === []`, we should show it, as suggest is supported and
            // we couldn't find anything that matched the entered pattern.
            showEmptyState: items !== undefined,

            // Without this option, filtering by regex like `perf.*` will show nothing.
            // However, it clearly matches values like `perfmanager`.
            // Yeah, the naming is not obvious at all.
            isFilterable: false,
        },
    };
};
