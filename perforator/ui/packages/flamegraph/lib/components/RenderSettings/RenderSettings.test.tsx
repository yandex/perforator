/** @jest-environment jsdom */ // eslint-disable-line jsdoc/check-tag-names

import React from 'react';

import { describe, expect, it, jest } from '@jest/globals';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import { ThemeProvider } from '@gravity-ui/uikit';

import { defaultPprofOptions } from '../../pprof/options';

import { RenderSettings } from './RenderSettings';

import '@testing-library/jest-dom/jest-globals';


// Gravity UI's popup hooks use matchMedia, which jsdom does not implement.
window.matchMedia ??= (media: string): MediaQueryList => ({
    media,
    matches: false,
    onchange: null,
    addListener() {},
    removeListener() {},
    addEventListener() {},
    removeEventListener() {},
    dispatchEvent: () => false,
});

const RENDER_SETTINGS = 'Render settings';
const ADVANCED = 'Advanced';
const APPLY = 'Apply';
const MAX_DEPTH_LABEL = 'Maximum depth (0 means unlimited)';
const MIN_WEIGHT_LABEL = 'Minimum weight (% of the full profile, 0–100)';
const MAX_DEPTH_RANGE_ERROR = 'Maximum depth must be an integer between 0 and 4294967295.';

const options = {
    maxDepth: 10,
    minWeight: 0.2,
    showFileNames: false,
    showLineNumbers: true,
};

describe('RenderSettings', () => {
    it('opens the settings dialog and focuses the first field', async () => {
        const user = userEvent.setup();
        renderSettings(jest.fn());

        await user.click(screen.getByRole('button', { name: RENDER_SETTINGS }));

        const dialog = await screen.findByRole('dialog');
        expect(dialog).toBeVisible();
        expect(screen.getByRole('heading', { name: 'Local render settings' })).toBeVisible();
        await waitFor(() => expect(screen.getByLabelText(MIN_WEIGHT_LABEL)).toHaveFocus());
    });

    it('explains invalid values next to their fields without native validation blocking submit', async () => {
        const user = userEvent.setup();
        renderSettings(jest.fn());
        await user.click(screen.getByRole('button', { name: RENDER_SETTINGS }));

        await user.click(screen.getByRole('button', { name: ADVANCED }));
        const depth = screen.getByLabelText(MAX_DEPTH_LABEL);
        const weight = screen.getByLabelText(MIN_WEIGHT_LABEL);
        fireEvent.change(depth, { target: { value: '' } });
        fireEvent.change(weight, { target: { value: 'not a number' } });
        await user.click(screen.getByRole('button', { name: APPLY }));

        expect(await screen.findByText('Enter a maximum depth.')).toBeVisible();
        expect(screen.getByText('Enter a number for minimum weight.')).toBeVisible();
        expect(screen.getAllByRole('alert')).toHaveLength(2);
        await waitFor(() => expect(depth).toHaveFocus());

        fireEvent.change(depth, { target: { value: '-1' } });
        fireEvent.change(weight, { target: { value: '101' } });
        await user.click(screen.getByRole('button', { name: APPLY }));
        expect(await screen.findByText(MAX_DEPTH_RANGE_ERROR)).toBeVisible();
        expect(screen.getByText('Minimum weight must be a percentage between 0 and 100.')).toBeVisible();

        fireEvent.change(depth, { target: { value: '4294967296' } });
        await user.click(screen.getByRole('button', { name: APPLY }));
        expect(await screen.findByText(MAX_DEPTH_RANGE_ERROR)).toBeVisible();
    });

    it.each([defaultPprofOptions.minWeight, 0.12345678901234566])('preserves an unchanged fraction exactly (%s)', async (minWeight) => {
        const user = userEvent.setup();
        const onApply = jest.fn();
        const currentOptions = { ...options, minWeight };
        renderSettings(onApply, currentOptions);
        await user.click(screen.getByRole('button', { name: RENDER_SETTINGS }));
        expect(screen.getByLabelText(MIN_WEIGHT_LABEL)).toHaveValue(String(minWeight * 100));
        await user.click(screen.getByRole('button', { name: APPLY }));
        expect(onApply).toHaveBeenCalledWith(currentOptions);
    });

    it('converts percentages to fractions and keeps unlimited depth in Advanced', async () => {
        const user = userEvent.setup();
        const onApply = jest.fn();
        renderSettings(onApply);
        await user.click(screen.getByRole('button', { name: RENDER_SETTINGS }));
        fireEvent.change(screen.getByLabelText(MIN_WEIGHT_LABEL), { target: { value: '5' } });
        await user.click(screen.getByRole('button', { name: ADVANCED }));
        expect(screen.getByText('Includes service label levels (container, process and thread). 0 means unlimited.')).toBeVisible();
        fireEvent.change(screen.getByLabelText(MAX_DEPTH_LABEL), { target: { value: '0' } });
        await user.click(screen.getByRole('button', { name: APPLY }));
        expect(onApply).toHaveBeenCalledWith({ ...options, minWeight: 0.05, maxDepth: 0 });
    });

    it('lets Cancel discard edits and Reset restore existing defaults before Apply', async () => {
        const user = userEvent.setup();
        const onApply = jest.fn();
        renderSettings(onApply);
        await user.click(screen.getByRole('button', { name: RENDER_SETTINGS }));

        await user.click(screen.getByRole('button', { name: ADVANCED }));
        const depth = screen.getByLabelText(MAX_DEPTH_LABEL);
        await user.clear(depth);
        await user.type(depth, '99');
        await user.click(screen.getByRole('button', { name: 'Cancel' }));
        expect(onApply).not.toHaveBeenCalled();
        expect(screen.getByRole('button', { name: RENDER_SETTINGS })).toHaveFocus();

        await user.click(screen.getByRole('button', { name: RENDER_SETTINGS }));
        await user.click(screen.getByRole('button', { name: ADVANCED }));
        expect(screen.getByLabelText(MAX_DEPTH_LABEL)).toHaveValue('10');
        await user.click(screen.getByRole('button', { name: 'Reset' }));
        await user.click(screen.getByRole('button', { name: APPLY }));

        expect(onApply).toHaveBeenCalledWith({
            ...defaultPprofOptions,
            showLineNumbers: options.showLineNumbers,
        });
    });
});

function renderSettings(onApply: jest.Mock, currentOptions = options) {
    return render(
        <ThemeProvider theme="light">
            <RenderSettings options={currentOptions} busy={false} onApply={onApply} />
        </ThemeProvider>,
    );
}
