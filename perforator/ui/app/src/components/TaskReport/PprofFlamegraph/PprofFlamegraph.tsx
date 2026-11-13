import React from 'react';

import { createPortal } from 'react-dom';
import { useParams, useSearchParams } from 'react-router-dom';

import { prerenderColors, RenderSettings } from '@perforator/flamegraph';
import { unsupportedPprofWasm } from '@perforator/flamegraph/dist/pprof/capabilities';
import type { PprofRenderOptions } from '@perforator/flamegraph/dist/pprof/options';
import { defaultPprofOptions, pprofQueryKeys, readPprofOptions, writePprofOptions } from '@perforator/flamegraph/dist/pprof/options';
import { usePprofProfile } from '@perforator/flamegraph/dist/pprof/usePprofProfile';

import { ArrowDown } from '@gravity-ui/icons';
import { Alert, Button, Icon, Loader, useThemeType } from '@gravity-ui/uikit';

import { uiFactory } from 'src/factory';
import { useUserSettings } from 'src/providers/UserSettingsProvider/UserSettingsContext';
import { buildProfileRum } from 'src/utils/buildProfileRum';
import { cn } from 'src/utils/cn';
import { withMeasureTime } from 'src/utils/logging';

import { Visualisation } from '../Visualisation/Visualisation';

import i18n from './i18n';

import './PprofFlamegraph.scss';


const createWorker = () => new Worker(
    new URL('../../../utils/pprof/profile.worker.ts', import.meta.url), { type: 'module' },
);

const b = cn('pprof-flamegraph');

