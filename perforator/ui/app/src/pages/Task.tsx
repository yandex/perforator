import React from 'react';

import { useParams } from 'react-router-dom';

import { Loader } from '@gravity-ui/uikit';

import { Fullscreen } from 'src/components/Fullscreen/Fullscreen';
import { FullscreenProvider } from 'src/components/Fullscreen/FullscreenProvider';
import { TaskCard } from 'src/components/TaskCard/TaskCard';
import { TaskHeader } from 'src/components/TaskCard/TaskHeader';
import { TaskReport } from 'src/components/TaskReport/TaskReport';
import type { TaskResult } from 'src/models/Task';
import { TaskState } from 'src/models/Task';
import { apiClient } from 'src/utils/api';

import type { Page } from './Page';


const POLLING_PERIOD = 1000;  // 1s


export const Task: Page = ({ embed, header }) => {
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
        // @ts-ignore
        pollingInterval.current = setInterval(() => {
            getTask();
        }, POLLING_PERIOD);

        getTask();

        return () => { clearInterval(pollingInterval.current); };
    }, [taskId]);

    const state = task?.Status?.State;

    const isFinished = state === TaskState.Finished || state === TaskState.Failed;
    if (isFinished || error) {
        clearInterval(pollingInterval.current);
        pollingInterval.current = undefined;
    }

    const taskCard = (state === TaskState.Finished && embed)
        ? null
        : (
            <TaskCard
                taskId={taskId!}
                task={task}
                error={error}
            />
        );
    const taskReport = (task && state !== TaskState.Created && state !== TaskState.Running)
        ? (<TaskReport taskId={taskId} task={task} />)
        : <Loader/>;

    const timeline = (task) ? <TaskHeader
        task={task}
        embed={embed}
        header={header}
    /> : null;

    return (
        <FullscreenProvider initialEnalbed={true}>
            <Fullscreen>
                {timeline}
                {taskCard}
                {taskReport}
            </Fullscreen>
        </FullscreenProvider>
    );
};
