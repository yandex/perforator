import React from 'react';

import { InternalError } from '@gravity-ui/illustrations';

import { uiFactory } from 'src/factory';
import { buildProfileRum } from 'src/utils/buildProfileRum';

import { ErrorPage } from '../ErrorPage/ErrorPage';

import i18n from './i18n';


interface ErrorBoundaryProps {
    children?: React.ReactNode;
}

interface ErrorBoundaryState {
    hasError: boolean;
    error?: Error;
}

export class ErrorBoundary extends React.Component<ErrorBoundaryProps, ErrorBoundaryState> {
    static getDerivedStateFromError(error: Error): ErrorBoundaryState {
        return { hasError: true, error };
    }

    state: ErrorBoundaryState = {
        hasError: false,
    };

    componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
        buildProfileRum.error();
        uiFactory().logError(error, { errorInfo });
    }


    render() {
        const { error, hasError } = this.state;
        if (hasError) {
            return <ErrorPage picture={InternalError} title={error?.message ?? i18n('unknownError')} />;
        }

        return this.props.children;
    }
}
