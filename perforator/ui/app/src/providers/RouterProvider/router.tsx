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
            errorElement: makePage(NotFound, 'Not found'),
            children: [
                {
                    index: true,
                    element: makePage(ProfileList, undefined),
                },
                {
                    path: routes.profiles,
                    element: makePage(ProfileList, 'Profiles'),
                },
                {
                    path: routes.diff,
                    element: makePage(DiffLists, 'Diff'),
                },
                {
                    path: routes.task,
                    element: makePage(Task, 'Profile'),
                },
                {
                    path: routes.profile,
                    element: makePage(Profile, 'Profile'),
                },
                {
                    path: routes.build,
                    element: makePage(BuildProfile, 'Profile'),
                },
                {
                    path: routes.tasks,
                    element: makePage(History, 'History'),
                },
                {
                    path: routes.tutorialBasics,
                    element: makePage(DemoPage, 'Demo'),
                },
                {
                    path: routes.clusterTop,
                    element: makePage(ClusterTop, 'Cluster Top'),
                },
            ],
        },
    ]);
    return router;
};
