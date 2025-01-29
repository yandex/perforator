import { NotFound as Illustration } from '@gravity-ui/illustrations';

import { ErrorPage } from 'src/components/ErrorPage/ErrorPage';

import type { Page } from './Page';


export const NotFound: Page = () => {
    return <ErrorPage picture={Illustration} title="Page not found" />;
};
