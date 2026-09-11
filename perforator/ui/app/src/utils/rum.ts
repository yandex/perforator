import type { Memory } from './performance';


export interface Rum {
    finishDataLoading?: (value: string) => void;
    finishDataRendering?: (value: string) => void;
    makeSpaSubPage?: (value: string, options?: object, isBlock?: boolean, shouldFinalizePageMetrics?: boolean, params?: Record<string, unknown>) => Record<string, any> | undefined;
    startDataRendering?: (value: string, renderType: string, shouldCall: boolean) => void;
    logMemory?: (zone: string, value: Memory) => void;
    logInt?: (name: string, value: number) => void;
    sendDelta?: (deltaName: string, value: number, params?: Record<string, any>) => void;
    sendResTiming?: (url: string) => void;
}

export const fakeRum: Rum = {};
