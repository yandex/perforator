import { beforeEach, describe, expect, it, jest } from '@jest/globals';

import { LocalStorageKey } from 'src/const/localStorage';
import { routes } from 'src/const/routes';

import { navigateToLineNumbers } from './navigateToLineNumbers';


describe('navigateToLineNumbers', () => {
    beforeEach(() => {
        localStorage.clear();
        window.history.replaceState({}, '', '/');
    });

    it.each(['raw', 'text'] as const)('should rebuild instead of reusing a cached JSON task for %s output', format => {
        localStorage.setItem(LocalStorageKey.CachedLineNumberTasks, JSON.stringify({ 'old-task': 'json-task' }));
        const navigate = jest.fn<() => void>();

        navigateToLineNumbers(navigate, { from: '', to: '', maxProfiles: 1, format }, 'old-task');

        expect(navigate).toHaveBeenCalledWith(expect.objectContaining({ pathname: routes.build }), { replace: false });
    });
});
