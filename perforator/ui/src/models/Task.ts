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
    rawProfile?: 'true' | 'false';
}


export type TaskResult = GetTaskResponse;
