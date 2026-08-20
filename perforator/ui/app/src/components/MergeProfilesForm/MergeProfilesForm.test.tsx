import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';

import { describe, expect, it, jest } from '@jest/globals';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import type * as MergeProfilesFormModule from './MergeProfilesForm';


const BUILD_PATH = '/build';
const mockService = 'perforator';
const mockTokens = `[["service","=","${mockService}"]]`;
const mockSelector = `{service="${mockService}"}`;
const mockInterval = {
    start: 'now-2h',
    end: 'now-1h',
};
const mockMaxProfiles = 321;
const expectedSearchParams = {
    selector: mockSelector,
    tokens: mockTokens,
    from: mockInterval.start,
    to: mockInterval.end,
    maxProfiles: String(mockMaxProfiles),
};

jest.mock('@perforator/flamegraph', () => ({
    Hotkey: ({ value }: {value: string}) => <span>{value}</span>,
}));

jest.mock('../ProfileTable/ProfileTable', () => ({
    ProfileTable: ({ query }: {query: {selector?: string}}) => (
        <output aria-label="Preview selector">{query.selector}</output>
    ),
}));

jest.mock('../TimeIntervalInput/TimeIntervalInput', () => ({
    TimeIntervalInput: ({ onUpdate }: {onUpdate: (value: {start: string; end: string}) => void}) => (
        <button onClick={() => onUpdate(mockInterval)}>Use test interval</button>
    ),
}));

jest.mock('./SampleSizeInput/SampleSizeInput', () => ({
    SampleSizeInput: ({ onUpdate }: {onUpdate: (value: number) => void}) => (
        <button onClick={() => onUpdate(mockMaxProfiles)}>Use 321 profiles</button>
    ),
}));

jest.mock('./TokensInput/TokensInput', () => ({
    TokensInput: ({ onUpdate }: {onUpdate: (value: string) => void}) => (
        <button onClick={() => onUpdate(mockTokens)}>Use service token</button>
    ),
}));

const { MergeProfilesForm } = jest.requireActual<typeof MergeProfilesFormModule>('./MergeProfilesForm');

describe('MergeProfilesForm', () => {
    it('should preview and submit the edited query', async () => {
        // Arrange
        const user = userEvent.setup();
        renderMergeProfilesForm();

        // Act
        await user.click(screen.getByRole('button', { name: 'Use service token' }));
        await user.click(screen.getByRole('button', { name: 'Use test interval' }));
        await user.click(screen.getByRole('button', { name: 'Use 321 profiles' }));
        await user.click(screen.getByRole('button', { name: /Merge profiles/ }));

        // Assert
        await expectBuildLocation(expectedSearchParams);
    });

    it('should submit the query with the keyboard shortcut', async () => {
        // Arrange
        const user = userEvent.setup();
        renderMergeProfilesForm();
        await user.click(screen.getByRole('button', { name: 'Use service token' }));
        await user.click(screen.getByRole('button', { name: 'Use test interval' }));
        await user.click(screen.getByRole('button', { name: 'Use 321 profiles' }));
        await waitFor(() => expect(screen.getByLabelText('Preview selector')).toHaveTextContent(mockSelector));

        // Act
        fireEvent.keyDown(screen.getByRole('button', { name: /Merge profiles/ }), {
            code: 'Enter',
            ctrlKey: true,
        });

        // Assert
        await expectBuildLocation(expectedSearchParams);
    });

    it('should not submit an empty selector', async () => {
        // Arrange
        const user = userEvent.setup();
        renderMergeProfilesForm();

        // Act
        await user.click(screen.getByRole('button', { name: /Merge profiles/ }));

        // Assert
        expect(screen.queryByLabelText('Current location')).not.toBeInTheDocument();
        expect(screen.getByText('Preview of profiles matching selector')).toBeVisible();
    });
});

// Helpers

function renderMergeProfilesForm() {
    return render(
        <MemoryRouter initialEntries={['/']}>
            <Routes>
                <Route path="/" element={<MergeProfilesForm />} />
                <Route path={BUILD_PATH} element={<LocationOutput />} />
            </Routes>
        </MemoryRouter>,
    );
}

function LocationOutput() {
    const location = useLocation();
    return <output aria-label="Current location">{location.pathname}{location.search}</output>;
}

async function expectBuildLocation(expectedParams: Record<string, string>) {
    const location = await screen.findByLabelText('Current location');
    const url = new URL(location.textContent!, 'http://localhost');
    expect(url.pathname).toBe(BUILD_PATH);
    expect(Object.fromEntries(url.searchParams)).toEqual(expectedParams);
}
