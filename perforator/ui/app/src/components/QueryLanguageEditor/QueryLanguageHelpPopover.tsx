import React from 'react';

import { HelpMark, Link } from '@gravity-ui/uikit';

import { uiFactory } from 'src/factory';
import { cn } from 'src/utils/cn';

import i18n from './i18n';


const b = cn('query-language-editor');

export const QueryLanguageHelpPopover: React.FC = () => (
    <HelpMark
        iconSize={'l'}
        className={b('help')}
        popoverProps={{ className: b('help-popover') }}
    >
        {i18n('selectorDescription')}
        <code className={b('help-code')}>{' {label="value"}'}</code>
        {i18n('exampleHeading')}
        <code className={b('help-code')}>
            {'{service="my-app", cpu=~"AMD.*", event_type = "wall.seconds"}'}
        </code>
        <br/>
        {i18n('operatorsHeading')}
        <br/>{'• =, != : '}{i18n('exactMatch')}{'|'}{i18n('exactMatchOrExample')}{'app1|app2'}{i18n('exactMatchEnd')}
        <br/>{'• =~, !~ : '}{i18n('regexMatch')}
        <br/>{'• <, >, <=, >= : '}{i18n('ordinalComparison')}
        <br />
        <br />
        <Link target="_blank" href={uiFactory().queryLanguageDocsLink()}>
            {i18n('documentation')}
        </Link>
    </HelpMark>
);
