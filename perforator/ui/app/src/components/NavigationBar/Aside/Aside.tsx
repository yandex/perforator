import React from 'react';

import BarsDescendingAlignLeftIcon from '@gravity-ui/icons/svgs/bars-descending-align-left.svg?raw';
import ClockArrowRotateLeftIcon from '@gravity-ui/icons/svgs/clock-arrow-rotate-left.svg?raw';
import GraduationCapIcon from '@gravity-ui/icons/svgs/graduation-cap.svg?raw';
import ScalesUnbalancedIcon from '@gravity-ui/icons/svgs/scales-unbalanced.svg?raw';
import ServerIcon from '@gravity-ui/icons/svgs/server.svg?raw';
import type { DrawerItemProps, MenuItem } from '@gravity-ui/navigation';
import { PageLayoutAside } from '@gravity-ui/navigation';

import PerforatorLogo from 'src/assets/perforator.svg?raw';
import { Link } from 'src/components/Link/Link';
import { Tutorials } from 'src/components/Tutorials/Tutorials';
import { LocalStorageKey } from 'src/const/localStorage';
import { uiFactory } from 'src/factory';

import { NavigationFooter } from '../NavigationFooter/NavigationFooter';
import { SettingsPanel } from '../SettingsPanel/SettingsPanel';

import type { AsideProps } from './AsideProps';


interface MenuLink {
    title: string;
    icon: string;
    link: string;
}

const isClusterTopEnabled = Boolean(localStorage.getItem(LocalStorageKey.ClusterTop));

const menuLinks: MenuLink[] = [
    {
        title: 'Profiles',
        icon: BarsDescendingAlignLeftIcon,
        link: '/',
    },
    ...(isClusterTopEnabled ? [{
        title: 'Cluster Top',
        icon: ServerIcon,
        link: '/cluster-top',
    }] : []),
    {
        title: 'History',
        icon: ClockArrowRotateLeftIcon,
        link: '/tasks',
    },
    {
        title: 'Diff',
        icon: ScalesUnbalancedIcon,
        link: '/diff',
    },
];

const makeMenuItem = (link: MenuLink): MenuItem => ({
    id: link.title,
    title: link.title,
    icon: link.icon,
    current: window.location.pathname === link.link,
    itemWrapper: (props, makeItem) => (
        <Link
            className="gn-composite-bar-item__link"
            href={link.link || '#'}
        >
            {makeItem(props)}
        </Link>
    ),
});

const menuLinkItems = menuLinks.map(makeMenuItem);

type PanelItems = 'settings' | 'tutorials';

export const Aside: React.FC<AsideProps> = ({ setCompact }: AsideProps) => {
    const asideRef = React.useRef<HTMLDivElement>(null);

    const [showPanel, setShowPanel] = React.useState<PanelItems | null>(null);

    const panelItems = React.useMemo(() => [
        {
            id: 'settings',
            content: <SettingsPanel />,
            visible: showPanel === 'settings',
        },
        {
            id: 'tutorials',
            children: <Tutorials onItemClick={() => setShowPanel(null)} />,
            visible: showPanel === 'tutorials',
        },
    ] as DrawerItemProps[], [showPanel]);

    const items = menuLinkItems.concat(
        [{
            title: 'Learn',
            icon: GraduationCapIcon,
            id: 'tutorials',
            current: showPanel === 'tutorials' || window.location.pathname.startsWith('/tutorials'),
            onItemClick: () => setShowPanel(
                showPanel ? null : 'tutorials',
            ),
        }],
    );

    return (
        <PageLayoutAside
            ref={asideRef}
            logo={{
                icon: PerforatorLogo,
                text: 'Perforator',
                iconSize: 32,
                href: '/',
            }}
            multipleTooltip
            headerDecoration
            onChangeCompact={setCompact}
            subheaderItems={uiFactory().useSubheaderItems(asideRef)}
            menuItems={items}
            panelItems={panelItems}
            onClosePanel={() => setShowPanel(null)}
            renderFooter={({ compact }) => (
                <NavigationFooter
                    compact={compact}
                    toggleSettings={() => setShowPanel(settings => settings ? null : 'settings')}
                />
            )}
        />
    );
};
