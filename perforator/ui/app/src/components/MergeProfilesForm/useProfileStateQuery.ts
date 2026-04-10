import React from 'react';

import { useSearchParams } from 'react-router-dom';

import {
    defaultProfileTaskQuery,
    taskQueryToSearchParams,
} from 'src/utils/profileTask';
import {
    cutTimeFromSelector,
    parseTimestampFromSelector,
} from 'src/utils/selector';

import type { QueryInputResult } from './QueryInput';


function isNumber(n: unknown) {
    const num = Number(n);
    return !Number.isNaN(num);
}

export function useProfileStateQuery({
    inMemory,
}: { inMemory?: boolean } = {}) {
    const defaultQuery = defaultProfileTaskQuery();

    const initialSearchParams = new URLSearchParams(
        document.location.search,
    );
    const selector = initialSearchParams.get('selector');
    const maxProfiles = initialSearchParams.get('maxProfiles');
    const from = initialSearchParams.get('from');
    const to = initialSearchParams.get('to');

    if (maxProfiles && isNumber(maxProfiles)) {
        defaultQuery.maxProfiles = Number(maxProfiles);
    }

    if (from) {
        defaultQuery.from = from;
    }

    if (to) {
        defaultQuery.to = to;
    }

    if (selector && !from && !to && !inMemory) {
        const { from: parsedFrom, to: parsedTo } = parseTimestampFromSelector(selector);
        if (parsedFrom && parsedTo) {
            defaultQuery.from = parsedFrom;
            defaultQuery.to = parsedTo;
            defaultQuery.selector = cutTimeFromSelector(selector);
        }
    }

    const defaultQueryParams = taskQueryToSearchParams(defaultQuery);
    const [searchParams, setSearchParams] = useSearchParams(
        inMemory ? {} : defaultQueryParams,
    );

    const params: unknown = Object.fromEntries(searchParams);
    const [query, setQueryState] = React.useState<QueryInputResult>(
        inMemory ? defaultQuery : (params as QueryInputResult),
    );

    const setQuery = (newQuery: QueryInputResult) => {
        if (!inMemory) {
            setSearchParams(taskQueryToSearchParams(newQuery));
        }
        setQueryState(newQuery);
    };

    return [query, setQuery] as const;
}
