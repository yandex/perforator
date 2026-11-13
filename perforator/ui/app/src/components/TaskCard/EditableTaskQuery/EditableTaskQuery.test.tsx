import type { ReactNode } from 'react';
import { generatePath, Route, Routes, useLocation } from 'react-router-dom';

import { describe, expect, it, jest } from '@jest/globals';
import { fireEvent, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import { routes } from 'src/const/routes';
import type { TaskResult } from 'src/models/Task';
import { renderWithRouter } from 'src/test/renderWithRouter';

import type * as EditableTaskQueryModule from './EditableTaskQuery';
import i18n from './i18n';


interface TaskOptions {
    profileId: string;
    service: string;
    from: string;
    to: string;
    maxProfiles: number;
    lineNumbers: boolean;
    format?: 'raw' | 'json';
}

const BUILD_PATH = routes.build;
const HEADER_TEXT = 'Profile';
const DEFAULT_TASK_OPTIONS: TaskOptions = {
    profileId: 'profile-1',
    service: 'api',
    from: '2026-08-19T10:00:00.000Z',
    to: '2026-08-19T10:10:00.000Z',
    maxProfiles: 1,
    lineNumbers: true,
};
const mockSelectedInterval = {
    start: '2026-08-19T10:00:00.000Z',
    end: '2026-08-19T10:05:00.000Z',
};

jest.mock('src/components/MergeProfilesForm/TokensInput/SwitchableSelectorInput', () => ({
    SwitchableSelectorInput: ({ selector, onUpdate }: {selector?: string; onUpdate: (value: string) => void}) => (
        <input
            aria-label="Selector"
            value={selector ?? ''}
            onChange={event => onUpdate(event.target.value)}
        />
    ),
}));

jest.mock('../../TimeIntervalInput/TimeIntervalInput', () => ({
    TimeIntervalInput: ({ header, onUpdate, additionalItems }: {
        header: ReactNode;
        onUpdate: (value: {start: string; end: string}) => void;
        additionalItems?: ReactNode;
    }) => (
        <div>
            {header}
            <button onClick={() => onUpdate(mockSelectedInterval)}>Use selected interval</button>
            {additionalItems}
        </div>
    ),
}));

const { EditableTaskQuery } = jest.requireActual<typeof EditableTaskQueryModule>('./EditableTaskQuery');

describe('EditableTaskQuery', () => {
    it('should render a minimal header without task timestamps', () => {
        // Arrange
        renderEditableTaskQuery(null);

        // Act
        const header = screen.getByRole('heading', { name: HEADER_TEXT });

        // Assert
        expect(header).toBeVisible();
        expect(screen.queryByRole('textbox', { name: 'Selector' })).not.toBeInTheDocument();
    });

    it('should enable controls after an edit and restore the selector on cancel', async () => {
        // Arrange
        const user = userEvent.setup();
        const profileId = 'profile-1';
        const service = 'api';
        const initialSelector = `{id="${profileId}",service="${service}"}`;
        const editedSelector = '{service="worker"}';
        renderEditableTaskQuery(makeTask({
            ...DEFAULT_TASK_OPTIONS,
            profileId,
            service,
        }));
        const selector = screen.getByRole('textbox', { name: 'Selector' });
        const cancelButton = screen.getByRole('button', { name: i18n('cancelChanges') });
        const saveButton = screen.getByRole('button', { name: i18n('saveChanges') });
        expect(cancelButton).toBeDisabled();
        expect(saveButton).toBeDisabled();

        // Act
        await user.clear(selector);
        await user.paste(editedSelector);

        // Assert
        expect(cancelButton).toBeEnabled();
        expect(saveButton).toBeEnabled();

        // Act
        await user.click(cancelButton);

        // Assert
        expect(selector).toHaveValue(initialSelector);
        expect(cancelButton).toBeDisabled();
        expect(saveButton).toBeDisabled();
    });

    it('should navigate with an edited selector and preserve task options', async () => {
        // Arrange
        const user = userEvent.setup();
        const profileId = 'profile-1';
        const service = 'api';
        const taskFrom = '2026-08-19T10:00:00.000Z';
        const taskTo = '2026-08-19T10:10:00.000Z';
        const maxProfiles = 1;
        const lineNumbers = true;
        const editedSelector = '{service="worker"}';
        const expectedFrom = '2026-08-19T09:55:00.000Z';
        const expectedTo = '2026-08-19T10:15:00.000Z';
        renderEditableTaskQuery(makeTask({
            profileId,
            service,
            from: taskFrom,
            to: taskTo,
            maxProfiles,
            lineNumbers,
        }));
        const selector = screen.getByRole('textbox', { name: 'Selector' });
        await user.clear(selector);
        await user.paste(editedSelector);

        // Act
        await user.click(screen.getByRole('button', { name: i18n('saveChanges') }));

        // Assert
        expectBuildLocation({
            selector: editedSelector,
            from: expectedFrom,
            to: expectedTo,
            maxProfiles: String(maxProfiles),
            lineNumbers: String(lineNumbers),
            format: 'json',
        });
    });

    it('should remove a single profile id when rerunning a selected interval', async () => {
        // Arrange
        const user = userEvent.setup();
        const profileId = 'profile-1';
        const service = 'api';
        const maxProfiles = 1;
        const lineNumbers = true;
        const expectedSelector = `{service="${service}"}`;
        renderEditableTaskQuery(makeTask({
            ...DEFAULT_TASK_OPTIONS,
            profileId,
            service,
            maxProfiles,
            lineNumbers,
        }));

        // Act
        await user.click(screen.getByRole('button', { name: 'Use selected interval' }));
        await user.click(screen.getByRole('button', { name: i18n('saveChanges') }));

        // Assert
        expectBuildLocation({
            selector: expectedSelector,
            from: mockSelectedInterval.start,
            to: mockSelectedInterval.end,
            maxProfiles: String(maxProfiles),
            lineNumbers: String(lineNumbers),
            format: 'json',
        });
    });

    it('should save edits with the keyboard shortcut', async () => {
        // Arrange
        const user = userEvent.setup();
        const profileId = 'profile-1';
        const service = 'api';
        const taskFrom = '2026-08-19T10:00:00.000Z';
        const taskTo = '2026-08-19T10:10:00.000Z';
        const maxProfiles = 1;
        const lineNumbers = true;
        const editedSelector = '{service="worker"}';
        const expectedFrom = '2026-08-19T09:55:00.000Z';
        const expectedTo = '2026-08-19T10:15:00.000Z';
        renderEditableTaskQuery(makeTask({
            profileId,
            service,
            from: taskFrom,
            to: taskTo,
            maxProfiles,
            lineNumbers,
        }));
        const selector = screen.getByRole('textbox', { name: 'Selector' });
        await user.clear(selector);
        await user.paste(editedSelector);

        // Act
        fireEvent.keyDown(selector, { code: 'Enter', metaKey: true });

        // Assert
        expectBuildLocation({
            selector: editedSelector,
            from: expectedFrom,
            to: expectedTo,
            maxProfiles: String(maxProfiles),
            lineNumbers: String(lineNumbers),
            format: 'json',
        });
    });

    it('keeps raw format and pprof options while editing', async () => {
        const user = userEvent.setup();
        renderEditableTaskQuery(makeTask({ ...DEFAULT_TASK_OPTIONS, format: 'raw' }), '?maxDepth=8&minWeight=0.1&showFileNames=false&lineNumbers=true');
        await user.click(screen.getByRole('button', { name: 'Use selected interval' }));
        await user.click(screen.getByRole('button', { name: 'Save changes' }));

        const url = currentLocationUrl();
        expect(url.searchParams.get('format')).toBe('raw');
        expect(url.searchParams.get('maxDepth')).toBe('8');
        expect(url.searchParams.get('minWeight')).toBe('0.1');
        expect(url.searchParams.get('showFileNames')).toBe('false');
        expect(url.searchParams.get('lineNumbers')).toBe('true');
    });
});

// Helpers

function renderEditableTaskQuery(task: TaskResult | null, search = '') {
    window.history.replaceState({}, '', `/${search}`);
    return renderWithRouter(
        <Routes>
            <Route
                path={routes.task}
                element={<EditableTaskQuery task={task} header={<h1>{HEADER_TEXT}</h1>} />}
            />
            <Route path={BUILD_PATH} element={<LocationOutput />} />
        </Routes>,
        [`${generatePath(routes.task, { taskId: 'task-1' })}${search}`],
    );
}

function LocationOutput() {
    const location = useLocation();
    return <output aria-label="Current location">{location.pathname}{location.search}</output>;
}

function expectBuildLocation(expectedSearchParams: Record<string, string>) {
    const url = currentLocationUrl();
    expect(url.pathname).toBe(BUILD_PATH);
    expect(Object.fromEntries(url.searchParams)).toEqual(expectedSearchParams);
}

function currentLocationUrl() {
    return new URL(screen.getByLabelText('Current location').textContent!, 'http://localhost');
}

function makeTask({
    profileId,
    service,
    from,
    to,
    maxProfiles,
    lineNumbers,
    format = 'json',
}: TaskOptions): TaskResult {
    return {
        Spec: {
            MergeProfiles: {
                MaxSamples: maxProfiles,
                Query: {
                    Selector: `{id="${profileId}", service="${service}", timestamp>="${from}", timestamp<="${to}"}`,
                },
                Format: {
                    [format === 'raw' ? 'RawProfile' : 'JSONFlamegraph']: format === 'raw' ? {} : { ShowLineNumbers: lineNumbers },
                },
            },
        },
    } as unknown as TaskResult;
}
