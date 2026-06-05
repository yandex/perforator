import React from 'react';

import { Loader } from '@gravity-ui/uikit';

import type { Page } from './Page';


const Demo = React.lazy(() => import('src/components/Demo/Demo').then((imported) => ({ default: imported.Demo })));

export const DemoPage: Page = ({ header }) => {
    return <>
        {header}
        <React.Suspense fallback={<Loader/>}><Demo /></React.Suspense>
    </>;
};
