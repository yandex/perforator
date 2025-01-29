import type { monaco } from 'react-monaco-editor';


export const getEditorOptions = (): monaco.editor.IStandaloneEditorConstructionOptions => ({
    automaticLayout: true,
    fixedOverflowWidgets: true,
    folding: false,
    fontFamily: 'var(--g-font-family-monospace)',
    fontSize: 14,
    lineDecorationsWidth: 0,
    lineHeight: 18,
    lineNumbers: 'off',
    minimap: { enabled: false },
    overviewRulerLanes: 0,
    renderLineHighlight: 'none',
    scrollBeyondLastColumn: 0,
    scrollBeyondLastLine: false,
    scrollbar: { horizontal: 'hidden' },
    suggest: { showIcons: false },
    wordBasedSuggestions: 'off',
});
