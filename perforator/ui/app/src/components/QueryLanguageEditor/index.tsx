import React from 'react';

import { QuerySuggestProvider } from 'src/providers/QuerySuggestProvider';

import { QueryLanguageEditorImpl, QueryLanguageFallback } from './lazy';
import type { QueryLanguageEditorProps } from './QueryLanguageEditor';

import './QueryLanguageEditor.scss';


export const QueryLanguageEditor: React.FC<QueryLanguageEditorProps> = props => (
    <QuerySuggestProvider>
        <React.Suspense fallback={<QueryLanguageFallback className={props.className} selector={props.selector}/>}>
            <QueryLanguageEditorImpl {...props} />
        </React.Suspense>
    </QuerySuggestProvider>
);

export type { QueryLanguageEditorProps } from './QueryLanguageEditor';
export { QueryLanguageHelpPopover } from './QueryLanguageHelpPopover';
