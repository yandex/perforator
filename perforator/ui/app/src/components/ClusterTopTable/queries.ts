import { useCallback } from 'react';

import type { QueryFunctionContext } from '@tanstack/react-query';
import { useInfiniteQuery } from '@tanstack/react-query';

import type { ClusterTopResponse } from 'src/generated/perforator/proto/perforator/perforator';
import { ClusterTopGenerationStatus } from 'src/generated/perforator/proto/perforator/perforator';
import { apiClient } from 'src/utils/api';


export const PAGE_LIMIT = '100';

const IN_PROGRESS_STALE_TIME = 10_000;
const IN_PROGRESS_GC_TIME = 60_000;
const COMPLETED_STALE_TIME = 10 * 60_000;
const COMPLETED_GC_TIME = 60 * 60_000;


export const clusterTopGenerationsQueryKeys = () => ['clusterTop', 'generations'] as const;

export const clusterTopFunctionsQueryKeys = (generation: number, orderBy: string, filter: string) => [
    'clusterTop',
    'functions',
    generation,
    orderBy,
    filter,
] as const;

export const clusterTopServicesQueryKeys = (generation: number, orderBy: string, functionName: string) => [
    'clusterTop',
    'services',
    generation,
    orderBy,
    functionName,
] as const;


export function getGenerationCachePolicy(status?: ClusterTopGenerationStatus) {
    if (status === ClusterTopGenerationStatus.COMPLETED) {
        return {
            staleTime: COMPLETED_STALE_TIME,
            gcTime: COMPLETED_GC_TIME,
            refetchInterval: false as const,
        };
    }

    return {
        staleTime: IN_PROGRESS_STALE_TIME,
        gcTime: IN_PROGRESS_GC_TIME,
        refetchInterval: IN_PROGRESS_STALE_TIME,
    };
}

export function getFunctionTopNextPageParam(lastPage: ClusterTopResponse, allPages: ClusterTopResponse[]) {
    return lastPage.HasMore ? allPages.length * Number(PAGE_LIMIT) : undefined;
}

type UseFunctionTopArgs = {
    generation: number;
    orderBy: string;
    currentFilter?: string;
    generationStatus: ClusterTopGenerationStatus;
};

export function useFunctionTopQuery({ generation, orderBy, currentFilter, generationStatus }: UseFunctionTopArgs) {

    const queryFn = useCallback(({ pageParam }: QueryFunctionContext): Promise<ClusterTopResponse> => apiClient.getFunctionTop({
        Generation: generation,
        OrderBy: orderBy,
        Pagination: { Offset: String(pageParam), Limit: PAGE_LIMIT },
        FunctionPattern: currentFilter || undefined,
    }).then((value) => value.data), [currentFilter, generation, orderBy]);

    return useInfiniteQuery({
        queryKey: clusterTopFunctionsQueryKeys(generation, orderBy, currentFilter ?? ''),
        queryFn: queryFn,
        initialPageParam: 0,
        getNextPageParam: getFunctionTopNextPageParam,
        retry: 1,
        ...getGenerationCachePolicy(generationStatus),
    });
}
