import type { monaco } from 'react-monaco-editor';

import { QUERY_LANGUAGE_OPERATORS } from 'src/providers/QuerySuggestProvider';
import {
    nextSelectorConditionKey,
    type SelectorCondition,
} from 'src/utils/selector';

import { TOKEN_FIELD_TO_MONACO_TYPE } from './const';


type Rule = monaco.languages.IMonarchLanguageRule;
type Rules = Rule[];

const COMMON_RULES: Rules = [
    [/{/, 'delimiter.open'],
    [/}/, 'delimiter.close'],
    [/\s/, 'space'],
];

const makeQuotesRules = (token: string): Rules => [
    [/\\"/, token],
    [/"/, { token, next: '@pop' }],
    [/./, token],
];

const makeRules = ({
    key,
    matchingRegex,
    nextRegex,
    nextToken,
    quotes = true,
}: {
    key: keyof SelectorCondition;
nextToken: string;
    matchingRegex?: RegExp;
    nextRegex?: RegExp;
    quotes?: boolean;
}): {[key: string]: Rules} => {
    const token = TOKEN_FIELD_TO_MONACO_TYPE[key]!;
    const quotesState = `${key}InQuotes`;
    const nextState = `@${nextSelectorConditionKey(key)}`;
    const onNext = { token: nextToken, next: nextState };
    const rules: Rules = [
        ...COMMON_RULES,
        ...(quotes
            ? [
                [/"/, { token, next: quotesState }],
            ] : []
        ) as Rules,
        ...(nextRegex
            ? [
                [nextRegex, onNext],
                [/./, { token }],
            ] : []
        ) as Rules,
        ...(matchingRegex
            ? [
                [matchingRegex, { token }],
                [/./, onNext],
            ] : []
        ) as Rules,
    ];
    return {
        [key]: rules,
        [quotesState]: makeQuotesRules(token),
    };
};

export const setupTokens = (): monaco.languages.IMonarchLanguage => {
    const operatorsCharacters = [
        ...new Set(QUERY_LANGUAGE_OPERATORS.map(operator => Array.from(operator)).flat()),
    ];
    const operatorsRegex = new RegExp(operatorsCharacters.map(ch => `\\${ch}`).join('|'));

    return {
        operators: operatorsCharacters,
        tokenizer: {
            root: [
                [/./, { token: '@rematch', next: '@field' }],
            ],
            ...makeRules({
                key: 'field',
                nextRegex: operatorsRegex,
                nextToken: '@rematch',
            }),
            ...makeRules({
                key: 'operator',
                matchingRegex: operatorsRegex,
                nextToken: '@rematch',
                quotes: false,
            }),
            ...makeRules({
                key: 'value',
                nextRegex: /,/,
                nextToken: 'comma',
            }),
        },
    };
};
