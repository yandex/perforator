type StringTableIndex = number

export interface FormatNode {
    parentIndex: number;
    textId: StringTableIndex;
    /** added during render */
    selfSampleCount?: number;
    /** added during render */
    selfEventCount?: number;
    /** added during render */
    baseSelfSampleCount?: number;
    /** added during render */
    baseSelfEventCount?: number;
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
    frameOrigin?: StringTableIndex;
    file?: StringTableIndex;
    kind?: StringTableIndex;
    omittedNode?: boolean;
    /** added during render for uninteresting frames on alt-click */
    omittedEventCount?: number;
    omittedSampleCount?: number;
    inlined?: boolean;
    childrenIndices?: Set<number>;
}

export type ProfileData = {
    rows: FormatNode[][];
    stringTable: string[];
    meta: ProfileMeta;
}


export type ProfileMeta = {
    eventType: StringTableIndex;
    frameType: StringTableIndex;
    version: number;
}


export type StringifiableFields = 'frameOrigin' | 'file' | 'kind' | 'textId';
export type StringifiedNode = Omit<FormatNode, StringifiableFields> & {
    [key in StringifiableFields]: string;
};
