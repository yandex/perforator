import { createBrowserRouter, Outlet } from 'react-router-dom';

import type { PageComponent, PagePublicProps } from 'src/components/Page/Page';
import { PageContainer } from 'src/components/Page/PageContainer/PageContainer';
import { routes } from 'src/const/routes';
import { DemoPage } from 'src/pages/DemoPage';

import {
    BuildProfile,
    ClusterTop,
    DiffLists,
    History,
    NotFound,
    Profile,
    ProfileList,
    Task,
} from '../../pages';

import i18n from './i18n';


export const getRouter = (pageProps: PagePublicProps) => {
    const makePage = (page: PageComponent, title: Optional<string>) => (
        <PageContainer
            page={page}
            pageProps={pageProps}
            title={title}
        />
    );

    const router = createBrowserRouter([
        {
            path: routes.home,
            element: <Outlet />,
            errorElement: makePage(NotFound, i18n('notFound')),
            children: [
                {
                    index: true,
                    element: makePage(ProfileList, undefined),
                },
                {
                    path: routes.profiles,
                    element: makePage(ProfileList, i18n('profiles')),
                },
                {
                    path: routes.diff,
                    element: makePage(DiffLists, i18n('diff')),
                },
                {
                    path: routes.task,
                    element: makePage(Task, i18n('profile')),
                },
                {
                    path: routes.profile,
                    element: makePage(Profile, i18n('profile')),
                },
                {
                    path: routes.build,
                    element: makePage(BuildProfile, i18n('profile')),
                },
                {
                    path: routes.tasks,
                    element: makePage(History, i18n('history')),
                },
                {
                    path: routes.tutorialBasics,
                    element: makePage(DemoPage, i18n('demo')),
                },
                {
                    path: routes.clusterTop,
                    element: makePage(ClusterTop, i18n('clusterTop')),
                },
            ],
        },
    ]);
    return router;
};
