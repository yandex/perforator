import type { monaco } from 'react-monaco-editor';


export const getCommandId = (
    editor: Optional<monaco.editor.ICodeEditor>,
    command: string,
) => (
    `${editor?.getId()}.${command}`
);
