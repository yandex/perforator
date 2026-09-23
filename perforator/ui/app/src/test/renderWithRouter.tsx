import type { ReactNode } from 'react';
import { generatePath, MemoryRouter } from 'react-router-dom';

import { render } from '@testing-library/react';

import { ThemeProvider } from '@gravity-ui/uikit';

import { routes } from 'src/const/routes';


export function renderWithRouter(ui: ReactNode, initialEntries: string[] = [generatePath(routes.home)]) {
    return render(
        <ThemeProvider theme="light">
            <MemoryRouter initialEntries={initialEntries}>{ui}</MemoryRouter>
        </ThemeProvider>,
    );
}
