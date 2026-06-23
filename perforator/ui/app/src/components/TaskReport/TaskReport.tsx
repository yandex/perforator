import React from 'react';

import { useNavigate } from 'react-router-dom';

import { Alert, Button, Loader } from '@gravity-ui/uikit';

import { uiFactory } from 'src/factory';
import type { ProfileTaskQuery, TaskResult } from 'src/models/Task';
import { boolToString } from 'src/utils/bool';
import { cn } from 'src/utils/cn';
import { getFormat, isDiffTaskResult } from 'src/utils/renderingFormat';

import { ErrorPanel } from '../ErrorPanel/ErrorPanel';
import { useFullscreen } from '../Fullscreen/FullscreenContext';
import { navigateToLineNumbers } from '../TaskCard/navigateToLineNumbers';

import { TaskFlamegraph } from './TaskFlamegraph/TaskFlamegraph';
import { TextProfile } from './TextProfile/TextProfile';

import './TaskReport.scss';


export interface TaskReportProps {
    task: TaskResult | null;
    taskId?: string;
}

const b = cn('task-report');

export const TaskReport: React.FC<TaskReportProps> = ({ task, taskId }: TaskReportProps) => {
    const url = task?.Result?.MergeProfiles?.ProfileURL || task?.Result?.DiffProfiles?.ProfileURL;
    const { enabled: fullscreen } = useFullscreen();

    const spec = task?.Spec?.MergeProfiles;
    const diffspec = task?.Spec?.DiffProfiles;
    const query = spec?.Query;

    const isDiff = isDiffTaskResult(task);
    const mergeRenderFormat = task?.Spec?.MergeProfiles?.Format;
    const diffRenderFormat = task?.Spec?.DiffProfiles?.RenderFormat;
    const isLegacyFormat = isDiff && 'FlamegraphOptions' in (task?.Spec?.DiffProfiles || {});
    const format = getFormat(mergeRenderFormat) ?? getFormat(diffRenderFormat) ?? (isLegacyFormat ? 'Flamegraph' : undefined);
    const formatField = spec?.Format?.JSONFlamegraph ?? spec?.Format?.Flamegraph;
    const areLineNumbersEnabled = formatField?.ShowLineNumbers ?? diffspec?.RenderFormat?.JSONFlamegraph?.ShowLineNumbers ?? false;
    const navigate = useNavigate();
    const handleLineNumbers = React.useCallback(
        () => {
            if (!taskId) {
                return;
            }
            navigateToLineNumbers(navigate, {
                selector: query?.Selector ?? diffspec?.BaselineQuery?.Selector,
                diffSelector: diffspec?.DiffQuery?.Selector,
                maxProfiles: spec?.MaxSamples! ?? diffspec?.BaselineQuery?.MaxSamples,
                lineNumbers: boolToString(!areLineNumbersEnabled),
            } as ProfileTaskQuery,
            taskId,
            );
        }, [areLineNumbersEnabled, navigate, taskId, query?.Selector, spec?.MaxSamples]);

    const renderContent = () => {
        if (!url) {
            return <ErrorPanel message="Task finished without profile" />;
        }
        if (format === 'RawProfile') {
            const message = (
                <div>
                    <div>
                        Task finished with a raw pprof profile
                    </div>
                    <Button className="task-report__download-raw" href={url}>Download</Button>
                </div>
            );

            return <Alert
                className={b('alert')}
                theme="info"
                view="outlined"
                title="Nothing to show there"
                message={message}
            />;
        }

        if (format === 'HTMLVisualisation') {
            return <IFrameReport url={url}/>;
        }

        if (format === 'Flamegraph' && !uiFactory().parseLegacyFormat) {
            return <IFrameReport url={url}/>;
        }

        if (format === 'TextProfile') {
            return <TextProfile url={url} />;
        }

        if (!format) {
            return <Alert
                className={b('alert')}
                theme="danger"
                view="outlined"
                title="Error"
                message={`Unknown format in ${JSON.stringify(mergeRenderFormat || diffRenderFormat)}`}
            />;
        }

        return <TaskFlamegraph
            format={format}
            url={url}
            isDiff={isDiff}
            lineNumbers={areLineNumbersEnabled}
            onLineNumbersChange={handleLineNumbers}
        />;

    };

    return (
        <div className={b({ fullscreen })}>
            {renderContent()}
        </div>
    );
};


export interface IFrameReportProps {
    url: string;
}

export const IFrameReport: React.FC<IFrameReportProps> = ({ url }) => {
    const [loaded, setLoaded] = React.useState(false);

    return (
        <div className="task-report">
            {!loaded ? <Loader /> : null}
            <iframe
                id='profile'
                src={url}
                style={{
                    width: '100%',
                    height: '4200px',
                    border: 0,
                }}
                onLoad={() => setLoaded(true)}
            />
        </div>
    );
};
