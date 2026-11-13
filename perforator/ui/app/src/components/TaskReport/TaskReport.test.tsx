import { generatePath, Route, Routes, useLocation } from 'react-router-dom';

import { afterEach, beforeEach, describe, expect, it, jest } from '@jest/globals';
import { fireEvent, screen } from '@testing-library/react';

import { LocalStorageKey } from 'src/const/localStorage';
import { routes } from 'src/const/routes';
import { uiFactory } from 'src/factory';
import type { TaskResult } from 'src/models/Task';
import { UserSettingsProvider } from 'src/providers/UserSettingsProvider';
import { renderWithRouter } from 'src/test/renderWithRouter';
import { BuildProfileRum, buildProfileRum } from 'src/utils/buildProfileRum';

import i18n from './i18n';
import type * as TaskReportModule from './TaskReport';


jest.mock('../Fullscreen/FullscreenContext', () => ({ useFullscreen: () => ({ enabled: false }) }));
jest.mock('./PprofFlamegraph/PprofFlamegraph', () => ({
    PprofFlamegraph: ({ onRetryLegacy }: {onRetryLegacy: (options: {showLineNumbers: boolean}) => void}) =>
        <button onClick={() => onRetryLegacy({ showLineNumbers: true })}>Retry</button>,
}));
jest.mock('./TaskFlamegraph/TaskFlamegraph', () => ({
    TaskFlamegraph: ({ onLineNumbersChange }: {onLineNumbersChange: () => void}) =>
        <button onClick={onLineNumbersChange}>Line numbers</button>,
}));

const currentLocationLabel = 'Current location';

const { TaskReport } = jest.requireActual<typeof TaskReportModule>('./TaskReport');

function LocationOutput() {
    const location = useLocation();
    return <output aria-label={currentLocationLabel}>{location.pathname}{location.search}</output>;
}

function renderTask(task: TaskResult, search = '') {
    window.history.replaceState({}, '', `/${search}`);
    return renderWithRouter(
        <UserSettingsProvider><Routes>
            <Route path={routes.task} element={<><TaskReport task={task} taskId="task-1" /><LocationOutput /></>} />
            <Route path={routes.build} element={<LocationOutput />} />
        </Routes></UserSettingsProvider>,
        [`${generatePath(routes.task, { taskId: 'task-1' })}${search}`],
    );
}

function mergeTask(format: 'RawProfile' | 'JSONFlamegraph', queryMaxSamples?: number, maxSamples = 100): TaskResult {
    return {
        Spec: { MergeProfiles: {
            Query: { Selector: '{service="test"}', MaxSamples: queryMaxSamples },
            MaxSamples: maxSamples,
            Format: { [format]: { ShowLineNumbers: false } },
        } },
        Result: { MergeProfiles: { ProfileURL: 'https://example.test/profile' } },
    } as unknown as TaskResult;
}

