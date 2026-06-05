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
    tokens?: Optional<string>;
    onUpdate: (tokens: Optional<string>) => void;
}

export const TokensInput: React.FC<TokensInputProps> = ({ initialTokens, tokens, onUpdate }: TokensInputProps) => {
    const [tokenList, setTokenList] = React.useState<Token[]>(parseTokensString(initialTokens));

    React.useEffect(() => {
        setTokenList(parseTokensString(initialTokens));
    }, [initialTokens]);

    React.useEffect(() => onUpdate(initialTokens), []);
    React.useEffect(() => {
        if (tokens) {
            setTokenList(parseTokensString(tokens));
        }
    }, [tokens]);

    const handleChange = (value: Token[]) => {
        setTokenList(value);
        onUpdate(serializeTokens(value));
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
            tokens={tokens ? parseTokensString(tokens) : tokenList}
            onChange={handleChange}
            validateToken={validateToken}
            fields={tokenFields}
            onSuggest={handleSuggest}
        />
    );
};
