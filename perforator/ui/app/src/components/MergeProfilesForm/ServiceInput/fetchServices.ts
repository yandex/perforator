import { apiClient } from 'src/utils/api';


export const fetchServices = async (
    value: string,
    params?: {
        offset?: number;
        limit?: number;
    },
    config?: {
        signal?: AbortSignal;
    },
): Promise<Optional<string[]>> => {
    const response = await apiClient.getServices({
        'Paginated.Offset': params?.offset ?? 0,
        'Paginated.Limit': params?.limit ?? 100,
        Regex: value,
    }, {
        signal: config?.signal,
    });
    const services = response?.data?.Services || [];
    return services.map(({ ServiceID: service }) => service).filter(service => service);
};
