import type { Theme } from '@gravity-ui/uikit';


export type ShortenMode = 'true' | 'false' | 'hover';

export type NumTemplatingFormat = 'exponent' | 'hugenum';

export interface UserSettings {
    monospace: 'default' | 'system';
    numTemplating: NumTemplatingFormat;
    theme: Theme;
    shortenFrameTexts: ShortenMode;
    reverseFlameByDefault: boolean;
}
