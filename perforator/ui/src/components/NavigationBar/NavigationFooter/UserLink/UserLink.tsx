import React from 'react';

import { uiFactory } from 'src/factory';
import { getUserLogin } from 'src/utils/login';

import { NavigationFooterLink } from '../NavigationFooterLink/NavigationFooterLink';

import './UserLink.scss';


export interface UserLinkProps {
    compact: boolean;
}

export const UserLink: React.FC<UserLinkProps> = props => {
    const login = React.useMemo(() => getUserLogin() || '', []);
    const userLink = uiFactory().makeUserLink(login);
    const avatarLink = uiFactory().makeUserAvatarLink(login);
    return (
        <NavigationFooterLink
            text={login}
            renderIcon={() => <img className="user-link__avatar" src={avatarLink} />}
            url={userLink}
            compact={props.compact}
        />
    );
};
