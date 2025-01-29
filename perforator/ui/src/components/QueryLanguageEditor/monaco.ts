import type { monaco } from 'react-monaco-editor';

import type {
    Suggestions,
    SuggestState,
} from 'src/providers/QuerySuggestProvider';

import { QUERY_LANGUAGE_ID } from './const';
import { setupSuggest } from './suggest';
import { setupTokens } from './tokens';

import './imports';


export { getEditorOptions } from './options';


export { QUERY_LANGUAGE_ID };


export const registerLanguage = (
    monacoInstance: typeof monaco,
    handleQuerySuggest: (state: SuggestState) => Promise<Suggestions>,
) => {
    if (monacoInstance.languages.getLanguages().some(({ id }) => id === QUERY_LANGUAGE_ID)) {
        // the language is already registered
        return;
    }
    monacoInstance.languages.register({ id: QUERY_LANGUAGE_ID });
    monacoInstance.languages.setLanguageConfiguration(QUERY_LANGUAGE_ID, {
        autoClosingPairs: [
            { open: '{', close: '}' },
            { open: '"', close: '"' },
            { open: '\'', close: '\'' },
        ],
    });
    monacoInstance.languages.setMonarchTokensProvider(QUERY_LANGUAGE_ID, setupTokens());
    monacoInstance.languages.registerCompletionItemProvider(QUERY_LANGUAGE_ID, {
        provideCompletionItems: setupSuggest(monacoInstance, handleQuerySuggest),
    });
};
