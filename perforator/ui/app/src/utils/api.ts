import type {
    AxiosInstance,
    AxiosRequestConfig,
    AxiosResponse,
} from 'axios';
import axios from 'axios';

import type { Paginated } from 'src/generated/perforator/proto/lib/pagination/pagination.ts';
import type {
    ClusterTopRequest,
    ClusterTopResponse,
    ListClusterTopGenerationResponse,
    ListProfilesRequest,
    ListProfilesResponse,
    ListServicesResponse,
    ListSuggestionsRequest,
    ListSuggestionsResponse,
} from 'src/generated/perforator/proto/perforator/perforator';
import type {
    GetTaskResponse,
    ListTasksRequest,
    ListTasksResponse,
    StartTaskResponse,
} from 'src/generated/perforator/proto/perforator/task_service';


type RequestData = any;
type RequestSender = () => Promise<AxiosResponse>;

type AllowedOptions = Partial<Pick<AxiosRequestConfig, 'cancelToken' | 'signal'>>;

class PerforatorApiClient {
    protected httpClient: AxiosInstance;

    constructor() {
        this.httpClient = axios.create({
            baseURL: '/',
        });
    }

    getServices(params: RequestData, options: AllowedOptions): Promise<AxiosResponse<ListServicesResponse>> {
        return this.get('/api/v0/services', params, options);
    }

    getSuggestions(params: ListSuggestionsRequest): Promise<AxiosResponse<ListSuggestionsResponse>> {
        return this.get('/api/v0/suggestions', params);
    }

    getGenerations(params: RequestData, options: AllowedOptions): Promise<AxiosResponse<ListClusterTopGenerationResponse>> {
        return this.get('/api/v0/top/generations', params, options);
    }

    getFunctionTop(params: Pick<ClusterTopRequest, 'Generation' | 'Pagination' | 'FunctionPattern' | 'OrderBy'>): Promise<AxiosResponse<ClusterTopResponse>> {
        return this.get('/api/v0/top/functions', params);
    }

    getServiceTop(params: Pick<ClusterTopRequest, 'Generation' | 'FunctionPattern' | 'OrderBy'>): Promise<AxiosResponse<ClusterTopResponse>> {
        return this.get('/api/v0/top/service', params);
    }

    getProfiles(params: ListProfilesRequest, options?: AllowedOptions): Promise<AxiosResponse<ListProfilesResponse>> {
        return this.get('/api/v0/profiles', params, options);
    }

    getTask(taskId: string): Promise<AxiosResponse<GetTaskResponse>> {
        return this.get(`/api/v0/tasks/${taskId}`);
    }

    getTasks(params: ListTasksRequest): Promise<AxiosResponse<ListTasksResponse>> {
        return this.get('/api/v0/tasks', params);
    }

    startTask(data: RequestData): Promise<AxiosResponse<StartTaskResponse>> {
        return this.post('/api/v0/tasks', data);
    }

    protected makeRequest(sender: RequestSender): Promise<AxiosResponse> {
        return sender();
    }

    protected get<T extends any>(url: string, data: RequestData = {}, options: AllowedOptions = {}): Promise<AxiosResponse<T>> {
        return this.makeRequest(
            () => this.httpClient.get<T>(url, { ...options, params: data, paramsSerializer: { dots: true } }),
        );
    }

    protected post(url: string, data: RequestData = {}): Promise<AxiosResponse> {
        return this.makeRequest(
            () => this.httpClient.post(url, data),
        );
    }
}

export const apiClient = new PerforatorApiClient();

export function getPagination(paginationState: {page: number; pageSize: number}): Paginated {
    return {
        Offset: String((paginationState.page - 1) * paginationState.pageSize),
        Limit: String(paginationState.pageSize),
    };
}
