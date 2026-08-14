import React, { useEffect, useState } from 'react';

import { BrowserRouter } from 'react-router-dom';

import { ThemeProvider, useThemeType } from '@gravity-ui/uikit';

import { Visualisation } from '../Visualisation/Visualisation';
import { base64toUint8Array } from '../../utils/base64';
import { decompressData } from '../../utils/decompressData';

import './Viewer.css';

const VisualisationImpl = () => {
    const type = useThemeType();
    const [data, setData] = useState<any>(null);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        const loadData = async () => {
            try {
                const rawData = window.__data__;
                const bytes = base64toUint8Array(rawData);
                const decompressed = await decompressData(bytes);
                const parsed = JSON.parse(decompressed);
                setData(parsed);
            } catch (error) {
                console.error('Failed to load data:', error);
            } finally {
                setLoading(false);
            }
        };

        loadData();
    }, []);

    if (!data) {
        return null;
    }

    return <Visualisation profileData={data} loading={loading} theme={type} />;
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
