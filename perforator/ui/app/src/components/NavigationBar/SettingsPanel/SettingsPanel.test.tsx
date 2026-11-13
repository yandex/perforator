import { afterEach, describe, expect, it, jest } from '@jest/globals';
import { render, screen } from '@testing-library/react';

import { LocalStorageKey } from 'src/const/localStorage';
import { uiFactory } from 'src/factory';
import { UserSettingsProvider } from 'src/providers/UserSettingsProvider';

import i18n from './i18n';
import { SettingsPanel } from './SettingsPanel';


describe('SettingsPanel', () => {
    afterEach(() => { jest.restoreAllMocks(); });

    it.each([false, true])('should offer local rendering only when the factory enables WASM: %s', enabled => {
        jest.spyOn(uiFactory(), 'wasmEnabled').mockReturnValue(enabled);
        localStorage.setItem(LocalStorageKey.UserSettings, JSON.stringify({ mergeIntoPprofByDefault: true }));

        render(<UserSettingsProvider><SettingsPanel /></UserSettingsProvider>);

        const toggle = screen.queryByRole('switch', { name: i18n('localRendering') });
        if (enabled) {
            expect(toggle).toBeChecked();
        } else {
            expect(toggle).not.toBeInTheDocument();
        }
    });
});
