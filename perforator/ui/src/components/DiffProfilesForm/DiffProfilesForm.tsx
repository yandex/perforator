import React from 'react';

import { useNavigate } from 'react-router-dom';

import { Button } from '@gravity-ui/uikit';

import type { ProfileTaskQuery } from 'src/models/Task';
import { redirectToTaskPage } from 'src/utils/profileTask';
import { composeDiffQuery } from 'src/utils/selector';

import { MergeProfilesForm } from '../MergeProfilesForm/MergeProfilesForm';

import './DiffProfilesForm.scss';


export const DiffProfilesForm: React.FC = () => {
    const navigate = useNavigate();

    const [baseline, setBaseline] = React.useState<
        ProfileTaskQuery | undefined
    >();
    const [diff, setDiff] = React.useState<ProfileTaskQuery | undefined>();

    const handleBaseQuery = React.useCallback((query: ProfileTaskQuery) => {
        setBaseline(query);
    }, [setBaseline]);

    const handleDiffQuery = React.useCallback((query: ProfileTaskQuery) => {
        setDiff(query);
    }, [setDiff]);

    const leftForm = React.useMemo(() => (
        <MergeProfilesForm
            removeMergeButton
            diff
            onRender={handleBaseQuery}
            compactTable
            className={'diff-profiles-form__list-item'}
        />
    ), [handleBaseQuery]);

    const rightForm = React.useMemo(() => (
        <MergeProfilesForm
            removeMergeButton
            diff
            onRender={handleDiffQuery}
            compactTable
            inMemory
            className={'diff-profiles-form__list-item'}
        />
    ), [handleDiffQuery]);

    const renderDiff = async () => {
        if (baseline && diff) {
            redirectToTaskPage(navigate, composeDiffQuery(baseline, diff));
        }
    };

    return (
        <div className="diff-profiles-form">
            <Button view="action" className="diff-profiles-form__button" onClick={renderDiff}>Diff profiles</Button>
            <div className="diff-profiles-form__list">
                {leftForm}
                <div className="diff-profiles-form__spacer_vertical"></div>
                {rightForm}
            </div>
        </div>
    );
};
