import React from 'react';

import { BrowserRouter } from 'react-router-dom';

import { ThemeProvider, useThemeType } from '@gravity-ui/uikit';


import { Visualisation } from '../Visualisation/Visualisation';

import '@gravity-ui/uikit/styles/styles.css';

import './Viewer.css';

const data = window.__data__;

const VisualisationImpl = () => {
    const type = useThemeType();
    return <Visualisation profileData={data} loading={false} theme={type} />;
};

export const ViewerApp: React.FC<{}> = () => {
    return (
            <ThemeProvider theme={'system'}>
                <BrowserRouter>
                        <VisualisationImpl/>
                </BrowserRouter>
            </ThemeProvider>
    );
};
