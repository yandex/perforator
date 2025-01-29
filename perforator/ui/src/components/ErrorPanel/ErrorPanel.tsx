import React from 'react';

import { Alert } from '@gravity-ui/uikit';


export interface ErrorPanelProps {
    message: string;
    title?: string;
}

export const ErrorPanel: React.FC<ErrorPanelProps> = props => {
    return (
        <Alert
            theme="danger"
            view="filled"
            title={props.title ?? 'Error'}
            message={props.message}
        />
    );
};
