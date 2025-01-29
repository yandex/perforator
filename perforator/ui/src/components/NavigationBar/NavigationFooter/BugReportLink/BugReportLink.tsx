import React from 'react';

import { Bug } from '@gravity-ui/icons';

import { uiFactory } from 'src/factory';

import { NavigationFooterLink, openLink } from '../NavigationFooterLink/NavigationFooterLink';


export interface BugReportLinkProps {
    compact: boolean;
}

export const BugReportLink: React.FC<BugReportLinkProps> = props => (
    <NavigationFooterLink
        text="Report a bug"
        icon={Bug}
        compact={props.compact}
        onClick={() => openLink(uiFactory().bugReportLink())}
    />
);
