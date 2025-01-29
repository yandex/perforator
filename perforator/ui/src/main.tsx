import React from 'react';

import ReactDOM from 'react-dom/client';

import { buildFactory, uiFactory } from 'src/factory';

import { App } from './components/App/App';

import './utils/rum';

import '@gravity-ui/uikit/styles/styles.scss';
import './styles/base.scss';


buildFactory().then(() => {
    uiFactory().configureApp();
    ReactDOM.createRoot(document.getElementById('root')!, {
        onRecoverableError: (error, errorInfo) => {
            uiFactory().logError(error, { errorInfo });
        },
    }).render(
        <React.StrictMode>
            <App />
        </React.StrictMode>,
    );
});
