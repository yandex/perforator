import React from 'react';

import { supportsPprofWasm, unsupportedPprofWasm } from './capabilities';
import type { PprofRenderOptions } from './options';
import { pprofOptionsKey } from './options';
import type { PprofRenderResult } from './protocol';
import type { PprofStage } from './session';
import { PprofSession } from './session';


interface State {
    result?: PprofRenderResult;
    options?: PprofRenderOptions;
    error?: Error;
    busy: boolean;
    stage?: PprofStage;
    cancelled: boolean;
    revision: number;
    startedAt?: number;
}

export function usePprofProfile(source: string | Blob, {
    createWorker,
    engine,
    requested,
    optionsIdentity,
    onCommit,
    onStartRequest,
    onResult,
    onNavigationCancel,
}: {
    createWorker: () => Worker;
    engine?: Blob;
    requested: PprofRenderOptions | undefined;
    optionsIdentity: string;
    onCommit: (options: PprofRenderOptions) => void;
    onStartRequest?: () => void;
    onResult?: () => void;
    onNavigationCancel?: () => void;
}) {
    const sessionRef = React.useRef<PprofSession>();
    const [state, setState] = React.useState<State>({ busy: false, cancelled: false, revision: 0 });
    const stateRef = React.useRef(state);
    stateRef.current = state;
    const generation = React.useRef(0);
    const commitRef = React.useRef(onCommit);
    commitRef.current = onCommit;
    const onStartRequestRef = React.useRef(onStartRequest);
    onStartRequestRef.current = onStartRequest;
    const onResultRef = React.useRef(onResult);
    onResultRef.current = onResult;
    const onNavigationCancelRef = React.useRef(onNavigationCancel);
    onNavigationCancelRef.current = onNavigationCancel;
    const lastRequest = React.useRef<{ options: PprofRenderOptions; commit: boolean }>();

    React.useEffect(() => {
        setState({ busy: false, cancelled: false, revision: 0 });
        stateRef.current = { busy: false, cancelled: false, revision: 0 };
        lastRequest.current = undefined;
        const session = new PprofSession(source, createWorker, engine);
        sessionRef.current = session;
        return () => {
            // This is a request counter, not a DOM ref: invalidate all pending completions.
            // eslint-disable-next-line react-hooks/exhaustive-deps
            ++generation.current;
            session.dispose();
            sessionRef.current = undefined;
        };
    }, [source, createWorker, engine]);

    const render = React.useCallback(async (options: PprofRenderOptions, request: { commit?: boolean; force?: boolean } = {}) => {
        const { commit: shouldCommit = false, force = false } = request;
        const key = pprofOptionsKey(options);
        const shown = stateRef.current;
        if (!force && !shown.busy && shown.result && !shown.error && !shown.cancelled &&
            shown.options && pprofOptionsKey(shown.options) === key) {
            return;
        }
        const current = ++generation.current;
        const startedAt = performance.now();
        lastRequest.current = { options, commit: shouldCommit };
        setState(previous => ({ ...previous, busy: true, stage: 'loading', cancelled: false, error: undefined, startedAt }));
        try {
            // React Strict Mode replays effects before this microtask. Do not
            // start a download for an effect that has already been disposed.
            await Promise.resolve();
            if (current !== generation.current) {
                return;
            }
            onStartRequestRef.current?.();
            if (!supportsPprofWasm()) {
                throw new Error(unsupportedPprofWasm);
            }
            const session = sessionRef.current;
            if (!session) {
                throw new Error('The profile session is not active.');
            }
            const result = await session.render(options, stage => {
                if (current === generation.current) {
                    setState(previous => ({ ...previous, stage }));
                }
            });
            if (current !== generation.current) {
                return;
            }
            onResultRef.current?.();
            if (shouldCommit) {
                commitRef.current(options);
            }
            setState(previous => ({ result, options, busy: false, stage: undefined, cancelled: false, startedAt,
                revision: previous.revision + 1 }));
        } catch (error) {
            if (current === generation.current) {
                const failure = error instanceof Error ? error : new Error(String(error));
                const wasCancelled = failure.name === 'AbortError';
                setState(previous => ({ ...previous, busy: false, stage: undefined,
                    cancelled: wasCancelled, error: wasCancelled ? undefined : failure,
                }));
            }
        }
    }, []);

    const cancel = React.useCallback(() => {
        ++generation.current;
        sessionRef.current?.cancel();
        setState(previous => ({ ...previous, busy: false, stage: undefined, cancelled: true, error: undefined }));
    }, []);

    const cancelForNavigation = () => {
        ++generation.current;
        sessionRef.current?.cancel();
        setState(previous => ({ ...previous, busy: false, stage: undefined, cancelled: false, error: undefined }));
    };

    const restore = () => {
        setState(previous => ({ ...previous, cancelled: false, error: undefined }));
    };

    React.useEffect(() => {
        const requestedKey = requested && pprofOptionsKey(requested);
        const pendingKey = lastRequest.current && pprofOptionsKey(lastRequest.current.options);
        if (stateRef.current.busy && pendingKey !== requestedKey) {
            onNavigationCancelRef.current?.();
            cancelForNavigation();
        }
        const shownKey = stateRef.current.options && pprofOptionsKey(stateRef.current.options);
        if (requested && shownKey !== requestedKey) {
            render(requested);
        } else if (requested && (stateRef.current.error || stateRef.current.cancelled)) {
            restore();
        }
        // Search and zoom query changes do not change optionsIdentity.
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [source, createWorker, engine, optionsIdentity]);

    const retry = React.useCallback(() => {
        if (lastRequest.current) {
            render(lastRequest.current.options, { commit: lastRequest.current.commit });
        }
    }, [render]);

    const requestedKey = requested && pprofOptionsKey(requested);
    const shownKey = state.options && pprofOptionsKey(state.options);
    const stale = Boolean(state.result && (state.busy || state.error || state.cancelled || !requestedKey ||
        shownKey !== requestedKey));

    return { ...state, render, cancel, retry, stale };
}
