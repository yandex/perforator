import { installPprofWorker } from '@perforator/flamegraph/dist/pprof/worker';


installPprofWorker(new URL('../../generated/wasm/profile.wasm', import.meta.url).href);