export function PprofFlamegraph({ url, headerActions, onRetryLegacy }: {
    url: string;
    headerActions?: HTMLDivElement | null;
    onRetryLegacy?: (options: PprofRenderOptions) => void;
}) {
    const { taskId } = useParams();
    const attempt = buildProfileRum.forTask(taskId);
    const [query, setQuery] = useSearchParams();
    const theme = useThemeType();
    const { userSettings } = useUserSettings();
    const pageName = query.get('tab') === 'top' || query.get('tab') === 'sbs' ? 'top-table' : 'task-flamegraph';
    const renderingPage = React.useRef<string | undefined>(undefined);
    const completedRevision = React.useRef(0);
    const reportedFailureAt = React.useRef<number>();
    const commit = (options: PprofRenderOptions) => setQuery(previous => {
        const next = writePprofOptions(new URLSearchParams(previous), options);
        for (const key of ['frameDepth', 'framePos', 'omittedIndexes']) {
            next.delete(key);
        }
        return next;
    });
    let requested: PprofRenderOptions | undefined;
    let queryError: Error | undefined;
    try {
        requested = readPprofOptions(query);
    } catch (error) {
        queryError = error instanceof Error ? error : new Error(String(error));
    }
    const optionsInUrl = JSON.stringify(pprofQueryKeys.map(key => query.get(key)));
    const profile = usePprofProfile(url, {
        createWorker,
        requested,
        optionsIdentity: optionsInUrl,
        onCommit: commit,
        onStartRequest: () => {
            renderingPage.current = pageName;
            uiFactory().rum()?.makeSpaSubPage?.(pageName, undefined, undefined, false,
                { additional: encodeURIComponent(JSON.stringify({ flamegraphFormat: 'RawProfile', renderEngine: 'wasm' })) });
        },
        onResult: () => {
            const currentPage = renderingPage.current;
            if (currentPage) {
                uiFactory().rum()?.finishDataLoading?.(currentPage);
                uiFactory().rum()?.sendResTiming?.(url);
                uiFactory().rum()?.startDataRendering?.(currentPage, '', false);
            }
        },
    });
    const data = React.useMemo(() => {
        if (!profile.result) {
            return null;
        }
        const framesCount = profile.result.profile.rows.reduce((count, row) => count + row.length, 0);
        return withMeasureTime(prerenderColors, 'prerenderColors', ms =>
            uiFactory().rum()?.sendDelta?.('prerenderColors', ms, { additional: { framesCount, renderEngine: 'wasm' } }),
        )(profile.result.profile, { theme });
    }, [profile.result, theme]);
    React.useEffect(() => {
        if (profile.result) {
            const { metrics } = profile.result;
            uiFactory().rum()?.sendDelta?.('wasm-render', metrics.renderMs, { additional: { ...metrics } });
        }
    }, [profile.result]);
    const error = queryError ?? profile.error;
    const errorMessage = error?.message;
    const retryable = Boolean(profile.error && profile.error.message !== unsupportedPprofWasm && !queryError);
    const supportLink = error ? uiFactory().bugReportLink() : undefined;
    React.useEffect(() => {
        if (errorMessage || profile.cancelled) {
            attempt?.finish(errorMessage ? 'error' : 'cancelled');
        }
    }, [attempt, errorMessage, profile.cancelled]);
    React.useEffect(() => {
        if ((profile.error || profile.cancelled) && profile.startedAt !== undefined &&
            reportedFailureAt.current !== profile.startedAt) {
            reportedFailureAt.current = profile.startedAt;
            renderingPage.current = undefined;
            uiFactory().rum()?.sendDelta?.(profile.cancelled ? 'wasm-render.cancelled' : 'wasm-render.error',
                performance.now() - profile.startedAt, { additional: { renderEngine: 'wasm' } });
        }
    }, [profile.error, profile.cancelled, profile.startedAt]);

    const renderOptions = profile.options ?? requested ?? defaultPprofOptions;
    const loadingText = profile.stage === 'processing' ? i18n('processing') : i18n('loading');
    const loadingPreviousText = profile.stage === 'processing' ? i18n('processingPrevious') : i18n('loadingPrevious');
    const stale = profile.stale;
    let status = '';
    if (profile.busy) {
        status = data ? loadingPreviousText : loadingText;
    } else if (profile.cancelled) {
        status = data ? i18n('cancelledPrevious') : i18n('cancelled');
    } else if (stale) {
        status = i18n('previousRender');
    }
    const resetRenderOptions = () => setQuery(previous => {
        const next = new URLSearchParams(previous);
        for (const key of pprofQueryKeys) {
            next.delete(key);
        }
        return next;
    });
    const actions = <>
        <RenderSettings options={renderOptions} busy={profile.busy}
            onApply={options => {
                profile.render(options, { commit: true, force: Boolean(queryError) });
            }} />
        <Button href={url}>
            <Icon data={ArrowDown} />
            {i18n('downloadPprof')}
        </Button>
    </>;

    return <div className="pprof-flamegraph">
        {headerActions && createPortal(actions, headerActions)}
        {!headerActions && <div className="pprof-flamegraph__controls">{actions}</div>}
        <div className={b('progress', { overlay: Boolean(data) })}>
            <div role="status" className="pprof-flamegraph__status">{status}</div>
            {(profile.busy || profile.cancelled) && <div className="pprof-flamegraph__controls">
                {profile.busy && <>
                    <Loader size="s" />
                    <Button onClick={profile.cancel}>{i18n('cancel')}</Button>
                </>}
                {profile.cancelled && !profile.busy && <Button onClick={profile.retry}>{i18n('retry')}</Button>}
            </div>}
        </div>
        {error && <div role="alert">
            <Alert theme="danger" title={i18n('renderError')} message={error.message} actions={<Alert.Actions>
                {retryable && <Alert.Action onClick={profile.retry}>{i18n('retry')}</Alert.Action>}
                {queryError && <Alert.Action onClick={resetRenderOptions}>{i18n('resetOptions')}</Alert.Action>}
                {onRetryLegacy && <Alert.Action onClick={() => onRetryLegacy(renderOptions)}>
                    {i18n('retryLegacy')}
                </Alert.Action>}
                {supportLink && <Alert.Action href={supportLink} target="_blank" rel="noopener noreferrer">
                    {i18n('contactSupport')}
                </Alert.Action>}
            </Alert.Actions>} />
        </div>}
        {data && <Visualisation key={profile.revision} profileData={data} loading={false} isDiff={false}
            showLineNumbers={renderOptions.showLineNumbers}
            disableLineNumbers={profile.busy}
            setShowLineNumbers={showLineNumbers => {
                uiFactory().reachGoal('LINE_NUMBERS');
                profile.render({ ...renderOptions, showLineNumbers }, { commit: true });
            }}
            onFinishRendering={() => {
                if (!profile.busy && completedRevision.current !== profile.revision && renderingPage.current) {
                    completedRevision.current = profile.revision;
                    uiFactory().rum()?.finishDataRendering?.(renderingPage.current);
                    renderingPage.current = undefined;
                    if (profile.startedAt !== undefined) {
                        uiFactory().rum()?.sendDelta?.('wasm-render-e2e', performance.now() - profile.startedAt,
                            { additional: { renderEngine: 'wasm', renderKind: profile.revision === 1 ? 'initial' : 'apply' } });
                    }
                    attempt?.finish();
                }
            }}
            theme={theme} userSettings={userSettings} />}
    </div>;
}
