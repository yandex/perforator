import React from 'react';

import {
    CircleQuestion,
    Gear,
    LogoTelegram,
} from '@gravity-ui/icons';

import { uiFactory } from 'src/factory';

import { BugReportLink } from './BugReportLink/BugReportLink';
import { NavigationFooterLink } from './NavigationFooterLink/NavigationFooterLink';
import { UserLink } from './UserLink/UserLink';


export interface NavigationFooterProps {
    compact: boolean;
    toggleSettings: () => void;
}

export const NavigationFooter: React.FC<NavigationFooterProps> = props => {
    const { compact } = props;
    return (
        <>
            {!uiFactory().docsLink() ? null : <NavigationFooterLink
                text="Documentation"
                url={uiFactory().docsLink()}
                icon={CircleQuestion}
                compact={compact}
            />}
            {!uiFactory().supportChatLink() ? null : <NavigationFooterLink
                text="Support chat"
                url={uiFactory().supportChatLink()}
                icon={LogoTelegram}
                compact={compact}
            />}
            {!uiFactory().bugReportLink() ? null : <BugReportLink
                compact={compact}
            />}
            <NavigationFooterLink
                text="Settings"
                icon={Gear}
                onClick={props.toggleSettings}
                compact={compact}
            />
            {!uiFactory().authorizationSupported() ? null : <UserLink
                compact={compact}
            />}
        </>
    );
};
