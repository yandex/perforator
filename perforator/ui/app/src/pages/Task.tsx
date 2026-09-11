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
import { buildProfileRum } from 'src/utils/buildProfileRum';
import { getFormat } from 'src/utils/renderingFormat';

import type { Page } from './Page';


const POLLING_PERIOD = 1000;  // 1s


export const Task: Page = (props) => {
    const { taskId } = useParams();
    // Reset polling and rendered data together so an old task cannot finish
    // the new task's build measurement while its first response is pending.
    return <TaskPage key={taskId} {...props} />;
};

const TaskPage: Page = ({ embed, header }) => {
    const pollingInterval = React.useRef<number | undefined>(undefined);

    const { taskId } = useParams();
    const [task, setTask] = React.useState<TaskResult | null>(null);
    const [error, setError] = React.useState<Error | undefined>(undefined);
    const attempt = buildProfileRum.forTask(taskId);

    const getTask = async (isActive: () => boolean) => {
        if (!pollingInterval.current) {
            return;
        }
        try {
            const response = await apiClient.getTask(taskId!);
            if (!isActive() || !pollingInterval.current) {
                return;
            }
            attempt?.setTaskFormat(getFormat(response.data.Spec?.MergeProfiles?.Format)
                ?? getFormat(response.data.Spec?.DiffProfiles?.RenderFormat));
            if (response.data.Status?.State === TaskState.Failed) {
                attempt?.finish('error');
            }
            setTask(response?.data);
        } catch (e) {
            if (isActive() && pollingInterval.current) {
                attempt?.finish('error');
                if (e instanceof Error) {
                    setError(e);
                }
            }
        }
    };

    React.useEffect(() => {
        let active = true;
        const isActive = () => active;
        pollingInterval.current = window.setInterval(() => {
            getTask(isActive);
        }, POLLING_PERIOD);

        getTask(isActive);

        return () => {
            active = false;
            clearInterval(pollingInterval.current);
        };
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
