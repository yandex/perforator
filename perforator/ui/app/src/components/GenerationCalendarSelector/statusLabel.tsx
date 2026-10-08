import { Label } from '@gravity-ui/uikit';

import { ClusterTopGenerationStatus } from 'src/generated/perforator/proto/perforator/perforator';

import i18n from './i18n';


export function generationStatusLabel(status: ClusterTopGenerationStatus) {
    switch (status) {
    case ClusterTopGenerationStatus.IN_PROGRESS:
        return <Label theme="warning" size="xs">{i18n('building')}</Label>;
    case ClusterTopGenerationStatus.COMPLETED:
        return <Label theme="success" size="xs">{i18n('completed')}</Label>;
    default:
        return <Label theme="unknown" size="xs">{i18n('unknown')}</Label>;
    }
}
