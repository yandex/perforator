import React from 'react';

import type { ProgressTheme } from '@gravity-ui/uikit';
import { Progress } from '@gravity-ui/uikit';

import { TaskState } from 'src/models/Task';

import { ErrorPanel } from '../../ErrorPanel/ErrorPanel';


export interface TaskProgressProps {
    state: TaskState;
    error?: string;
}

export const TaskProgress: React.FC<TaskProgressProps> = props => {
    const { state } = props;

    if (state === TaskState.Failed || props.error) {
        return <ErrorPanel message={props.error ?? 'Task failed without error message'} />;
    }

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
            text={state}
            loading={state !== TaskState.Finished}
            theme={theme}
            value={progressPercentage}
        />
    );
};
