import React, { useCallback } from 'react';

import { AxiosError } from 'axios';

import { DelayedTextInput } from '@gravity-ui/components';
import { Select as GravitySelect } from '@gravity-ui/uikit';

import { cn } from 'src/utils/cn';
import { createErrorToast } from 'src/utils/toaster';

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
    listValues: (filter: SelectFilter, { signal }: { signal: AbortSignal}) => Promise<string[]>;
}

const PAGE_SIZE = 100;

export const Select: React.FC<SelectProps> = props => {
    const value = props.value ? [props.value] : [];

    const [loadState, setLoadState] = React.useState<'idle' | 'loading' | 'error'>('idle');
    const [items, setItems] = React.useState<string[]>(value);
    const [query, setQuery] = React.useState<string>(items[0]);
    const [offset, setOffset] = React.useState<number>(0);

    const prevQueryRef = React.useRef<string | null>(null);
    const abortControllerRef = React.useRef(new AbortController());
    React.useEffect(() => {
        return () => abortControllerRef.current.abort();
    }, []);

    const hasMore = React.useMemo(() => items.length !== 0 && items.length % PAGE_SIZE === 0, [items]);

    const filterItems = async () => {
        if (!abortControllerRef.current.signal.aborted) {
            abortControllerRef.current.abort();
        }
        abortControllerRef.current = new AbortController();
        setLoadState('loading');

        try {
            const newItems = (await props.listValues({ value: query, offset: offset, limit: (PAGE_SIZE) }, { signal: abortControllerRef.current.signal }));

            setItems(oldItems => oldItems.concat(newItems));

            setLoadState('idle');
        } catch (error) {
            if (error instanceof AxiosError && error.code === 'ERR_CANCELED') {
                return;
            }

            setLoadState('error');
            createErrorToast(
                error,
                { name: 'list-services', title: 'Failed to load service names' },
            );
        }
    };

    React.useEffect(() => {
        if (query !== prevQueryRef.current) {
            prevQueryRef.current = query;
            setItems([]);
        }
        filterItems();
    }, [query, offset]);

    const options = React.useMemo(
        () => items.map(item => ({ value: item, content: item })),
        [items],
    );

    const handleLoadMore = useCallback(() => {
        if (loadState === 'idle') {
            setOffset(o => o + PAGE_SIZE);
        }
    }, [loadState]);

    const handleQueryChange = useCallback((q: string) => {
        setQuery(q);
        setOffset(0);
    }, []);

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
                        onUpdate={handleQueryChange}
                        onKeyDown={inputProps.onKeyDown}
                        autoFocus
                    />
                </div>
            )}
            className={b('control')}
            popupClassName={b('popup')}
            // controls loader in the end of the list in select
            loading={hasMore || loadState === 'loading'}
            width="max"
            renderEmptyOptions={() => (
                <div className={b('empty')}>
                    {query ? 'No matches found :(' : 'Enter search string…'}
                </div>
            )}
            onLoadMore={handleLoadMore}
        />
    );
};
