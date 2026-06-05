import { DiffProfilesForm } from 'src/components/DiffProfilesForm/DiffProfilesForm';

import type { Page } from './Page';


export const DiffLists: Page = ({ header }) => {
    return <>
        {header}
        <DiffProfilesForm />
    </>;
};
