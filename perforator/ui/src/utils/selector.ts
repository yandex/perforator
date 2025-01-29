import type { ProfileTaskQuery } from 'src/models/Task';

import { getIsoDate } from './date.ts';


export const EMPTY_SELECTOR = '{}';

export type SelectorCondition = {
    field?: string;
    operator?: string;
    value?: string;
}

export const makeSelectorFromConditions = (conditions: SelectorCondition[]): string => {
    const conditionStrings = conditions.map(condition => (
        `${condition.field}${condition.operator ?? '='}"${condition.value}"`
    ));
    return `{${conditionStrings.join(', ')}}`;
};

const makeTimeConditions = (query: ProfileTaskQuery): SelectorCondition[] => ([
    { field: 'timestamp', operator: '>=', value: getIsoDate(query.from)! },
    { field: 'timestamp', operator: '<=', value: getIsoDate(query.to)! },
]);

// both baseline and diff think they're the baseline
export function composeDiffQuery(
    baseline: ProfileTaskQuery,
    diff: ProfileTaskQuery): ProfileTaskQuery {
    return {
        ...baseline,
        diffSelector: makeSelector(diff),
    };
}

export const makeSelector = (query: ProfileTaskQuery): string => {
    if (query.selector) {
        if (!query.selector.includes('timestamp')) {
            // NOTE: what if service name contains 'timestamp' as a substring?
            // I believe it'll be better with the tokenized input component
            const comma = ', ';
            const timeSelector = makeSelectorFromConditions(makeTimeConditions(query))
                .replace('{', comma);
            return query.selector
                .replace(/}$/, timeSelector)
                .replace('{' + comma, '{');
        }
        return query.selector;
    }
    const conditions = makeTimeConditions(query);
    if (query.service) {
        conditions.push({ field: 'service', value: query.service });
    }
    if (query.profileId) {
        conditions.push({ field: 'id', value: query.profileId });
    }
    return makeSelectorFromConditions(conditions);
};


const timestampRg = 'timestamp(>=|<=)"(.*?)"';
const timestampRegex = new RegExp(timestampRg, 'g');
/** timestamp is wellknown field */
export function parseTimestampFromSelector(selector: string) {
    const matches = selector.matchAll(timestampRegex);
    const res: {from?: string; to?: string} = {};
    for (const match of matches) {
        if (match[1] === '>=') {
            res.from = match[2];
        }
        if (match[1] === '<=') {
            res.to = match[2];
        }
    }
    return res;
}

const timestampCutRegex = new RegExp(`((${timestampRg})(,\\s*)?)|((,\\s*)?${timestampRg})`, 'g');

export function cutTimeFromSelector(selector: string): string {
    return selector.replace(timestampCutRegex, '');
}

export const SELECTOR_CONDITION_KEYS: (keyof SelectorCondition)[] = ['field', 'operator', 'value'];

// we need to iterate over token keys for suggest
export const nextSelectorConditionKey = (current: keyof SelectorCondition) => (
    SELECTOR_CONDITION_KEYS[
        (SELECTOR_CONDITION_KEYS.indexOf(current) + 1) % SELECTOR_CONDITION_KEYS.length
    ]
);
