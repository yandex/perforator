// NOTE: one day we'll get this data from backend


const EQUALITY_OPERATORS = [
    '=',
    '!=',
];
const REGEX_OPERATORS = [
    ...EQUALITY_OPERATORS,
    '=~',
    '!~',
];

export interface QueryField {
    field: string;
    operators: string[];
}

export type QueryFields = Map<string, QueryField>;

export const getQueryFields = async (): Promise<QueryField[]> => ([
    {
        field: 'service',
        operators: REGEX_OPERATORS,
    },
    {
        field: 'cluster',
        operators: REGEX_OPERATORS,
    },
    {
        field: 'pod_id',
        operators: REGEX_OPERATORS,
    },
    {
        field: 'node_id',
        operators: REGEX_OPERATORS,
    },
    {
        field: 'cpu',
        operators: REGEX_OPERATORS,
    },
    {
        field: 'event_type',
        operators: ['='],
    },
    {
        field: 'build_ids',
        operators: ['='],
    },
    {
        field: 'system_name',
        operators: REGEX_OPERATORS,
    },
    {
        field: 'id',
        operators: EQUALITY_OPERATORS,
    },
]);
