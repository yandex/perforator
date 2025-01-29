export interface FormatNode {
    parentIndex: number;
    textId: number;
    sampleCount: number;
    eventCount: number;
    /**
     * either hash once after downloading or get from mapping
     * already darkened if dark theme is active
     */
    color?: string | number;
    /** only add during render */
    x?: number;
    /** only for diff */
    baseEventCount?: number;
    /** only for diff */
    baseSampleCount?: number;
    frameOrigin?: number;
    file?: number;
    kind?: number;
    inlined?: boolean;
}

export type ProfileData = {
    rows: FormatNode[][];
    stringTable: string[];
}
