import { useMemo } from 'react';


export function useRegexError(searchQuery: string) {
    return useMemo(() => {
        try {
            RegExp(searchQuery);
            return null;
        } catch (error: unknown) {
            if (typeof error === 'object' && error !== null && 'message' in error && typeof error.message === 'string') {
                return error.message;
            }
            else if (typeof error === 'string') {
                return error;
            }
            else {
                return 'Unknown error in regexp';
            }
        }
    }, [searchQuery]);
}
