import React from 'react';

import { PageLayout } from '@gravity-ui/navigation';
import { Container } from '@gravity-ui/uikit';

import { ErrorBoundary } from 'src/components/ErrorBoundary/ErrorBoundary';
import { LocalStorageKey } from 'src/const/localStorage';
import { cn } from 'src/utils/cn';
import { setPageTitle } from 'src/utils/title';

import { NavigationBar } from '../../NavigationBar/NavigationBar';
import type { PageComponent, PageProps } from '../Page';
import { PageFooter } from '../PageFooter/PageFooter';
import { PageHeading } from '../PageHeading/PageHeading';

import './PageContainer.scss';


const b = cn('page-container');

export type PageContainerProps = {
    page: PageComponent;
    pageProps: PageProps;
    title?: string;
};

export const PageContainer: React.FC<PageContainerProps> = props => {
    const { pageProps } = props;

    const [compact, setCompact] = React.useState(
        localStorage.getItem(LocalStorageKey.AsideHeaderCompact) !== 'false',
    );

    const handleCompactChange = React.useCallback((value: boolean) => {
        setCompact(value);
        localStorage.setItem(LocalStorageKey.AsideHeaderCompact, value.toString());
    }, []);

    React.useMemo(() => setPageTitle(props.title), [props.title]);

    const { embed } = pageProps;
    const className = b({ embed });
    const pageClassName = cn('page')({ embed });
    const navigation = embed ? null : (
        <NavigationBar
            compact={false}
            setCompact={handleCompactChange}
        />
    );

    return (
        <ErrorBoundary>
            <PageLayout compact={compact} className={className}>
                {navigation}
                <PageLayout.Content>
                    <Container className={pageClassName}>
                        <PageHeading embed={embed} />
                        {React.createElement(props.page, pageProps)}
                        {embed ? null : <PageFooter />}
                    </Container>
                </PageLayout.Content>
            </PageLayout>
        </ErrorBoundary>
    );
};
