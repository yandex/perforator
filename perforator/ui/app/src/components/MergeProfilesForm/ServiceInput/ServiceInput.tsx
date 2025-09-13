import React from 'react';

import { Select, type SelectFilter } from 'src/components/Select/Select';

import { fetchServices } from './fetchServices';

import './ServiceInput.scss';


export interface ServiceInputProps {
    service?: string;
    onUpdate: (service: string | undefined) => void;
}

export const ServiceInput: React.FC<ServiceInputProps> = props => {
    React.useEffect(() => {
        props.onUpdate(props.service);
    }, []);

    const listValues = React.useCallback(async (filter: SelectFilter, { signal }: { signal: AbortSignal}) => (
        filter.value
            ? await fetchServices(filter.value, {
                offset: filter.offset,
                limit: filter.limit,
            }, { signal }) || []
            : []
    ), []);

    return <Select
        listValues={listValues}
        onUpdate={props.onUpdate}
        value={props.service}
        placeholder={'Service regexp'}
    />;
};


