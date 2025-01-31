import { createSearchParams } from 'react-router-dom';

import { uiFactory } from 'src/factory';
import type { FlamegraphOptions, RenderFormat } from 'src/generated/perforator/proto/perforator/perforator';
import type { ProfileTaskQuery } from 'src/models/Task';

import { apiClient } from './api';
import { makeSelector } from './selector';


export const taskQueryToSearchParams = (query: ProfileTaskQuery): { [key: string]: string } => (
    Object.fromEntries(
        Object.entries(query)
            .filter(([_, value]) => value !== undefined)
            .map(([key, value]) => [key, value.toString()]),
    )
);

export const defaultProfileTaskQuery = (): ProfileTaskQuery => ({
    from: 'now-1d',
    to: 'now',
    maxProfiles: uiFactory().defaultSampleSize(),
});


function getRenderFlamegraph(query: ProfileTaskQuery, flamegraphOptions: FlamegraphOptions): Partial<RenderFormat> {
    if (query.rawProfile === 'true') {
        return { RawProfile: flamegraphOptions };
    }
    return { JSONFlamegraph: flamegraphOptions };

}

export const startProfileTask = async (
    query: ProfileTaskQuery,
): Promise<string> => {
    const diffSelector = query.diffSelector;

    const baseRequest = {
        IdempotencyKey: query.idempotencyKey,
    };


    const flamegraphOptions: FlamegraphOptions = {
        MaxDepth: 256,
        MinWeight: 1e-10,
    };

    const symbolizeOptions = {
        Symbolize: true,
    };

    const maxProfiles = query.maxProfiles;
    const flamegraphRender = getRenderFlamegraph(query, flamegraphOptions);

    const request =
        diffSelector
            ? {
                ...baseRequest,
                Spec: {
                    DiffProfiles: {
                        DiffQuery: {
                            Selector: diffSelector,
                            MaxSamples: maxProfiles,
                        },
                        BaselineQuery: {
                            Selector: makeSelector(query),
                            MaxSamples: maxProfiles,
                        },
                        SymbolizeOptions: symbolizeOptions,
                        RenderFormat: {
                            ...flamegraphRender,
                        },

                    },
                },
            }
            : {
                ...baseRequest,
                Spec: {
                    MergeProfiles: {
                        Format: {
                            ...flamegraphRender,
                            Symbolize: symbolizeOptions,
                        },
                        MaxSamples: maxProfiles,
                        Query: {
                            Selector: makeSelector(query),
                        },
                    },
                },
            };

    const response = await apiClient.startTask(request);
    return response?.data?.TaskID;
};

export const redirectToTaskPage = (
    navigate: (data: object, options: object) => void,
    query: ProfileTaskQuery,
    replace = false,
) => {
    navigate(
        {
            pathname: '/build',
            search: createSearchParams(taskQueryToSearchParams(query)).toString(),
        },
        { replace },
    );
};
