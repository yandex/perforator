export interface Rum {
    finishDataLoading?: (value: string) => void;
    finishDataRendering?: (value: string) => void;
    makeSpaSubPage?: (value: string) => void;
    startDataRendering?: (value: string, renderType: string, shouldCall: boolean) => void;
}

export const fakeRum: Rum = {};
