import type { QueryInput, QueryInputRenderer } from 'src/components/MergeProfilesForm/QueryInput';
import { QuerySuggestProvider } from 'src/providers/QuerySuggestProvider';

import { TokensInput } from './TokensInput';
import { makeSelectorFromTokensString } from './utils';


const renderTokensInput: QueryInputRenderer = (query, setQuery, setTableSelector) => (
    <QuerySuggestProvider>
        <TokensInput
            initialTokens={query.tokens}
            onUpdate={tokens => {
                if (tokens) {
                    if (setTableSelector) {
                        setTableSelector(makeSelectorFromTokensString(tokens));
                    }
                    setQuery(currentQuery => ({
                        ...currentQuery,
                        tokens,
                    }));
                }
            }} />
    </QuerySuggestProvider>
);

export const TOKENS_QUERY_INPUT: QueryInput = {
    name: 'Tokens',
    queryField: 'tokens',
    render: renderTokensInput,
};
