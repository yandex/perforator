import { readFileSync } from 'node:fs';
import { runInNewContext } from 'node:vm';

import { afterEach, describe, expect, it, jest } from '@jest/globals';

import { BuildProfileRum } from './buildProfileRum';
import type { Rum } from './rum';


afterEach(() => {
    window.dispatchEvent(new Event('pagehide'));
    expect(console.error).not.toHaveBeenCalled();
    jest.restoreAllMocks();
});

describe('BuildProfileRum', () => {
    it('should measure the entire build through the task redirect once', () => {
        const { tracker, metrics, clock } = setup();
        tracker.navigate(routeLocation('build', '/build', ''));
        const attempt = tracker.forBuild('build')!;
        attempt.taskId = 'created-task';
        clock.now = 125;
        tracker.navigate(routeLocation('task', '/task/created-task', ''));
        attempt.setTaskFormat('JSONFlamegraph');
        // Changes to zoom do not create another build attempt.
        tracker.navigate(routeLocation('zoom', '/task/created-task', '?frameDepth=2'));
        clock.now = 825;
        tracker.forTask('created-task')!.finish();
        attempt.finish();
        attempt.finish('error');
        window.dispatchEvent(new Event('pagehide'));
        expect(metrics).toEqual([{ name: 'data.load.finish.network', value: 800, page: 'page.build-flamegraph', additional: { profileKind: 'merge', renderEngine: 'backend', tab: 'flame' } }]);
    });

    it.each(['error', 'abandoned'] as const)('should keep %s out of successful latency', outcome => {
        const { tracker, metrics, clock } = setup();
        tracker.navigate(routeLocation('build', '/build'));
        clock.now = 125;
        tracker.forBuild('build')!.finish(outcome);
        tracker.forBuild('build')!.finish();
        expect(metrics).toEqual([{ name: `build.${outcome}`, value: 100, page: 'page.build-flamegraph', additional: { profileKind: 'merge', renderEngine: 'unknown', tab: 'flame' } }]);
    });

    it('should abandon a pending build on navigation and ignore its late completion', () => {
        const { tracker, metrics, clock } = setup();
        tracker.navigate(routeLocation('old', '/build'));
        const old = tracker.forBuild('old')!;
        clock.now = 75;
        tracker.navigate(routeLocation('new', '/build', ''));
        old.finish();
        old.finish('error');
        old.taskId = 'old-task';
        expect(tracker.forTask('old-task')).toBeUndefined();
        clock.now = 175;
        tracker.forBuild('new')!.finish();
        expect(metrics).toEqual([
            { name: 'build.abandoned', value: 50, page: 'page.build-flamegraph', additional: { profileKind: 'merge', renderEngine: 'unknown', tab: 'flame' } },
            { name: 'data.load.finish.network', value: 100, page: 'page.build-flamegraph', additional: { profileKind: 'merge', renderEngine: 'unknown', tab: 'flame' } },
        ]);
    });

    it('should abandon when navigating to an unrelated task', () => {
        const { tracker, metrics } = setup();
        tracker.navigate(routeLocation('build', '/build'));
        tracker.forBuild('build')!.taskId = 'expected';
        tracker.navigate(routeLocation('other', '/task/other'));
        expect(metrics.map(metric => metric.name)).toEqual(['build.abandoned']);
        expect(tracker.forTask('other')).toBeUndefined();
    });

    it('should report page closure once and leave completed/error attempts alone', () => {
        const { tracker, metrics, clock } = setup();
        tracker.navigate(routeLocation('build', '/build'));
        clock.now = 225;
        window.dispatchEvent(new Event('pagehide'));
        window.dispatchEvent(new Event('pagehide'));
        tracker.forBuild('build')!.finish();
        expect(metrics).toEqual([{ name: 'build.abandoned', value: 200, page: 'page.build-flamegraph', additional: { profileKind: 'merge', renderEngine: 'unknown', tab: 'flame' } }]);
    });

    it('should not start another measurement for router updates at the same location', () => {
        const { tracker, metrics } = setup();
        tracker.navigate(routeLocation('build', '/build'));
        tracker.navigate(routeLocation('build', '/build'));
        tracker.error();
        tracker.error();
        expect(metrics.map(metric => metric.name)).toEqual(['build.error']);
    });

    it.each(['flame', 'top', 'sbs'])('should report diff profiles and the requested %s tab', tab => {
        const { tracker, metrics } = setup();
        tracker.navigate(routeLocation('build', '/build', `?diffSelector=service%3Dother&tab=${tab}`));
        const attempt = tracker.forBuild('build')!;
        attempt.taskId = 'json';
        tracker.navigate(routeLocation('task', '/task/json'));
        attempt.setTaskFormat('JSONFlamegraph');
        attempt.finish();
        expect(metrics[0].additional).toEqual({ profileKind: 'diff', renderEngine: 'backend', tab });
    });

    it.each(['RawProfile', 'TextProfile'])('should exclude %s without a task-page renderer', format => {
        const { tracker, metrics } = setup();
        tracker.navigate(routeLocation('build', '/build', '?format=raw'));
        const attempt = tracker.forBuild('build')!;
        attempt.taskId = 'download';
        tracker.navigate(routeLocation('task', '/task/download'));
        attempt.setTaskFormat(format);
        attempt.finish();
        attempt.finish('abandoned');
        expect(metrics).toEqual([]);
    });

    it.each(['/build/', '/Build', '/BUILD//'])('should match the router for %s', pathname => {
        const { tracker, metrics } = setup();
        tracker.navigate(routeLocation('build', pathname));
        tracker.forBuild('build')!.taskId = 'created-task';
        tracker.navigate(routeLocation('task', '/Task/created-task/'));
        tracker.forTask('created-task')!.finish();
        expect(metrics.map(metric => metric.name)).toEqual(['data.load.finish.network']);
    });

    it('should keep building when the tab becomes hidden', () => {
        const { tracker, metrics } = setup();
        tracker.navigate(routeLocation('build', '/build'));
        jest.spyOn(document, 'visibilityState', 'get').mockReturnValue('hidden');
        document.dispatchEvent(new Event('visibilitychange'));
        expect(metrics).toEqual([]);
        tracker.forBuild('build')!.finish();
        expect(metrics.map(metric => metric.name)).toEqual(['data.load.finish.network']);
    });

    it('should exclude direct task navigation', () => {
        const { tracker, metrics } = setup();
        tracker.navigate(routeLocation('excluded', '/task/existing'));
        tracker.error();
        window.dispatchEvent(new Event('pagehide'));
        expect(tracker.forBuild('excluded')).toBeUndefined();
        expect(metrics).toEqual([]);
    });
});

