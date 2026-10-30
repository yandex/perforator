import type { ProfileData } from '../models/Profile';

import type { PprofRenderOptions } from './options';


export interface PprofWorkerRequest {
    id: number;
    source?: ArrayBuffer;
    engine?: ArrayBuffer;
    options: PprofRenderOptions;
}

export interface PprofRenderMetrics {
    profileParseMs: number;
    renderMs: number;
    jsonParseMs: number;
    jsonBytes: number;
    wasmMemoryBytes: number;
}

export interface PprofRenderResult {
    profile: ProfileData;
    metrics: PprofRenderMetrics;
}

export type PprofWorkerResponse =
    | { id: number; result: PprofRenderResult }
    | { id: number; error: string };
