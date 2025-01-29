import React from 'react';

import { FooterItem } from '@gravity-ui/navigation';
import { Icon } from '@gravity-ui/uikit';


const ITEM_ICON_SIZE = 18;

export const openLink = (url?: string) => {
    if (url) {
        window
            .open(url, '_blank')
            ?.focus();
    }
};

export interface NavigationFooterLinkProps {
    text: string;
    compact: boolean;
    icon?: (props: React.SVGProps<SVGSVGElement>) => React.JSX.Element;
    renderIcon?: () => React.JSX.Element;
    url?: string;
    onClick?: () => void;
}

export const NavigationFooterLink: React.FC<NavigationFooterLinkProps> = props => {
    const handleClick = props.onClick ?? (() => openLink(props.url));
    const icon = props.renderIcon
        ? props.renderIcon()
        : (props.icon ? (<Icon size={ITEM_ICON_SIZE} data={props.icon} />) : null);
    return (
        <FooterItem
            compact={props.compact}
            item={{
                id: props.text,
                title: props.text,
                onItemClick: handleClick,
                itemWrapper: (params, makeItem) => makeItem({ ...params, icon }),
            }}
        />
    );
};
