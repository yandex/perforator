import { MergeProfilesForm } from 'src/components/MergeProfilesForm/MergeProfilesForm';

import type { Page } from './Page';


export const ProfileList: Page = ({ header }) => {
    return <>
        <MergeProfilesForm header={header} />
    </>;
};
