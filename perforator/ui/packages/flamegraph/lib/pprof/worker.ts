import type { PprofWorkerRequest, PprofWorkerResponse } from './protocol';
import { PprofRuntime } from './runtime';


export function installPprofWorker(wasmUrl?: string): void {
    const worker = self as unknown as {
        onmessage: ((event: MessageEvent<PprofWorkerRequest>) => void) | null;
        postMessage: (message: PprofWorkerResponse) => void;
    };
    let runtime: PprofRuntime | undefined;
    let running = false;

    worker.onmessage = async ({ data }) => {
        if (running) {
            worker.postMessage({ id: data.id, error: 'The profile engine is already rendering.' });
            return;
        }
        running = true;
        try {
            let profileParseMs = 0;
            if (data.source) {
                runtime?.dispose();
                if (data.engine) {
                    runtime = await PprofRuntime.createFromBytes(data.engine);
                } else if (wasmUrl) {
                    runtime = await PprofRuntime.create(wasmUrl);
                } else {
                    throw new Error('The embedded profile engine is missing.');
                }
                const started = performance.now();
                runtime.load(data.source);
                profileParseMs = performance.now() - started;
            }
            if (!runtime) {
                throw new Error('No profile is loaded.');
            }
            const result = await runtime.render(data.options);
            result.metrics.profileParseMs = profileParseMs;
            worker.postMessage({ id: data.id, result });
        } catch (error) {
            worker.postMessage({ id: data.id, error: error instanceof Error ? error.message : String(error) });
        } finally {
            running = false;
        }
    };
}
