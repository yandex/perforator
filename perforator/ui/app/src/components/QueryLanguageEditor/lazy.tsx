import React from 'react';


export const QueryLanguageEditorImpl = React.lazy(() => import('./QueryLanguageEditor').then(i => ({ default: i.QueryLanguageEditorImpl })));

export const QueryLanguageFallback: React.FC<{selector?: string; className?: string}> = ({ selector, className }) => (
    <code className={'selector-input__skeleton' + (className ? ' ' + className : '')}>{selector}</code>
);
