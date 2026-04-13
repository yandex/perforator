import React from 'react';

import { TokenizedInput, type TokenizedInputSuggestionContext } from '@gravity-ui/components';

import { useQuerySuggest } from 'src/providers/QuerySuggestProvider';

import {
    getTokenFields,
    handleTokensSuggest,
    parseTokensString,
    serializeTokens,
    type Token,
} from './utils';


export interface TokensInputProps {
    initialTokens: Optional<string>;
    onUpdate: (tokens: Optional<string>) => void;
}

export const TokensInput: React.FC<TokensInputProps> = props => {
    const [tokens, setTokens] = React.useState<Token[]>(parseTokensString(props.initialTokens));

    React.useEffect(() => props.onUpdate(props.initialTokens), []);

    const handleChange = (value: Token[]) => {
        setTokens(value);
        props.onUpdate(serializeTokens(value));
    };

    const validateToken = () => undefined;  // otherwise empty values get marked as errors

    const { handleQuerySuggest } = useQuerySuggest();

    const tokenFields = React.useMemo(() => getTokenFields(), []);

    const handleSuggest = React.useCallback(
        async (ctx: TokenizedInputSuggestionContext<Token>) => handleTokensSuggest(ctx, handleQuerySuggest),
        [handleQuerySuggest],
    );

    return (
        <TokenizedInput
            tokens={tokens}
            onChange={handleChange}
            validateToken={validateToken}
            fields={tokenFields}
            onSuggest={handleSuggest}
        />
    );
};
