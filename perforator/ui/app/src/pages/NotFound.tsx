import { NotFound as Illustration } from '@gravity-ui/illustrations';

import { ErrorPage } from 'src/components/ErrorPage/ErrorPage';

import i18n from './i18n';
import type { Page } from './Page';


export const NotFound: Page = ({ header }) => {
    return <>
        {header}
        <ErrorPage picture={Illustration} title={i18n('pageNotFound')} />
    </>;
};
