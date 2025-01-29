import React from 'react';

import type { QueryInput, QueryInputRenderer } from 'src/components/MergeProfilesForm/QueryInput';
import type { SelectFilter } from 'src/components/Select/Select';
import { uiFactory } from 'src/factory';
import { makeSelectorFromConditions } from 'src/utils/selector';

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

    const listValues = React.useCallback(async (filter: SelectFilter) => (
        filter.value
            ? await fetchServices(filter.value, {
                offset: filter.offset,
                limit: filter.limit,
            }) || []
            : []
    ), []);

    return uiFactory().renderSelect({
        value: props.service,
        placeholder: 'PodSetID regexp',
        onUpdate: props.onUpdate,
        listValues,
    });
};

const makeSelectorWithService = (service: string): string => (
    makeSelectorFromConditions([{ field: 'service', value: service }])
);

const renderServiceInput: QueryInputRenderer = (query, setQuery, setTableSelector) => (
    <div className="service-input">
        <ServiceInput
            service={query.service}
            onUpdate={(service) => {
                if (service) {
                    if (setTableSelector) {
                        setTableSelector(makeSelectorWithService(service));
                    }
                    setQuery({
                        ...query,
                        service,
                    });
                }
            }}
        />
    </div>
);

export const SERVICE_QUERY_INPUT: QueryInput = {
    name: 'Service',
    queryField: 'service',
    render: renderServiceInput,
};
