import { NotFound as Illustration } from '@gravity-ui/illustrations';

import { ErrorPage } from 'src/components/ErrorPage/ErrorPage';

import type { Page } from './Page';


export const NotFound: Page = ({ header }) => {
    return <>
        {header}
        <ErrorPage picture={Illustration} title="Page not found" />
    </>;
};
