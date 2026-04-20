import React from 'react';

import { HelpMark, Link } from '@gravity-ui/uikit';

import { uiFactory } from 'src/factory';
import { cn } from 'src/utils/cn';


const b = cn('query-language-editor');

export const QueryLanguageHelpPopover: React.FC = () => (
    <HelpMark
        iconSize={'l'}
        className={b('help')}
        popoverProps={{ className: b('help-popover') }}
    >
        Selector consists of comma-separated triples of keys, operators and values wrapped
        in curly braces, i.e.{' '}
        <code className={b('help-code')}>{' {label="value"}'}</code>
        Example:
        <code className={b('help-code')}>
            {'{service="my-app", cpu=~"AMD.*", event_type = "wall.seconds"}'}
        </code>
        <br/>
        Operators:
        <br/>• =, != : Exact match (use | for OR, e.g., "app1|app2")
        <br/>• =~, !~ : Regex match
        <br/>• {'<'}, {'>'}, {'<='}, {'>='} : Ordinal comparison (useful for time)
        <br />
        <br />
        <Link target="_blank" href={uiFactory().queryLanguageDocsLink()}>
            Read full documentation
        </Link>
    </HelpMark>
);
