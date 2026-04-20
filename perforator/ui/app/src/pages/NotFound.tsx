import { NotFound as Illustration } from '@gravity-ui/illustrations';

import { ErrorPage } from 'src/components/ErrorPage/ErrorPage';

import type { Page } from './Page';


export const NotFound: Page = (props) => {
    return <>
        {props.header}
        <ErrorPage picture={Illustration} title="Page not found" />
    </>;
};
