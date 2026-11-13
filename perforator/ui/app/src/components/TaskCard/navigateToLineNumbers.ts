import type { NavigateFunction } from 'react-router-dom';
import { generatePath } from 'react-router-dom';

import { LocalStorageKey } from 'src/const/localStorage';
import { routes } from 'src/const/routes';
import { uiFactory } from 'src/factory';
import type { ProfileTaskQuery } from 'src/models/Task';
import { redirectToTaskPage } from 'src/utils/profileTask';
import { preserveWellKnownQueryParams } from 'src/utils/profileTask/preserveWellKnown';


export function navigateToLineNumbers(navigate: NavigateFunction, query: ProfileTaskQuery, currentTaskId: string) {
    let localCache;
    try {
        const cacheField = localStorage.getItem(LocalStorageKey.CachedLineNumberTasks);
        localCache = cacheField ? JSON.parse(cacheField) : {};
    } catch {
        localCache = {};
    }
    const q = preserveWellKnownQueryParams(new URLSearchParams(window.location.search));
    if (query.lineNumbers) {
        q.set('lineNumbers', query.lineNumbers);
    }
    const cached = localCache?.[currentTaskId];
    if (query.format === 'json' && typeof cached === 'string') {
        navigate(`${generatePath(routes.task, { taskId: cached })}?${q.toString()}`);
    } else {
        query.prevTask = currentTaskId;
        query = {
            ...(Object.fromEntries(q.entries())),
            ...query,
        };

        uiFactory().reachGoal('LINE_NUMBERS');
        redirectToTaskPage(navigate, query);
    }
}
