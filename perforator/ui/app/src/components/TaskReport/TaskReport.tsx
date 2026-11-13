import React from 'react';

import { useNavigate } from 'react-router-dom';

import { Alert, Button, Loader } from '@gravity-ui/uikit';

import { uiFactory } from 'src/factory';
import type { ProfileTaskQuery, TaskResult } from 'src/models/Task';
import { useUserSettings } from 'src/providers/UserSettingsProvider';
import { boolToString } from 'src/utils/bool';
import { buildProfileRum } from 'src/utils/buildProfileRum';
import { cn } from 'src/utils/cn';
import { defaultProfileTaskQuery, redirectToTaskPage, taskFormatToQueryFormat } from 'src/utils/profileTask';
import { preserveWellKnownQueryParams } from 'src/utils/profileTask/preserveWellKnown';
import { getFormat, isDiffTaskResult } from 'src/utils/renderingFormat';

import { ErrorPanel } from '../ErrorPanel/ErrorPanel';
import { useFullscreen } from '../Fullscreen/FullscreenContext';
import { navigateToLineNumbers } from '../TaskCard/navigateToLineNumbers';

import i18n from './i18n';
import { PprofFlamegraph } from './PprofFlamegraph/PprofFlamegraph';
import { TaskFlamegraph } from './TaskFlamegraph/TaskFlamegraph';
import { TextProfile } from './TextProfile/TextProfile';

import './TaskReport.scss';


export interface TaskReportProps {
    task: TaskResult | null;
    taskId?: string;
    headerActions?: HTMLDivElement | null;
}

const b = cn('task-report');

export const TaskReport: React.FC<TaskReportProps> = ({ task, taskId, headerActions }: TaskReportProps) => {
    const url = task?.Result?.MergeProfiles?.ProfileURL || task?.Result?.DiffProfiles?.ProfileURL;
    const { enabled: fullscreen } = useFullscreen();
    const { userSettings } = useUserSettings();
    const mergeIntoPprofByDefault = uiFactory().wasmEnabled() && userSettings.mergeIntoPprofByDefault;

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
                maxProfiles: query?.MaxSamples || spec?.MaxSamples || diffspec?.BaselineQuery?.MaxSamples,
                lineNumbers: boolToString(!areLineNumbersEnabled),
                format: taskFormatToQueryFormat(task),
            } as ProfileTaskQuery,
            taskId,
            );
        }, [areLineNumbersEnabled, navigate, task, taskId, query?.Selector, query?.MaxSamples, spec?.MaxSamples, diffspec]);

    React.useEffect(() => {
        if (url && format === 'RawProfile' && (isDiff || !mergeIntoPprofByDefault)) {
            // Download-only tasks have no navigation-to-render measurement.
            buildProfileRum.forTask(taskId)?.skip();
        } else if (!url || (format !== 'JSONFlamegraph' && (isDiff || format !== 'RawProfile'))) {
            buildProfileRum.forTask(taskId)?.finish('error');
        }
    }, [taskId, url, isDiff, format, mergeIntoPprofByDefault]);

    const renderContent = () => {
        if (!url) {
            return <ErrorPanel message={i18n('missingProfile')} />;
        }
        if (format === 'RawProfile') {
            if (isDiff || !mergeIntoPprofByDefault) {
                const message = (
                    <div>
                        <div>
                            {i18n('rawProfile')}
                        </div>
                        <Button className="task-report__download-raw" href={url}>{i18n('download')}</Button>
                    </div>
                );

                return <Alert
                    className={b('alert')}
                    theme="info"
                    view="outlined"
                    title={i18n('nothingToShow')}
                    message={message}
                />;
            }
            return <PprofFlamegraph key={taskId} url={url} headerActions={headerActions}
                onRetryLegacy={query?.Selector ? options => redirectToTaskPage(navigate, {
                    ...defaultProfileTaskQuery(),
                    ...Object.fromEntries(preserveWellKnownQueryParams(new URLSearchParams(window.location.search))),
                    selector: query.Selector,
                    maxProfiles: query.MaxSamples || spec?.MaxSamples || uiFactory().defaultSampleSize(),
                    lineNumbers: boolToString(options.showLineNumbers),
                    format: 'json',
                }) : undefined} />;
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
                title={i18n('error')}
                message={`${i18n('unknownFormat')} ${JSON.stringify(mergeRenderFormat || diffRenderFormat)}`}
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
        <div className={b({ iframe: true })}>
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
