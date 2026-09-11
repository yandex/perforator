import type { Location } from 'react-router-dom';
import { matchPath } from 'react-router-dom';

import { uiFactory } from 'src/factory';

import type { Rum } from './rum';


type BuildLocation = Pick<Location, 'key' | 'pathname' | 'search'>;

interface BuildProfileAttempt {
    key: string;
    taskId: string | undefined;
    setTaskFormat: (format: string | undefined) => void;
    finish: (outcome?: Outcome) => void;
}

type Outcome = 'error' | 'abandoned';

// One attempt spans /build and its task redirect. Keep its identity so late
// promises from an earlier navigation cannot finish a newer measurement.
export class BuildProfileRum {
    private locationKey?: string;
    private active?: BuildProfileAttempt;

    private readonly getRum: () => Rum;

    constructor(getRum: () => Rum) {
        this.getRum = getRum;
    }

    navigate(location: BuildLocation) {
        if (location.key === this.locationKey) {
            return;
        }
        this.locationKey = location.key;
        if (this.active?.taskId && matchPath('/task/:taskId', location.pathname)?.params.taskId === this.active.taskId) {
            return;
        }
        this.active?.finish('abandoned');
        this.active = matchPath('/build', location.pathname) ? this.start(location) : undefined;
    }

    forBuild(key: string) {
        return this.active?.key === key ? this.active : undefined;
    }

    forTask(taskId: string | undefined) {
        return taskId && this.active?.taskId === taskId ? this.active : undefined;
    }

    error() {
        this.active?.finish('error');
    }

    private start(location: BuildLocation): BuildProfileAttempt {
        const query = new URLSearchParams(location.search);
        const rum = this.getRum();
        const page = 'build-flamegraph';
        const dimensions = {
            profileKind: query.has('diffSelector') ? 'diff' : 'merge',
            renderEngine: 'unknown',
            tab: query.get('tab') ?? 'flame',
        };
        const started = performance.now();
        const subpage = rum.makeSpaSubPage?.(page, { finishDataLoadingMetric: true }, false, false, {
            additional: encodeURIComponent(JSON.stringify(dimensions)),
        });
        let finished = false;
        const leave = () => attempt.finish('abandoned');
        const attempt: BuildProfileAttempt = {
            key: location.key,
            taskId: undefined,
            setTaskFormat: (format: string | undefined) => {
                if (finished || !format) {
                    return;
                }
                if (format !== 'JSONFlamegraph') {
                    finished = true;
                    window.removeEventListener('pagehide', leave);
                    return;
                }
                dimensions.renderEngine = 'backend';
                if (subpage) {
                    subpage.additional = encodeURIComponent(JSON.stringify(dimensions));
                }
            },
            finish: (outcome?: Outcome) => {
                if (finished) {
                    return;
                }
                finished = true;
                window.removeEventListener('pagehide', leave);
                if (outcome) {
                    rum.sendDelta?.(`build.${outcome}`, performance.now() - started, subpage);
                } else {
                    // This is deliberately the navigation-to-render boundary,
                    // including task creation, polling, download and rendering of
                    // the requested tab (canvas for flame, table commit for top/sbs).
                    rum.finishDataLoading?.(page);
                }
            },
        };
        // Best effort: rum-counter batches sends behind a 15 ms timer, which may
        // never run on tab close. For abandonment totals, use server-side counts
        // of created - success - error for completed cohorts, not this event.
        // visibilitychange is not an exit: builds can continue in background tabs.
        window.addEventListener('pagehide', leave);
        return attempt;
    }
}

export const buildProfileRum = new BuildProfileRum(() => uiFactory().rum());
