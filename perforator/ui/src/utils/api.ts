import type {
    AxiosInstance,
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

class PerforatorApiClient {
    protected httpClient: AxiosInstance;

    constructor() {
        this.httpClient = axios.create({
            baseURL: '/',
        });
    }

    getServices(params: RequestData): Promise<AxiosResponse<ListServicesResponse>> {
        return this.get('/api/v0/services', params);
    }

    getSuggestions(params: RequestData): Promise<AxiosResponse<ListSuggestionsResponse>> {
        return this.get('/api/v0/suggestions', params);
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

    protected get<T extends any>(url: string, data: RequestData = {}): Promise<AxiosResponse<T>> {
        return this.makeRequest(
            () => this.httpClient.get<T>(url, { params: data }),
        );
    }

    protected post(url: string, data: RequestData = {}): Promise<AxiosResponse> {
        return this.makeRequest(
            () => this.httpClient.post(url, data),
        );
    }
}

export const apiClient = new PerforatorApiClient();
