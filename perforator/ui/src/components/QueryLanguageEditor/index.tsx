import React from 'react';

import { Loader } from '@gravity-ui/uikit';

import { QuerySuggestProvider } from 'src/providers/QuerySuggestProvider';

import type { QueryLanguageEditorProps } from './QueryLanguageEditor';


const QueryLanguageEditorImpl = React.lazy(() => import('./QueryLanguageEditor').then(i => ({ default: i.QueryLanguageEditorImpl })));

export const QueryLanguageEditor: React.FC<QueryLanguageEditorProps> = props => (
    <QuerySuggestProvider>
        <React.Suspense fallback={<Loader className="selector-input__loader"/>}>
            <QueryLanguageEditorImpl {...props} />
        </React.Suspense>
    </QuerySuggestProvider>
);

export type { QueryLanguageEditorProps } from './QueryLanguageEditor';
