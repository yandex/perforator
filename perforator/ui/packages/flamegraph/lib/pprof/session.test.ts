import { Blob as NodeBlob } from 'node:buffer';

import { afterEach, beforeEach, describe, expect, it, jest } from '@jest/globals';

import { defaultPprofOptions } from './options';
import type { PprofRenderResult, PprofWorkerRequest, PprofWorkerResponse } from './protocol';
import { PprofSession } from './session';


class ErrorEventStub extends Event {
    readonly message: string;
    constructor(type: string, init: ErrorEventInit) {
        super(type);
        this.message = init.message ?? '';
    }
}


describe('PprofSession', () => {
    const originalFetch = globalThis.fetch;
    const originalBlob = globalThis.Blob;
    const originalErrorEvent = globalThis.ErrorEvent;
    beforeEach(() => {
        Object.assign(globalThis, { Blob: NodeBlob, ErrorEvent: ErrorEventStub });
        globalThis.fetch = jest.fn<typeof fetch>().mockResolvedValue({ ok: true, blob: async () => new Blob(['pprof']) } as Response);
    });
    afterEach(() => {
        Object.assign(globalThis, { fetch: originalFetch, Blob: originalBlob, ErrorEvent: originalErrorEvent });
    });

    it('should download and parse once across option changes', async () => {
        const worker = new WorkerStub();
        const session = new PprofSession('/profile', () => worker.asWorker());
        const stages: string[] = [];
        const first = session.render(defaultPprofOptions, stage => stages.push(stage));
        expect(stages).toEqual(['loading']);
        const load = await worker.next();
        expect(stages).toEqual(['loading', 'processing']);
        expect(Object.prototype.toString.call(load.source)).toBe('[object ArrayBuffer]');
        expect(worker.transferred).toEqual([load.source]);
        worker.reply({ id: load.id, result: result(100) });
        await first;

        const next = session.render({ ...defaultPprofOptions, maxDepth: 0 }, stage => stages.push(stage));
        const request = await worker.next();
        expect(stages).toEqual(['loading', 'processing', 'processing']);
        expect(request.source).toBeUndefined();
        worker.reply({ id: request.id, result: result(200) });
        expect((await next).profile.rows[0][0].eventCount).toBe(200);
        expect(globalThis.fetch).toHaveBeenCalledTimes(1);
        expect(worker.terminated).toBe(false);
        session.dispose();
        expect(worker.terminated).toBe(true);
    });

    it('should cancel a render and ignore its late result while retrying the same download', async () => {
        const oldWorker = new WorkerStub();
        const newWorker = new WorkerStub();
        const createWorker = jest.fn<() => Worker>().mockReturnValueOnce(oldWorker.asWorker()).mockReturnValueOnce(newWorker.asWorker());
        const session = new PprofSession('/profile', createWorker);
        const old = session.render(defaultPprofOptions);
        const rejected = expect(old).rejects.toMatchObject({ name: 'AbortError' });
        const oldRequest = await oldWorker.next();
        session.cancel();
        await rejected;
        expect(oldWorker.terminated).toBe(true);

        const retry = session.render(defaultPprofOptions);
        const newRequest = await newWorker.next();
        oldWorker.reply({ id: oldRequest.id, result: result(1) });
        newWorker.reply({ id: newRequest.id, result: result(2) });
        expect((await retry).profile.rows[0][0].eventCount).toBe(2);
        expect(new Uint8Array(newRequest.source!)).toEqual(new Uint8Array(oldRequest.source!));
        expect(globalThis.fetch).toHaveBeenCalledTimes(1);
        session.dispose();
    });

    it('should recover from a failed worker without downloading again', async () => {
        const workers = [new WorkerStub(), new WorkerStub()];
        let index = 0;
        const session = new PprofSession('/profile', () => workers[index++].asWorker());
        const first = session.render(defaultPprofOptions);
        const failed = expect(first).rejects.toThrow('bad pprof');
        const request = await workers[0].next();
        workers[0].reply({ id: request.id, error: 'bad pprof' });
        await failed;
        const retry = session.render(defaultPprofOptions);
        const next = await workers[1].next();
        workers[1].reply({ id: next.id, result: result(100) });
        await retry;
        expect(globalThis.fetch).toHaveBeenCalledTimes(1);
        session.dispose();
    });

    it('transfers embedded profile and engine bytes on initialization and keeps them for retry', async () => {
        const source = new Blob(['prepared pprof']);
        const engine = new Blob(['engine']);
        const firstWorker = new WorkerStub();
        const retryWorker = new WorkerStub();
        const factory = jest.fn<() => Worker>().mockReturnValueOnce(firstWorker.asWorker()).mockReturnValueOnce(retryWorker.asWorker());
        const session = new PprofSession(source, factory, engine);
        const first = session.render(defaultPprofOptions);
        const failed = expect(first).rejects.toThrow('allocation failed');
        const load = await firstWorker.next();
        expect(new Uint8Array(load.source!)).toEqual(new Uint8Array(await source.arrayBuffer()));
        expect(new Uint8Array(load.engine!)).toEqual(new Uint8Array(await engine.arrayBuffer()));
        expect(firstWorker.transferred).toEqual([load.source, load.engine]);
        firstWorker.reply({ id: load.id, error: 'allocation failed' });
        await failed;
        const retry = session.render(defaultPprofOptions);
        const next = await retryWorker.next();
        expect(new Uint8Array(next.source!)).toEqual(new Uint8Array(load.source!));
        expect(new Uint8Array(next.engine!)).toEqual(new Uint8Array(load.engine!));
        expect(retryWorker.transferred).toEqual([next.source, next.engine]);
        retryWorker.reply({ id: next.id, result: result(100) });
        await retry;
        expect(globalThis.fetch).not.toHaveBeenCalled();
        session.dispose();
    });

    it('should retain the worker failure message', async () => {
        const worker = new WorkerStub();
        const session = new PprofSession('/profile', () => worker.asWorker());
        const render = session.render(defaultPprofOptions);
        const failed = expect(render).rejects.toThrow('worker initialization failed');
        await worker.next();
        worker.onerror?.(new ErrorEvent('error', { message: 'worker initialization failed' }));
        await failed;
        expect(worker.terminated).toBe(true);
        session.dispose();
    });

    it('should reject a failed HTTP download before creating a worker', async () => {
        jest.mocked(globalThis.fetch).mockResolvedValue({ ok: false, status: 403 } as Response);
        const factory = jest.fn<() => Worker>();
        const session = new PprofSession('/profile', factory);
        await expect(session.render(defaultPprofOptions)).rejects.toThrow('HTTP 403');
        expect(factory).not.toHaveBeenCalled();
        session.dispose();
        await expect(session.render(defaultPprofOptions)).rejects.toThrow('disposed');
    });
});

// Helpers
class WorkerStub {
    onerror?: (event: Event) => void;
    onmessage?: (event: MessageEvent<PprofWorkerResponse>) => void;
    terminated = false;
    transferred: Transferable[] = [];
    private waiting?: (request: PprofWorkerRequest) => void;
    next() { return new Promise<PprofWorkerRequest>(resolve => { this.waiting = resolve; }); }
    postMessage(request: PprofWorkerRequest, transfer: Transferable[]) { this.transferred = transfer; this.waiting!(request); }
    reply(response: PprofWorkerResponse) { this.onmessage?.({ data: response } as MessageEvent<PprofWorkerResponse>); }
    terminate() { this.terminated = true; }
    asWorker() { return this as unknown as Worker; }
}

function result(eventCount: number): PprofRenderResult {
    return { profile: { rows: [[{ parentIndex: -1, textId: 0, sampleCount: 1, eventCount }]],
        stringTable: ['root'], meta: { version: 1, eventType: 0, frameType: 0 } },
    metrics: { profileParseMs: 0, renderMs: 0, jsonParseMs: 0, jsonBytes: 100, wasmMemoryBytes: 1024 } };
}
