import type { QueryFunctionContext } from '@tanstack/react-query';
import { useInfiniteQuery } from '@tanstack/react-query';

import type { ListTasksResponse } from 'src/generated/perforator/proto/perforator/task_service';
import { apiClient, getPagination } from 'src/utils/api';
import { getIsoDate } from 'src/utils/date';


export const PAGE_SIZE = 100;

const STALE_TIME = 10_000;

export const tasksQueryKeys = (from: string | undefined, to: string | undefined, user: string) =>
    ['tasks', from, to, user] as const;

function getNextPageParam(lastPage: ListTasksResponse, allPages: ListTasksResponse[]) {
    const loaded = allPages.reduce((sum, page) => sum + (page.Tasks?.length ?? 0), 0);
    const total = Number(lastPage.TotalCount ?? 0);
    return loaded < total ? allPages.length + 1 : undefined;
}

type UseTasksQueryArgs = {
    from: string | undefined;
    to: string | undefined;
    user: string;
};

export function useTasksQuery({ from, to, user }: UseTasksQueryArgs) {
    return useInfiniteQuery({
        queryKey: tasksQueryKeys(from, to, user),
        queryFn: ({ pageParam }: QueryFunctionContext): Promise<ListTasksResponse> => apiClient.getTasks({
            Query: {
                Author: user,
                From: getIsoDate(from),
                To: getIsoDate(to),
            },
            Pagination: getPagination({ page: pageParam as number, pageSize: PAGE_SIZE }),
        }).then((response) => response.data),
        initialPageParam: 1,
        getNextPageParam,
        staleTime: STALE_TIME,
        retry: 1,
    });
}
