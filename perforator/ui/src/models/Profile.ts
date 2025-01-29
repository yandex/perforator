type FrameX = number;
type FrameWidth = number;
type FrameText = number;
type FrameEventCount = number;
type FrameSampleCount = number;
type FrameFillStyle = number;

type FrameBaseEventCount = number;

type FrameBaseSampleCount = number;
type BaseFrameLevel = [
    FrameX[],
    FrameWidth[],
    FrameText[],
    FrameEventCount[],
    FrameSampleCount[],
    FrameFillStyle[],
];

type DiffFrameLevel = [
    ...BaseFrameLevel,
    FrameBaseEventCount[],
    FrameBaseSampleCount[]
];

export type FrameLevel = BaseFrameLevel | DiffFrameLevel;

export interface ProfileData {
    levels: FrameLevel[];
    stringTable: string[];
}


export interface NewFormatNode {
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

export type NewProfileData = {
    rows: NewFormatNode[][];
    stringTable: string[];
}
