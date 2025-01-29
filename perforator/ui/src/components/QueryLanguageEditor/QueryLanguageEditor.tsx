import React from 'react';

import MonacoEditor, { monaco } from 'react-monaco-editor';

import { CircleQuestion } from '@gravity-ui/icons';
import {
    Button,
    Icon,
    useThemeType,
} from '@gravity-ui/uikit';

import { uiFactory } from 'src/factory';
import { useQuerySuggest } from 'src/providers/QuerySuggestProvider';
import { cn } from 'src/utils/cn';
import { useDebounce } from 'src/utils/debounce';
import { EMPTY_SELECTOR } from 'src/utils/selector';

import { getEditorOptions, QUERY_LANGUAGE_ID, registerLanguage } from './monaco';
import { getCommandId } from './utils';

import './QueryLanguageEditor.scss';


const b = cn('query-language-editor');

const DELIMITERS = [
    'BracketLeft',
    'BracketRight',
    'Comma',
];

const removeNewlines = (value: string) => value.replace(/\n/g, '');

export interface QueryLanguageEditorProps {
    selector?: string;
    onUpdate: (selector: Optional<string>) => void;
    onSelectorChange?: (selector: Optional<string>) => void;
    height: string;
}

export const QueryLanguageEditorImpl: React.FC<QueryLanguageEditorProps> = props => {
    const editorOptions = React.useMemo(() => getEditorOptions(), []);

    const { handleQuerySuggest } = useQuerySuggest();

    React.useEffect(() => {
        if (props.selector === undefined) {
            props.onUpdate(EMPTY_SELECTOR);
        }
    }, [props]);

    const theme = useThemeType();

    const selector = removeNewlines(props.selector ?? EMPTY_SELECTOR);

    const handleWillMount = React.useCallback((monacoInstance: typeof monaco) => {
        registerLanguage(
            monacoInstance,
            handleQuerySuggest,
        );
    }, [handleQuerySuggest]);

    const debounce = useDebounce();

    const handleDidMount = React.useCallback((editor: monaco.editor.IStandaloneCodeEditor) => {
        const onSelectorChange = () => props.onSelectorChange?.(editor.getValue());

        const suggestController = (): any => editor.getContribution('editor.contrib.suggestController');
        const suggestVisible = () => Boolean(
            suggestController()?.model?.state,
        );

        // if user stopped typing for several seconds, show profiles
        // matching the current selector
        const debounceSelectorChange = () =>
            debounce(() => {
                if (!suggestVisible()) {
                    onSelectorChange();
                }
            }, 2000);

        editor.onKeyDown(ev => {
            // for single-line mode
            if (ev.code === 'Enter') {
                ev.preventDefault();
                if (!suggestVisible()) {
                    // show matching profiles if user pressed enter outside of suggest
                    onSelectorChange();
                }
            }
            if (DELIMITERS.includes(ev.code)) {
                // show matching profiles on comma
                onSelectorChange();
            }
            debounceSelectorChange();
        });

        const triggerSuggest = () => {
            // we need to close previous suggestions first
            suggestController()?.cancelSuggestWidget?.();

            // display suggest widget first
            // this way the user immediately sees “Loading…” message
            suggestController()?.model?.trigger?.({});

            suggestController()?.triggerSuggest?.();
        };

        monaco.editor.registerCommand(
            getCommandId(editor, 'afterSuggest'),
            () => {
                onSelectorChange();
                triggerSuggest();
            },
        );

        editor.onDidChangeModelContent(() => {
            const value = editor.getValue();
            const valueWithoutNewlines = removeNewlines(value);
            if (value !== valueWithoutNewlines) {
                editor.setValue(valueWithoutNewlines);
            }
            // suggest will be triggered by every keystroke
            triggerSuggest();
        });

        editor.onMouseDown(event => {
            if (event.target.type === monaco.editor.MouseTargetType.CONTENT_TEXT) {
                // suggest will be triggered by click as well
                triggerSuggest();
            }
            if (editor.getValue() === '{}') {
                // less clicks for a user to see the suggestions
                editor.setPosition({ lineNumber: 1, column: 2 });
                triggerSuggest();
            }
            debounceSelectorChange();
        });

        editor.onDidBlurEditorWidget(() => {
            onSelectorChange();
        });
    }, []);

    return (
        <div className={b(null)}>
            <div className={b('wrapper')}>
                <MonacoEditor
                    language={QUERY_LANGUAGE_ID}
                    value={selector}
                    onChange={props.onUpdate}
                    height={props.height}
                    options={editorOptions}
                    theme={theme === 'light' ? 'light' : 'vs-dark'}
                    editorWillMount={handleWillMount}
                    editorDidMount={handleDidMount}
                />
            </div>
            <Button
                view='flat'
                href={uiFactory().queryLanguageDocsLink()}
                target="_blank"
                title="Query language docs"
            >
                <Icon data={CircleQuestion} />
            </Button>
        </div>
    );
};
