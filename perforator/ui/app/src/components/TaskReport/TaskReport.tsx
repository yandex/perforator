import React from 'react';

import { Alert, Button, Loader } from '@gravity-ui/uikit';

import { uiFactory } from 'src/factory';
import type { TaskResult } from 'src/models/Task';
import { cn } from 'src/utils/cn';
import { getFormat, isDiffTaskResult } from 'src/utils/renderingFormat';

import { ErrorPanel } from '../ErrorPanel/ErrorPanel';
import { useFullscreen } from '../Fullscreen/FullscreenContext';

import { TaskFlamegraph } from './TaskFlamegraph/TaskFlamegraph';
import { TextProfile } from './TextProfile/TextProfile';

import './TaskReport.scss';


export interface TaskReportProps {
    task: TaskResult | null;
}

const b = cn('task-report');

export const TaskReport: React.FC<TaskReportProps> = props => {
    const url = props.task?.Result?.MergeProfiles?.ProfileURL || props.task?.Result?.DiffProfiles?.ProfileURL;
    const { enabled: fullscreen } = useFullscreen();

    const isDiff = isDiffTaskResult(props.task);
    const mergeRenderFormat = props.task?.Spec?.MergeProfiles?.Format;
    const diffRenderFormat = props.task?.Spec?.DiffProfiles?.RenderFormat;
    const isLegacyFormat = isDiff && 'FlamegraphOptions' in (props.task?.Spec?.DiffProfiles || {});
    const format = getFormat(mergeRenderFormat) ?? getFormat(diffRenderFormat) ?? (isLegacyFormat ? 'Flamegraph' : undefined);

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
                theme="info"
                view="outlined"
                title="Nothing to show there"
                message={message}
            />;
        }

        if (format === 'Flamegraph' && !uiFactory().parseLegacyFormat) {
            return <IFrameReport url={url}/>;
        }

        if (format === 'TextProfile') {
            return <TextProfile url={url} />;
        }

        if (!format) {
            return <Alert
                theme="danger"
                view="outlined"
                title="Error"
                message={`Unknown format in ${JSON.stringify(mergeRenderFormat || diffRenderFormat)}`}
            />;
        }

        // maybe better to split it into two components for each format
        return <TaskFlamegraph format={format} url={url} isDiff={isDiff} />;

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

