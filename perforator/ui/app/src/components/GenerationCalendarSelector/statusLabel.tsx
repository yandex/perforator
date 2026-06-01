import { Label } from '@gravity-ui/uikit';

import { ClusterTopGenerationStatus } from 'src/generated/perforator/proto/perforator/perforator';


export function generationStatusLabel(status: ClusterTopGenerationStatus) {
    switch (status) {
    case ClusterTopGenerationStatus.IN_PROGRESS:
        return <Label theme="warning" size="xs">Building</Label>;
    case ClusterTopGenerationStatus.COMPLETED:
        return <Label theme="success" size="xs">Completed</Label>;
    default:
        return <Label theme="unknown" size="xs">Unknown</Label>;
    }
}
