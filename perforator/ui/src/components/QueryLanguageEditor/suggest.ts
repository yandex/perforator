import type { monaco } from 'react-monaco-editor';

import type {
    Suggestions,
    SuggestState,
} from 'src/providers/QuerySuggestProvider';
import {
    nextSelectorConditionKey,
    type SelectorCondition as Token,
} from 'src/utils/selector';

import {
    makeMonacoTokenType,
    MONACO_TYPE_TO_TOKEN_FIELD,
    QUERY_LANGUAGE_ID,
} from './const';
import { getCommandId } from './utils';


type Word = [number, number];

const tokenizeInput = (
    monacoInstance: typeof monaco,
    text: string,
    cursorIndex: number,
): [SuggestState, Word] => {
    // First of all, we tokenize the text based on the rules from `tokens.ts`.
    // We take only the first element of the result as we only have one line.
    const monacoTokens = monacoInstance.editor.tokenize(text, QUERY_LANGUAGE_ID)[0] ?? [];

    const tokens: Token[] = [{}];
    let currentTokenKey: keyof Token = 'field';
    const lastToken = () => tokens[tokens.length - 1];

    let tokenUnderCursor = lastToken();
    let tokenKeyUnderCursor: keyof Token = currentTokenKey;
    let wordUnderCursor: Word = [0, 0];

    for (let i = 0; i < monacoTokens.length; i++) {
        const currentMonacoToken = monacoTokens[i];
        const currentMonacoTokenStart = currentMonacoToken.offset;
        const currentMonacoTokenEnd = monacoTokens[i + 1]?.offset ?? text.length;

        const tokenKey = MONACO_TYPE_TO_TOKEN_FIELD[currentMonacoToken.type];
        const value = text.slice(currentMonacoTokenStart, currentMonacoTokenEnd);

        if (tokenKey !== undefined) {
            lastToken()[tokenKey] = value.replace(/^"/, '').replace(/"$/, '');
            currentTokenKey = nextSelectorConditionKey(tokenKey);
        } else if (currentMonacoToken.type === makeMonacoTokenType('comma')) {
            // finally, a new token
            tokens.push({});
        }

        if (currentMonacoTokenStart < cursorIndex && cursorIndex <= currentMonacoTokenEnd) {
            tokenUnderCursor = lastToken();
            tokenKeyUnderCursor = tokenKey ?? currentTokenKey;
            wordUnderCursor = [
                // don't replace spaces with inserted text
                tokenKey !== undefined ? currentMonacoTokenStart : currentMonacoTokenEnd,
                currentMonacoTokenEnd,
            ];
        }
    }

    const suggestState = {
        tokens,
        currentToken: tokenUnderCursor,
        key: tokenKeyUnderCursor,
    };
    return [suggestState, wordUnderCursor];
};

const makeRange = (word: Word) => ({
    startLineNumber: 1,
    endLineNumber: 1,
    startColumn: word[0] + 1,
    endColumn: word[1] + 1,
});

const textToInsert = (tokenKey: keyof Token, value: string) => {
    if (tokenKey === 'value') {
        return `"${value}", `;
    } else if (tokenKey === 'operator') {
        return value + ' "$0"';  // place cursor between quotes
    }
    return value + ' ';
};

const getEditor = (
    monacoInstance: typeof monaco,
    model: monaco.editor.ITextModel,
) => {
    // there can be several instances of monaco editor,
    // so we choose the one the triggered the suggest
    for (const editor of monacoInstance.editor.getEditors()) {
        if (editor.getModel() === model) {
            return editor;
        }
    }
    return undefined;
};

export const setupSuggest = (
    monacoInstance: typeof monaco,
    handleQuerySuggest: (state: SuggestState) => Promise<Suggestions>,
) => (
    async (
        model: monaco.editor.ITextModel,
        position: monaco.Position,
    ): Promise<Optional<monaco.languages.CompletionList>> => {
        const text = model.getValue();

        const [suggestState, wordUnderCursor] = tokenizeInput(
            monacoInstance,
            text,
            position.column - 1,
        );
        const tokenKey = suggestState.key;

        const editor = getEditor(monacoInstance, model);

        const range = makeRange(wordUnderCursor);
        const makeSuggestion = (value: string, index: number) => ({
            label: value,
            insertText: textToInsert(tokenKey, value),
            kind: monacoInstance.languages.CompletionItemKind.Text,
            range,

            // to change cursor position
            insertTextRules: monacoInstance.languages.CompletionItemInsertTextRule.InsertAsSnippet,

            // https://github.com/microsoft/monaco-editor/issues/1889#issuecomment-642809145
            filterText: text.slice(range.startColumn - 1, range.endColumn),

            // for custom order of options
            sortText: String.fromCharCode(index),

            command: {
                title: '',
                id: tokenKey === 'value'
                    ? getCommandId(editor, 'afterSuggest')  // executes `onSuggest` callback
                    : 'editor.action.triggerSuggest',
            },
        });

        const options = await handleQuerySuggest(suggestState);
        const suggestions = options?.map((option, index) => makeSuggestion(option, index));

        if (suggestions === undefined) {
            editor?.trigger('hideSuggest', 'hideSuggestWidget', null);
            return undefined;
        }
        return { suggestions };
    }
);
