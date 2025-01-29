import React from 'react';

import { DelayedTextInput } from '@gravity-ui/components';
import { Select as GravitySelect } from '@gravity-ui/uikit';

import { cn } from 'src/utils/cn';

import './Select.scss';


const b = cn('select');

export interface SelectFilter {
    value?: string;
    offset?: number;
    limit?: number;
}

export interface SelectProps {
    value?: string;
    placeholder?: string;
    onUpdate: (value: Optional<string>) => void;
    listValues: (filter: SelectFilter) => Promise<string[]>;
}

export const Select: React.FC<SelectProps> = props => {
    const value = props.value ? [props.value] : [];

    const [items, setItems] = React.useState<string[]>(value);
    const [query, setQuery] = React.useState<string>(items[0]);
    const [loading, setLoading] = React.useState(false);

    const filterItems = async () => {
        setItems([]);
        setLoading(true);
        setItems(await props.listValues({ value: query }));
        setLoading(false);
    };

    React.useEffect(() => {
        filterItems();
    }, [query]);

    const options = React.useMemo(
        () => items.map(item => ({ value: item, content: item })),
        [items],
    );

    return (
        <GravitySelect
            value={value}
            options={options}
            placeholder={props.placeholder}
            onUpdate={values => props.onUpdate(values[0])}
            filterable={true}
            renderFilter={({ inputProps }) => (
                <div className={b('input')}>
                    <DelayedTextInput
                        view="clear"
                        placeholder="Search"
                        value={query}
                        onUpdate={setQuery}
                        onKeyDown={inputProps.onKeyDown}
                        autoFocus
                    />
                </div>
            )}
            popupClassName={b('popup')}
            loading={loading}
            width="max"
            renderEmptyOptions={() => (
                <div className={b('empty')}>
                    {query ? 'No matches found :(' : 'Enter search string…'}
                </div>
            )}
        />
    );
};
