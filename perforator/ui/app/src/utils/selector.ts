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

export const makeTimeConditions = (query: Pick<ProfileTaskQuery, 'from' | 'to'>): SelectorCondition[] => ([
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
    if (query.cluster) {
        conditions.push({ field: 'cluster', value: query.cluster });
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

export function cutSpaceFromSelector(selector: string): string {
    let result = '';
    let isInsideQuotedValue = false;
    let isEscaped = false;

    for (const char of selector) {
        if (isEscaped) {
            result += char;
            isEscaped = false;
            continue;
        }

        if (char === '\\' && isInsideQuotedValue) {
            result += char;
            isEscaped = true;
            continue;
        }

        if (char === '"') {
            isInsideQuotedValue = !isInsideQuotedValue;
            result += char;
            continue;
        }

        if (!isInsideQuotedValue && /\s/.test(char)) {
            continue;
        }

        result += char;
    }

    return result;
}

export function cutTimeFromSelector(selector: string): string {
    return selector.replace(timestampCutRegex, '');
}

function splitSelectorConditions(selector: string): string[] {
    const trimmedSelector = selector.trim();
    const innerSelector = trimmedSelector.startsWith('{') && trimmedSelector.endsWith('}')
        ? trimmedSelector.slice(1, -1)
        : trimmedSelector;
    const conditions: string[] = [];
    let conditionStart = 0;
    let isInsideQuotedValue = false;
    let isEscaped = false;

    for (let index = 0; index < innerSelector.length; index += 1) {
        const char = innerSelector[index];

        if (isEscaped) {
            isEscaped = false;
            continue;
        }

        if (char === '\\' && isInsideQuotedValue) {
            isEscaped = true;
            continue;
        }

        if (char === '"') {
            isInsideQuotedValue = !isInsideQuotedValue;
            continue;
        }

        if (char === ',' && !isInsideQuotedValue) {
            const condition = innerSelector.slice(conditionStart, index).trim();
            if (condition) {
                conditions.push(condition);
            }
            conditionStart = index + 1;
        }
    }

    const condition = innerSelector.slice(conditionStart).trim();
    if (condition) {
        conditions.push(condition);
    }

    return conditions;
}

function isProfileIdCondition(condition: string): boolean {
    return /^id\s*=\s*"([^"\\]|\\.)*"$/.test(condition);
}

export function cutIdFromSelector(selector: string): string {
    const conditions = splitSelectorConditions(selector)
        .filter(condition => !isProfileIdCondition(condition));

    return `{${conditions.join(',')}}`;
}

export function insertStatementIntoSelector(selector: string, statement: string): string {
    if (selector.length > 0 && selector[selector.length - 1] === '}') {
        const clearedSelector = removeOptionalTailingComma(selector);
        return clearedSelector.slice(0, -1) + ',' + statement + '}';
    }
    return '';
}

export const SELECTOR_CONDITION_KEYS: (keyof SelectorCondition)[] = ['field', 'operator', 'value'];

// we need to iterate over token keys for suggest
export const nextSelectorConditionKey = (current: keyof SelectorCondition) => (
    SELECTOR_CONDITION_KEYS[
        (SELECTOR_CONDITION_KEYS.indexOf(current) + 1) % SELECTOR_CONDITION_KEYS.length
    ]
);

export const removeOptionalTailingComma = (selector: string) => {
    return selector.replace(/,\s*}\s*$/, '}');
};

const serviceRg = /service\s*=\s*"(.*?)"/;
export function parseServiceFromSelector(selector: string) {
    const clearedSelector = removeOptionalTailingComma(selector);
    const matches = clearedSelector.match(serviceRg);
    return matches ? matches[1] : undefined;
}

export function validateSelectorContainsOnlyService(selector: string): boolean {
    const clearedSelector = removeOptionalTailingComma(selector);
    const maybeEmptySelector = clearedSelector.replace(serviceRg, '').replace(/\s+/, '');
    return maybeEmptySelector === EMPTY_SELECTOR;
}
