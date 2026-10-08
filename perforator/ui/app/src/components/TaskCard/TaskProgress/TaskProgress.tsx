import React from 'react';

import type { ProgressTheme } from '@gravity-ui/uikit';
import { Progress } from '@gravity-ui/uikit';

import { TaskState } from 'src/models/Task';

import { ErrorPanel } from '../../ErrorPanel/ErrorPanel';

import i18n from './i18n';


export interface TaskProgressProps {
    state: TaskState;
    error?: string;
}

export const TaskProgress: React.FC<TaskProgressProps> = ({ state, error }: TaskProgressProps) => {
    if (state === TaskState.Failed || error) {
        return <ErrorPanel message={error ?? i18n('failedWithoutMessage')} />;
    }

    const stateLabels: Partial<Record<TaskState, string>> = {
        [TaskState.Unknown]: i18n('unknown'),
        [TaskState.Created]: i18n('created'),
        [TaskState.Running]: i18n('running'),
        [TaskState.Finished]: i18n('finished'),
    };

    const themes: {[key in TaskState]?: ProgressTheme} = {
        [TaskState.Unknown]: 'misc',
        [TaskState.Created]: 'misc',
        [TaskState.Running]: 'info',
        [TaskState.Finished]: 'success',
    };
    const theme = themes[state] ?? 'info';

    const progressPercentages: {[key in TaskState]?: number} = {
        [TaskState.Unknown]: 50,
        [TaskState.Created]: 20,
        [TaskState.Running]: 50,
        [TaskState.Finished]: 100,
    };
    const progressPercentage = progressPercentages[state] ?? 50;

    return (
        <Progress
            text={stateLabels[state] ?? state}
            loading={state !== TaskState.Finished}
            theme={theme}
            value={progressPercentage}
        />
    );
};
