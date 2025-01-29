import React, { Suspense } from 'react';

import { AsideFallback } from '@gravity-ui/navigation';

import { uiFactory } from 'src/factory';

import type { AsideProps } from './Aside/AsideProps';


const AsideComponent = React.lazy(() =>
    import('./Aside/Aside').then(({ Aside }) => ({ default: Aside })),
);

export interface NavigationBarProps extends AsideProps {}

export const NavigationBar: React.FC<NavigationBarProps> = (props) => {
    return (
        <Suspense
            fallback={
                <AsideFallback
                    headerDecoration
                    subheaderItemsCount={uiFactory().subheaderItemsCount()}
                />
            }
        >
            <AsideComponent {...props} />
        </Suspense>
    );
};
