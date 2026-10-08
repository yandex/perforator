import React from 'react';

import {
    CircleQuestion,
    Gear,
    LogoTelegram,
    Snail,
} from '@gravity-ui/icons';

import { uiFactory } from 'src/factory';

import { BugReportLink } from './BugReportLink/BugReportLink';
import i18n from './i18n';
import { NavigationFooterLink } from './NavigationFooterLink/NavigationFooterLink';
import { UserLink } from './UserLink/UserLink';


export interface NavigationFooterProps {
    compact: boolean;
    toggleSettings: () => void;
}

export const NavigationFooter: React.FC<NavigationFooterProps> = ({ compact, toggleSettings }: NavigationFooterProps) => {
    return (
        <>
            {!uiFactory().docsLink() ? null : <NavigationFooterLink
                text={i18n('documentation')}
                url={uiFactory().docsLink()}
                icon={CircleQuestion}
                compact={compact}
            />}
            {!uiFactory().supportChatLink() ? null : <NavigationFooterLink
                text={i18n('supportChat')}
                url={uiFactory().supportChatLink()}
                icon={LogoTelegram}
                compact={compact}
            />}
            {!uiFactory().bugReportLink() ? null : <BugReportLink
                compact={compact}
            />}
            {!uiFactory().clientSpeedDebugLink() ? null : <NavigationFooterLink
                text={i18n('clientSpeedDebug')}
                url={uiFactory().clientSpeedDebugLink()}
                icon={Snail}
                compact={compact}
            />}
            <NavigationFooterLink
                text={i18n('settings')}
                icon={Gear}
                onClick={toggleSettings}
                compact={compact}
            />
            {!uiFactory().authorizationSupported() ? null : <UserLink
                compact={compact}
            />}
        </>
    );
};
