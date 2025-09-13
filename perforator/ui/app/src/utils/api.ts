import type {
    AxiosInstance,
    AxiosRequestConfig,
    AxiosResponse,
} from 'axios';
import axios from 'axios';

import type {
    ListProfilesResponse,
    ListServicesResponse,
    ListSuggestionsResponse,
} from 'src/generated/perforator/proto/perforator/perforator';
import type {
    GetTaskResponse,
    ListTasksResponse,
    StartTaskResponse,
} from 'src/generated/perforator/proto/perforator/task_service';


type RequestData = any;
type RequestSender = () => Promise<AxiosResponse>;

type AllowedOptions = Partial<Pick<AxiosRequestConfig, 'cancelToken' | 'signal'>>

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

    getSuggestions(params: RequestData, options: AllowedOptions): Promise<AxiosResponse<ListSuggestionsResponse>> {
        return this.get('/api/v0/suggestions', params, options);
    }

    getProfiles(params: RequestData): Promise<AxiosResponse<ListProfilesResponse>> {
        return this.get('/api/v0/profiles', params);
    }

    getTask(taskId: string): Promise<AxiosResponse<GetTaskResponse>> {
        return this.get(`/api/v0/tasks/${taskId}`);
    }

    getTasks(params: RequestData): Promise<AxiosResponse<ListTasksResponse>> {
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
            () => this.httpClient.get<T>(url, { ...options, params: data }),
        );
    }

    protected post(url: string, data: RequestData = {}): Promise<AxiosResponse> {
        return this.makeRequest(
            () => this.httpClient.post(url, data),
        );
    }
}

export const apiClient = new PerforatorApiClient();
