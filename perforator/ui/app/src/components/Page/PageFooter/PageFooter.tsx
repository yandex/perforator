import React from 'react';

import type { FooterMenuItem } from '@gravity-ui/navigation';
import { Footer } from '@gravity-ui/navigation';

import { uiFactory } from 'src/factory';

import i18n from './i18n';

import './PageFooter.scss';


export const PageFooter: React.FC = () => {
    const items: FooterMenuItem[] = [];
    if (uiFactory().docsLink()) {
        items.push({
            text: i18n('docs'),
            href: uiFactory().docsLink(),
            target: '_blank',
            className: 'page-footer__menu-item',
        });
    }
    const version = import.meta.env?.VITE_RELEASE_VERSION ?? import.meta.env?.VITE_REVISION;
    if (version) {
        items.push({
            text: i18n('versionPrefix') + version,
            href: uiFactory().ciLink(),
            target: '_blank',
            className: 'page-footer__menu-item',
            qa: 'page-footer_version',
        });
    }
    return (
        <Footer
            className="page-footer"
            copyright={uiFactory().footerCopyright()}
            menuItems={items}
        />
    );
};
