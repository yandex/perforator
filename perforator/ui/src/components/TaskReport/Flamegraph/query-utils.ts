/** adds or modifies when value is truthy, deletes query if falsy.
 * Record key is query key, record value is new query value */
export function modifyQuery(query: URLSearchParams, q: Record<string, string | false>) {
    for (const [field, value] of Object.entries(q)) {

        if (value) {
            query.set(field, value);
        } else {
            query.delete(field);
        }
    }

    return query;
}

export type GetStateFromQuery = ((name: string) => (string | undefined)) | ((name: string, defaultValue: string) => string)

export const getStateFromQueryParams: (params: URLSearchParams) => GetStateFromQuery = (params) => (name, defaultValue) => {
    if (params.has(name)) {
        return decodeURIComponent(params.get(name)!);
    } else {
        return defaultValue;
    }
};
