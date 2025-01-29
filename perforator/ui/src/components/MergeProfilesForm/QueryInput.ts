import type React from 'react';

import type { ProfileTaskQuery } from 'src/models/Task';


export type QueryInputResult = ProfileTaskQuery & {
    tokens?: string;
};

export type QueryInputRenderer = (
    query: QueryInputResult,
    setQuery: (query: QueryInputResult) => void,
    setTableSelector?: (selector: string) => void,
) => React.ReactNode;

export interface QueryInput {
    name: string;
    queryField: string;
    render: QueryInputRenderer;
    beta?: boolean;
}
