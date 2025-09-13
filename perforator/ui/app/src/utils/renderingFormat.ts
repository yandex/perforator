import type { RenderFormat } from 'src/generated/perforator/proto/perforator/perforator';
import type { TaskResult } from 'src/models/Task';


function getWellKnownKeysFromObject<O extends Record<string, any>, K extends keyof O>(obj: O, keys: K[]): K[] {
    return keys.filter(k => k in obj);
}

const getFormatLike = (o: RenderFormat) => getWellKnownKeysFromObject(o, ['Flamegraph', 'JSONFlamegraph', 'RawProfile', 'TextProfile']);
export const getFormat = (o?: RenderFormat) => o ? getFormatLike(o)[0] : undefined;

export function isDiffTaskResult(task: TaskResult | null) {
    return 'DiffProfiles' in (task?.Result || {});
}