describe('TaskReport derived actions', () => {
    beforeEach(() => {
        localStorage.clear();
        jest.spyOn(uiFactory(), 'wasmEnabled').mockReturnValue(true);
    });
    afterEach(() => { jest.restoreAllMocks(); });
    it.each([[0, '100'], [25, '25'], [undefined, '100']])('retries a raw task with the effective MaxSamples %s', (queryMaxSamples, expected) => {
        localStorage.setItem(LocalStorageKey.UserSettings, JSON.stringify({ mergeIntoPprofByDefault: true }));
        renderTask(mergeTask('RawProfile', queryMaxSamples), '?maxDepth=8&minWeight=0.1&showFileNames=false');
        fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
        const url = new URL(screen.getByLabelText(currentLocationLabel).textContent!, 'http://localhost');
        expect(url.pathname).toBe(routes.build);
        expect(url.searchParams.get('maxProfiles')).toBe(expected);
        expect(url.searchParams.get('selector')).toBe('{service="test"}');
        expect(url.searchParams.get('lineNumbers')).toBe('true');
        expect(url.searchParams.get('format')).toBe('json');
        expect(url.searchParams.get('maxDepth')).toBe('8');
    });

    it.each([false, true])('keeps JSON format and pprof options with opt-in %s', enabled => {
        localStorage.setItem(LocalStorageKey.UserSettings, JSON.stringify({ mergeIntoPprofByDefault: enabled }));
        renderTask(mergeTask('JSONFlamegraph'), '?maxDepth=8&minWeight=0.1&showFileNames=false');
        fireEvent.click(screen.getByRole('button', { name: 'Line numbers' }));
        const url = new URL(screen.getByLabelText(currentLocationLabel).textContent!, 'http://localhost');
        expect(url.searchParams.get('format')).toBe('json');
        expect(url.searchParams.get('lineNumbers')).toBe('true');
        expect(url.searchParams.get('maxDepth')).toBe('8');
        expect(url.searchParams.get('minWeight')).toBe('0.1');
        expect(url.searchParams.get('showFileNames')).toBe('false');
    });

    it('keeps format and options on a cached line-number toggle', () => {
        localStorage.setItem(LocalStorageKey.CachedLineNumberTasks, JSON.stringify({ 'task-1': 'task-2' }));
        renderTask(mergeTask('JSONFlamegraph'), '?maxDepth=8');
        fireEvent.click(screen.getByRole('button', { name: 'Line numbers' }));
        const url = new URL(screen.getByLabelText(currentLocationLabel).textContent!, 'http://localhost');
        expect(url.pathname).toBe(generatePath(routes.task, { taskId: 'task-2' }));
        expect(url.searchParams.get('format')).toBeNull();
        expect(url.searchParams.get('maxDepth')).toBe('8');
        expect(url.searchParams.get('lineNumbers')).toBe('true');
    });

    it('rebuilds instead of reusing an invalid cached task', () => {
        const cached = { taskId: 'task-2', format: 'json' };
        localStorage.setItem(LocalStorageKey.CachedLineNumberTasks, JSON.stringify({ 'task-1': cached }));
        renderTask(mergeTask('JSONFlamegraph'));
        fireEvent.click(screen.getByRole('button', { name: 'Line numbers' }));
        const url = new URL(screen.getByLabelText(currentLocationLabel).textContent!, 'http://localhost');
        expect(url.pathname).toBe(routes.build);
        expect(url.searchParams.get('format')).toBe('json');
        expect(url.searchParams.get('prevTask')).toBe('task-1');
    });

    it.each([undefined, false])('keeps raw profiles download-only when opt-in is %s', enabled => {
        if (enabled !== undefined) {
            localStorage.setItem(LocalStorageKey.UserSettings, JSON.stringify({ mergeIntoPprofByDefault: enabled }));
        }
        renderTask(mergeTask('RawProfile'));
        expect(screen.getByText(i18n('nothingToShow'))).toBeVisible();
        expect(screen.getByText(i18n('rawProfile'))).toBeVisible();
        expect(screen.getByRole('link', { name: i18n('download') })).toHaveAttribute('href', 'https://example.test/profile');
        expect(screen.queryByRole('button', { name: 'Retry' })).not.toBeInTheDocument();
    });

    it('excludes download-only raw tasks from render measurements', () => {
        const tracker = new BuildProfileRum(() => ({}));
        tracker.navigate({ key: 'build', pathname: '/build', search: '?format=raw' });
        const attempt = tracker.forBuild('build')!;
        jest.spyOn(buildProfileRum, 'forTask').mockReturnValue(attempt);
        const skip = jest.spyOn(attempt, 'skip');
        const finish = jest.spyOn(attempt, 'finish');
        renderTask(mergeTask('RawProfile'));
        expect(skip).toHaveBeenCalledTimes(1);
        expect(finish).not.toHaveBeenCalled();
    });

    it('keeps raw profiles download-only despite a saved opt-in when the factory disables WASM', () => {
        jest.spyOn(uiFactory(), 'wasmEnabled').mockReturnValue(false);
        localStorage.setItem(LocalStorageKey.UserSettings, JSON.stringify({ mergeIntoPprofByDefault: true }));

        renderTask(mergeTask('RawProfile'));

        expect(screen.getByRole('link', { name: i18n('download') })).toHaveAttribute('href', 'https://example.test/profile');
        expect(screen.queryByRole('button', { name: 'Retry' })).not.toBeInTheDocument();
    });
});
