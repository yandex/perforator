import { apiClient } from 'src/utils/api';
import { createErrorToast } from 'src/utils/toaster';


export const fetchServices = async (
    value: string,
    params?: {
        offset?: number;
        limit?: number;
    },
): Promise<Optional<string[]>> => {
    try {
        const response = await apiClient.getServices({
            'Paginated.Offset': params?.offset ?? 0,
            'Paginated.Limit': params?.limit ?? 100,
            Regex: value,
        });
        const services = response?.data?.Services || [];
        return services.map(({ ServiceID: service }) => service).filter(service => service);
    } catch (error: unknown) {
        createErrorToast(
            error,
            { name: 'list-services', title: 'Failed to load service names' },
        );
    }
    return [];
};
