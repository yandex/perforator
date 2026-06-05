import React from 'react';

import { Bug } from '@gravity-ui/icons';

import { uiFactory } from 'src/factory';

import { NavigationFooterLink } from '../NavigationFooterLink/NavigationFooterLink';
import { openLink } from '../NavigationFooterLink/utils';


export interface BugReportLinkProps {
    compact: boolean;
}

export const BugReportLink: React.FC<BugReportLinkProps> = ({ compact }: BugReportLinkProps) => (
    <NavigationFooterLink
        text="Report a bug"
        icon={Bug}
        compact={compact}
        onClick={() => openLink(uiFactory().bugReportLink())}
    />
);
