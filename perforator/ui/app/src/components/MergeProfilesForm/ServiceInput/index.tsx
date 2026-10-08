import { makeSelectorFromConditions } from 'src/utils/selector';

import type { QueryInput, QueryInputRenderer } from '../QueryInput';

import i18n from './i18n';
import { ServiceInput } from './ServiceInput';


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
                    setQuery(currentQuery => ({
                        ...currentQuery,
                        service,
                    }));
                }
            }} />
    </div>
);

export const SERVICE_QUERY_INPUT: QueryInput = {
    name: 'Service',
    label: i18n('label'),
    queryField: 'service',
    render: renderServiceInput,
};
