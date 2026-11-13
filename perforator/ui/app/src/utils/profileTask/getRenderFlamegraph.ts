import type { FlamegraphOptions, RenderFormat } from 'src/generated/perforator/proto/perforator/perforator';
import type { ProfileTaskQuery } from 'src/models/Task';


export function getRenderFlamegraph(
    query: ProfileTaskQuery,
    flamegraphOptions: FlamegraphOptions,
    settings: {wasmEnabled: boolean; mergeIntoPprofByDefault?: boolean},
): Partial<RenderFormat> {
    if (query.rawProfile === 'true' || query.format === 'raw') {
        return { RawProfile: flamegraphOptions };
    }
    if (query.format === 'text') {
        return { TextProfile: flamegraphOptions };
    }
    if (settings.wasmEnabled && settings.mergeIntoPprofByDefault && !query.diffSelector && !query.format) {
        return { RawProfile: {} };
    }
    return { JSONFlamegraph: flamegraphOptions };
}
