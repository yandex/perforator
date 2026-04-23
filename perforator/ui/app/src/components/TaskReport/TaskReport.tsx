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
import { useFetchResult } from './TaskFlamegraph/useFetchResult';
import { TextProfile } from './TextProfile/TextProfile';

import './TaskReport.scss';


export interface TaskReportProps {
    task: TaskResult | null;
    taskId?: string;
}

const b = cn('task-report');

export const TaskReport: React.FC<TaskReportProps> = props => {
    const url = props.task?.Result?.MergeProfiles?.ProfileURL || props.task?.Result?.DiffProfiles?.ProfileURL;
    const { enabled: fullscreen } = useFullscreen();

    const spec = props.task?.Spec?.MergeProfiles;
    const diffspec = props.task?.Spec?.DiffProfiles;
    const query = spec?.Query;

    const isDiff = isDiffTaskResult(props.task);
    const mergeRenderFormat = props.task?.Spec?.MergeProfiles?.Format;
    const diffRenderFormat = props.task?.Spec?.DiffProfiles?.RenderFormat;
    const isLegacyFormat = isDiff && 'FlamegraphOptions' in (props.task?.Spec?.DiffProfiles || {});
    const format = getFormat(mergeRenderFormat) ?? getFormat(diffRenderFormat) ?? (isLegacyFormat ? 'Flamegraph' : undefined);
    const formatField = spec?.Format?.JSONFlamegraph ?? spec?.Format?.Flamegraph;
    const areLineNumbersEnabled = formatField?.ShowLineNumbers ?? diffspec?.RenderFormat?.JSONFlamegraph?.ShowLineNumbers ?? false;
    const navigate = useNavigate();
    const handleLineNumbers = React.useCallback(
        () => {
            if (!props.taskId) {
                return;
            }
            navigateToLineNumbers(navigate, {
                selector: query?.Selector ?? diffspec?.BaselineQuery?.Selector,
                diffSelector: diffspec?.DiffQuery?.Selector,
                maxProfiles: spec?.MaxSamples! ?? diffspec?.BaselineQuery?.MaxSamples,
                lineNumbers: boolToString(!areLineNumbersEnabled),
            } as ProfileTaskQuery,
            props.taskId,
            );
        }, [areLineNumbersEnabled, navigate, props.taskId, query?.Selector, spec?.MaxSamples]);

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


    const extractData = React.useMemo(() => {
        return async (req: Response) => {
            return req.text().then((data) => {
                const blob = new Blob([data], { type: 'text/html' });
                const blobUrl = URL.createObjectURL(blob);
                return blobUrl;
            });
        };
    }, []);


    const { data: data, error, loading } = useFetchResult<string>({ url: url, extractData: extractData,
    });
    return (
        <div className={b({ iframe: true })}>
            {loading ? <Loader /> : null}
            {error ? <ErrorPanel message={error?.message} /> : null}
            <iframe
                id='profile'
                src={data}
                style={{
                    width: '100%',
                    height: '4200px',
                    border: 0,
                }}
            />
        </div>
    );
};
