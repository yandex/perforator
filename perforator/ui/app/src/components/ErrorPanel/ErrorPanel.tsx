import React from 'react';

import { Alert } from '@gravity-ui/uikit';

import i18n from './i18n';


export interface ErrorPanelProps {
    message: string;
    title?: string;
}

export const ErrorPanel: React.FC<ErrorPanelProps> = ({ message, title }: ErrorPanelProps) => {
    return (
        <Alert
            theme="danger"
            view="filled"
            title={title ?? i18n('error')}
            message={message}
        />
    );
};
