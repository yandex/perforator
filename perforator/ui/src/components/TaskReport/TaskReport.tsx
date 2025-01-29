import React from 'react';

import { Alert, Button, Loader } from '@gravity-ui/uikit';

import { uiFactory } from 'src/factory';
import type { RenderFormat } from 'src/generated/perforator/proto/perforator/perforator';
import type { TaskResult } from 'src/models/Task';

import { ErrorPanel } from '../ErrorPanel/ErrorPanel';

import { TaskFlamegraph } from './TaskFlamegraph/TaskFlamegraph';

import './TaskReport.scss';


export interface TaskReportProps {
    task: TaskResult | null;
}

function getWellKnownKeysFromObject<O extends Record<string, any>, K extends keyof O>(o: O, keys: K[]): K[] {
    return keys.filter(k => k in o);
}

const getFormatLike = (o: RenderFormat) => getWellKnownKeysFromObject(o, ['Flamegraph', 'JSONFlamegraph', 'RawProfile']);
const getFormat = (o?: RenderFormat) => o ? getFormatLike(o)[0] : undefined;

export const TaskReport: React.FC<TaskReportProps> = props => {
    const url = props.task?.Result?.MergeProfiles?.ProfileURL || props.task?.Result?.DiffProfiles?.ProfileURL;
    const isDiff = 'DiffProfiles' in (props.task?.Result || {});
    const mergeRenderFormat = props.task?.Spec?.MergeProfiles?.Format;
    const diffRenderFormat = props.task?.Spec?.DiffProfiles?.RenderFormat;
    const format = getFormat((props.task?.Spec?.MergeProfiles?.Format)) ?? getFormat(diffRenderFormat);

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
        <div className="task-report">
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
