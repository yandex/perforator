import { createBrowserRouter, Outlet } from 'react-router-dom';

import type { PageComponent, PageProps } from 'src/components/Page/Page';
import { PageContainer } from 'src/components/Page/PageContainer/PageContainer';

import {
    BuildProfile,
    DiffLists,
    History,
    NotFound,
    Profile,
    ProfileList,
    Task,
} from '../../pages';


export const getRouter = (pageProps: PageProps) => {
    const makePage = (page: PageComponent, title: Optional<string>) => (
        <PageContainer
            page={page}
            pageProps={pageProps}
            title={title}
        />
    );

    return createBrowserRouter([
        {
            path: '/',
            element: <Outlet />,
            errorElement: makePage(NotFound, 'Not found'),
            children: [
                {
                    index: true,
                    element: makePage(ProfileList, undefined),
                },
                {
                    path: 'profiles',
                    element: makePage(ProfileList, 'Profiles'),
                },
                {
                    path: 'diff',
                    element: makePage(DiffLists, 'Diff'),
                },
                {
                    path: 'task/:taskId',
                    element: makePage(Task, 'Profile'),
                },
                {
                    path: 'profile/:profileId',
                    element: makePage(Profile, 'Profile'),
                },
                {
                    path: 'build',
                    element: makePage(BuildProfile, 'Profile'),
                },
                {
                    path: 'tasks',
                    element: makePage(History, 'History'),
                },
            ],
        },
    ]);
};
