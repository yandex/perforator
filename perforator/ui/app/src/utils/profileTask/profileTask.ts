import { createSearchParams } from 'react-router-dom';

import { LocalStorageKey } from 'src/const/localStorage';
import { uiFactory } from 'src/factory';
import type { FlamegraphOptions, PostprocessOptions, RenderFormat } from 'src/generated/perforator/proto/perforator/perforator';
import { PythonStackPrettifyLevel } from 'src/generated/perforator/proto/perforator/perforator';
import type { ProfileTaskQuery } from 'src/models/Task';
import type { PythonPrettifyLevel } from 'src/providers/UserSettingsProvider/UserSettings';

import { apiClient } from '../api';
import { makeSelector } from '../selector';


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
    if (query.rawProfile === 'true' || query.format === 'raw') {
        return { RawProfile: flamegraphOptions };
    }
    if (query.format === 'text') {
        return { TextProfile: flamegraphOptions };
    }
    return { JSONFlamegraph: flamegraphOptions };

}

function toPythonStackPrettifyLevelProto(level: PythonPrettifyLevel | undefined): PythonStackPrettifyLevel {
    switch (level) {
    case 'mixed':
        return PythonStackPrettifyLevel.PYTHON_STACK_PRETTIFY_DEFAULT;
    case 'python-only':
        return PythonStackPrettifyLevel.PYTHON_STACK_PRETTIFY_STRICT;
    default:
        return PythonStackPrettifyLevel.PYTHON_STACK_PRETTIFY_OFF;
    }
}

export const startProfileTask = async (
    query: ProfileTaskQuery,
    settings: {pythonPrettifyLevel?: PythonPrettifyLevel},
): Promise<string> => {
    const diffSelector = query.diffSelector;

    const baseRequest = {
        IdempotencyKey: query.idempotencyKey,
    };


    const flamegraphOptions: FlamegraphOptions = {
        MaxDepth: 256,
        MinWeight: 1e-10,
        ShowLineNumbers: query.lineNumbers === 'true',
    };

    const symbolizeOptions = {
        Symbolize: true,
    };

    const maxProfiles = query.maxProfiles;
    const flamegraphRender = getRenderFlamegraph(query, flamegraphOptions);
    const prettifyLevel = toPythonStackPrettifyLevelProto(settings.pythonPrettifyLevel);
    const postprocessingOptions: PostprocessOptions = prettifyLevel !== PythonStackPrettifyLevel.PYTHON_STACK_PRETTIFY_OFF
        ? { MergePythonAndNativeStacks: true, MergePHPAndNativeStacks: true, PrettifyPythonStacksLevel: prettifyLevel }
        : {};

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
                            Postprocessing: postprocessingOptions,
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
                            Postprocessing: postprocessingOptions,
                        },
                        MaxSamples: maxProfiles,
                        Query: {
                            Selector: makeSelector(query),
                        },
                    },
                },
            };

    const response = await apiClient.startTask(request);
    const taskId = response?.data?.TaskID;

    if (query.prevTask) {
        try {
            const cache = localStorage.getItem(LocalStorageKey.CachedLineNumberTasks);
            const cachedLineNumberTasks = cache ? JSON.parse(cache) : {};
            cachedLineNumberTasks[query.prevTask] = taskId;
            cachedLineNumberTasks[taskId] = query.prevTask;
            localStorage.setItem(LocalStorageKey.CachedLineNumberTasks, JSON.stringify(cachedLineNumberTasks));
        } catch (e) {
            console.error(e);
        }
    }

    return taskId;
};

export function redirectToTaskPage<Q extends ProfileTaskQuery> (
    navigate: (data: object, options: object) => void,
    query: Q,
    replace = false,
) {
    return navigate(
        {
            pathname: '/build',
            search: createSearchParams(taskQueryToSearchParams(query)).toString(),
        },
        { replace },
    );
}
