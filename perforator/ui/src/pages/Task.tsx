import React from 'react';

import { useParams } from 'react-router-dom';

import { TaskCard } from 'src/components/TaskCard/TaskCard';
import { TaskReport } from 'src/components/TaskReport/TaskReport';
import type { TaskResult } from 'src/models/Task';
import { TaskState } from 'src/models/Task';
import { apiClient } from 'src/utils/api';

import type { Page } from './Page';


const POLLING_PERIOD = 1000;  // 1s

export const Task: Page = props => {
    const pollingInterval = React.useRef<number | undefined>(undefined);

    const { taskId } = useParams();
    const [task, setTask] = React.useState<TaskResult | null>(null);
    const [error, setError] = React.useState<Error | undefined>(undefined);

    const getTask = async () => {
        if (!pollingInterval.current) {
            return;
        }
        try {
            const response = await apiClient.getTask(taskId!);
            setTask(response?.data);
        } catch (e) {
            if (e instanceof Error) {
                setError(e);
            }
        }
    };

    React.useEffect(() => {
        getTask();

        // @ts-ignore
        pollingInterval.current = setInterval(() => {
            getTask();
        }, POLLING_PERIOD);

        return () => { clearInterval(pollingInterval.current); };
    }, [taskId]);

    const state = task?.Status?.State;
    const isFinished = state === TaskState.Finished || state === TaskState.Failed;
    if (isFinished || error) {
        clearInterval(pollingInterval.current);
        pollingInterval.current = undefined;
    }

    const taskCard = (state === TaskState.Finished && props.embed)
        ? null
        : (
            <TaskCard
                taskId={taskId!}
                task={task}
                error={error}
            />
        );
    const taskReport = state === TaskState.Finished
        ? (<TaskReport task={task} />)
        : null;

    return (
        <>
            {taskCard}
            {taskReport}
        </>
    );
};
