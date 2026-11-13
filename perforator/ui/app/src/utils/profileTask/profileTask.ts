import { createSearchParams, generatePath } from 'react-router-dom';

import { LocalStorageKey } from 'src/const/localStorage';
import { routes } from 'src/const/routes';
import { uiFactory } from 'src/factory';
import type { FlamegraphOptions, PostprocessOptions } from 'src/generated/perforator/proto/perforator/perforator';
import { PythonStackPrettifyLevel } from 'src/generated/perforator/proto/perforator/perforator';
import type { ProfileTaskQuery, TaskResult } from 'src/models/Task';
import type { PythonPrettifyLevel } from 'src/providers/UserSettingsProvider/UserSettings';

import { apiClient } from '../api';
import { getFormat } from '../renderingFormat';
import { makeSelector } from '../selector';

import { getRenderFlamegraph } from './getRenderFlamegraph';


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

export const taskFormatToQueryFormat = (task: TaskResult | null): ProfileTaskQuery['format'] => {
    const format = getFormat(task?.Spec?.MergeProfiles?.Format ?? task?.Spec?.DiffProfiles?.RenderFormat);
    return format === 'RawProfile' ? 'raw' : format === 'TextProfile' ? 'text' : 'json';
};


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
    settings: {pythonPrettifyLevel?: PythonPrettifyLevel; mergeIntoPprofByDefault?: boolean},
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
    const flamegraphRender = getRenderFlamegraph(query, flamegraphOptions, {
        wasmEnabled: uiFactory().wasmEnabled(),
        mergeIntoPprofByDefault: settings.mergeIntoPprofByDefault,
    });
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

    if (query.prevTask && flamegraphRender.JSONFlamegraph) {
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
            pathname: generatePath(routes.build),
            search: createSearchParams(taskQueryToSearchParams(query)).toString(),
        },
        { replace },
    );
}
