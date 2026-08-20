import { describe, expect, it, jest } from '@jest/globals';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import { ThemeProvider } from '@gravity-ui/uikit';

import { getQueryFields } from 'src/providers/QuerySuggestProvider/fields';
import { QuerySuggestContext } from 'src/providers/QuerySuggestProvider/QuerySuggestContext';

import { TokensInput } from './TokensInput';


describe('TokensInput', () => {
    it('should show every configured selector field in autocomplete', async () => {
        // Arrange
        const user = userEvent.setup();
        const fields = new Map((await getQueryFields()).map(field => [field.field, field]));
        render(
            <ThemeProvider theme="light">
                <QuerySuggestContext.Provider value={{ fields }}>
                    <TokensInput initialTokens="[]" onUpdate={jest.fn()} />
                </QuerySuggestContext.Provider>
            </ThemeProvider>,
        );

        // Act
        await user.click(screen.getByRole('textbox', { name: 'Field field in token 0' }));

        // Assert
        const expectedFields = [
            'service',
            'cluster',
            'pod_id',
            'node_id',
            'cpu',
            'event_type',
            'build_ids',
            'system_name',
            'id',
        ];
        for (let index = 0; index < expectedFields.length; index++) {
            expect(await screen.findByText(expectedFields[index])).toBeVisible();
        }
    });
});
