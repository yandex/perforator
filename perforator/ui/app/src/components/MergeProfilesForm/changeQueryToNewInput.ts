import { parseServiceFromSelector, validateSelectorContainsOnlyService } from 'src/utils/selector';

import type { QueryInput, QueryInputResult } from './QueryInput';
import { makeBasicTokensFromSelector } from './TokensInput/utils';


export const changeQueryToNewInput = (queryInput: QueryInput, tableSelector: string): Partial<QueryInputResult> | undefined => {
    if (queryInput.queryField === 'selector') {
        // fill selector after switching from another input mode
        return ({
            selector: tableSelector,
        });
    }
    else if (queryInput.queryField === 'tokens' && ((tableSelector && validateSelectorContainsOnlyService(tableSelector || '')))) {
        const tokens = makeBasicTokensFromSelector(tableSelector);
        return ({
            tokens,
        });
    } else if (queryInput.queryField === 'service' && tableSelector && validateSelectorContainsOnlyService(tableSelector || '')) {
        const service = parseServiceFromSelector(tableSelector);
        return ({
            service,
        });
    } else {
        return undefined;
    }
};
