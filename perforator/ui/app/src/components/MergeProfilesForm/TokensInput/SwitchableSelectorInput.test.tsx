import React from 'react';

import { describe, expect, it, jest } from '@jest/globals';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import { LocalStorageKey } from 'src/const/localStorage';

import type * as SwitchableSelectorInputModule from './SwitchableSelectorInput';


jest.mock('src/components/QueryLanguageEditor/lazy', () => {
    const ReactActual = jest.requireActual<typeof React>('react');
    return {
        QueryLanguageEditorImpl: ({ selector, onUpdate }: {selector?: string; onUpdate: (value: string) => void}) => (
            ReactActual.createElement('input', {
                'aria-label': 'Selector editor',
                value: selector ?? '',
                onChange: (event: React.ChangeEvent<HTMLInputElement>) => onUpdate(event.target.value),
            })
        ),
        QueryLanguageFallback: () => ReactActual.createElement('span', null, 'Loading selector editor'),
    };
});

jest.mock('src/providers/QuerySuggestProvider/QuerySuggestProvider', () => ({
    QuerySuggestProvider: ({ children }: {children: React.ReactNode}) => children,
}));

jest.mock('./TokensInput', () => ({
    TokensInput: () => <input aria-label="Tokens editor" />,
}));

const { SwitchableSelectorInput } = jest.requireActual<typeof SwitchableSelectorInputModule>('./SwitchableSelectorInput');

describe('SwitchableSelectorInput', () => {
    it('should use Tokens mode by default', () => {
        // Arrange
        render(<SelectorHarness />);

        // Act
        const tokenField = screen.getByRole('textbox', { name: 'Tokens editor' });

        // Assert
        expect(tokenField).toBeVisible();
        expect(screen.queryByRole('textbox', { name: 'Selector editor' })).not.toBeInTheDocument();
    });

    it('should restore the persisted Selector mode', async () => {
        // Arrange
        localStorage.setItem(LocalStorageKey.QueryInputKind, 'selector');

        // Act
        render(<SelectorHarness />);

        // Assert
        expect(await screen.findByRole('textbox', { name: 'Selector editor' })).toHaveValue('{service="api"}');
    });

    it('should preserve edits and persist mode changes', async () => {
        // Arrange
        const user = userEvent.setup();
        render(<SelectorHarness />);

        // Act
        await user.click(screen.getByText('Selector'));
        const editor = await screen.findByRole('textbox', { name: 'Selector editor' });
        await user.clear(editor);
        await user.paste('{service="worker"}');

        // Assert
        expect(screen.getByLabelText('Selector value')).toHaveTextContent('{service="worker"}');
        expect(localStorage.getItem(LocalStorageKey.QueryInputKind)).toBe('selector');
    });
});

// Helpers

function SelectorHarness() {
    const [selector, setSelector] = React.useState('{service="api"}');
    return (
        <>
            <SwitchableSelectorInput selector={selector} onUpdate={value => setSelector(value ?? '')} />
            <output aria-label="Selector value">{selector}</output>
        </>
    );
}
