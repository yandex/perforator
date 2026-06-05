import React from 'react';

import { Alert } from '@gravity-ui/uikit';


export interface ErrorPanelProps {
    message: string;
    title?: string;
}

export const ErrorPanel: React.FC<ErrorPanelProps> = ({ message, title }: ErrorPanelProps) => {
    return (
        <Alert
            theme="danger"
            view="filled"
            title={title ?? 'Error'}
            message={message}
        />
    );
};
