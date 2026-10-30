import type { PprofRenderOptions } from './options';
import { validatePprofOptions } from './options';
import type { PprofRenderResult, PprofWorkerRequest, PprofWorkerResponse } from './protocol';


const cancelled = () => new DOMException('Local rendering cancelled.', 'AbortError');
export type PprofStage = 'loading' | 'processing';

/** One downloaded artifact and one worker-owned parsed profile per task. */
export class PprofSession {
    private source?: Promise<Blob>;
    private downloadController?: AbortController;
    private worker?: Worker;
    private generation = 0;
    private busy = false;
    private disposed = false;
    private pending?: {
        id: number;
        resolve: (result: PprofRenderResult) => void;
        reject: (error: Error) => void;
    };

    private readonly input: string | Blob;
    private readonly engine?: Blob;
    private readonly createWorker: () => Worker;

    constructor(input: string | Blob, createWorker: () => Worker, engine?: Blob) {
        this.input = input;
        this.engine = engine;
        this.createWorker = createWorker;
    }

    async render(options: PprofRenderOptions, onStage?: (stage: PprofStage) => void): Promise<PprofRenderResult> {
        if (this.disposed) {
            throw new Error('This profile session has been disposed.');
        }
        validatePprofOptions(options);
        if (this.busy) {
            this.cancel();
        }
        const id = ++this.generation;
        this.busy = true;
        try {
            const downloading = !this.source;
            onStage?.(downloading ? 'loading' : 'processing');
            const source = await this.download();
            if (id !== this.generation) {
                throw cancelled();
            }
            if (downloading) {
                onStage?.('processing');
            }
            const needsSource = !this.worker;
            // Keep the Blobs for retry; transfer fresh buffers to the worker once.
            const [profileBytes, engineBytes] = needsSource
                ? await Promise.all([source.arrayBuffer(), this.engine?.arrayBuffer()])
                : [];
            if (id !== this.generation) {
                throw cancelled();
            }
            if (!this.worker) {
                const worker = this.createWorker();
                this.worker = worker;
                worker.onmessage = ({ data }: MessageEvent<PprofWorkerResponse>) => {
                    if (this.worker !== worker || this.pending?.id !== data.id) {
                        return;
                    }
                    const pending = this.pending;
                    this.pending = undefined;
                    if ('error' in data) {
                        worker.terminate();
                        this.worker = undefined;
                        pending.reject(new Error(data.error));
                    } else {
                        pending.resolve(data.result);
                    }
                };
                const fail = (event: Event) => {
                    if (this.worker === worker) {
                        worker.terminate();
                        this.worker = undefined;
                        const detail = event instanceof ErrorEvent && event.message ? ` ${event.message}` : '';
                        this.pending?.reject(new Error(`The local profile worker failed.${detail} You can retry or download the pprof.`));
                        this.pending = undefined;
                    }
                };
                worker.onerror = fail;
                worker.onmessageerror = fail;
            }
            return await new Promise<PprofRenderResult>((resolve, reject) => {
                this.pending = { id, resolve, reject };
                const request: PprofWorkerRequest = { id, options, ...(needsSource ? { source: profileBytes, engine: engineBytes } : {}) };
                try {
                    this.worker!.postMessage(request, [profileBytes, engineBytes].filter((buffer): buffer is ArrayBuffer => Boolean(buffer)));
                } catch (error) {
                    this.worker!.terminate();
                    this.worker = undefined;
                    this.pending = undefined;
                    reject(error);
                }
            });
        } finally {
            if (id === this.generation) {
                this.busy = false;
            }
        }
    }

    cancel(): void {
        ++this.generation;
        if (this.downloadController) {
            this.downloadController.abort();
            this.downloadController = undefined;
            this.source = undefined;
        }
        this.worker?.terminate();
        this.worker = undefined;
        this.pending?.reject(cancelled());
        this.pending = undefined;
        this.busy = false;
    }

    dispose(): void {
        this.cancel();
        this.source = undefined;
        this.disposed = true;
    }

    private download(): Promise<Blob> {
        if (!this.source && this.input instanceof Blob) {
            this.source = Promise.resolve(this.input);
        }
        if (!this.source) {
            const controller = new AbortController();
            this.downloadController = controller;
            const source = fetch(this.input as string, { signal: controller.signal }).then(async (response) => {
                if (!response.ok) {
                    throw new Error(`Could not download pprof: HTTP ${response.status}.`);
                }
                const blob = await response.blob();
                if (this.downloadController === controller) {
                    this.downloadController = undefined;
                }
                return blob;
            }).catch((error: Error) => {
                if (this.source === source) {
                    this.source = undefined;
                }
                throw error;
            });
            this.source = source;
        }
        return this.source;
    }

}