// Helpers
function routeLocation(key: string, pathname: string, search = '') {
    return { key, pathname, search };
}

function setup() {
    jest.spyOn(console, 'error').mockImplementation(() => {});
    const clock = { now: 25 };
    jest.spyOn(performance, 'now').mockImplementation(() => clock.now);
    const metrics: { name: string; value: number; page: string; additional: Record<string, string> }[] = [];
    // Run the installed counter's SPA implementation: a mock of finishDataLoading
    // would miss incorrect lifecycle options or measuring from the wrong stage.
    const counter = {
        enabled: true,
        getTime: () => clock.now,
        makeSubPage: (page: string) => ({ '2924': page, '689.2322': clock.now }),
        sendDelta: (name: string, value: number, params?: Record<string, any>) => {
            metrics.push({ name, value, page: params?.['2924'], additional: JSON.parse(decodeURIComponent(params?.additional)) });
        },
        spa: {} as Rum,
    };
    runInNewContext(readFileSync(require.resolve('@yandex-int/rum-counter/dist/bundle/spa-metric.js'), 'utf8'), {
        Ya: { Rum: counter }, window: {}, console,
    });
    const tracker = new BuildProfileRum(() => ({ ...counter.spa, sendDelta: counter.sendDelta }));
    return { tracker, metrics, clock };
}
