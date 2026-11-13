import type { GetTaskResponse } from 'src/generated/perforator/proto/perforator/task_service';


export { TaskState } from 'src/generated/perforator/proto/perforator/task_service';

export interface ProfileTaskQuery {
    idempotencyKey?: string;

    from: string;
    to: string;
    maxProfiles: number;

    diffSelector?: string;

    diffFrom?: string;
    diffTo?: string;

    selector?: string;

    service?: string;
    profileId?: string;
    /** @deprecated use format */
    rawProfile?: 'true' | 'false';
    format?: 'text' | 'raw' | 'json';
    cluster?: string;
    lineNumbers?: 'true' | 'false';
    prevTask?: string;
    maxDepth?: number;
    minWeight?: number;
    showFileNames?: 'true' | 'false';
}


export type TaskResult = GetTaskResponse;
