import React, { useEffect, useRef } from 'react';

import { useSearchParams } from 'react-router-dom';

import { type ExternalFlamegraphApi, prerenderColors } from '@perforator/flamegraph';

import type { CommonPreset } from '@gravity-ui/onboarding';
import { useForkRef, useThemeType } from '@gravity-ui/uikit';

import { Fullscreen } from 'src/components/Fullscreen/Fullscreen';
import { FullscreenProvider } from 'src/components/Fullscreen/FullscreenProvider';
import { Visualisation } from 'src/components/TaskReport/Visualisation/Visualisation';
import profileData from 'src/demo.json';
import { useUserSettings } from 'src/providers/UserSettingsProvider';
import { controller, FlamegraphSteps, OnboardingNames, useOnboardingPresets, useOnboardingStep, useWellKnownHint } from 'src/utils/onboarding';

import { HighlightVeil } from '../Veil/Veil';


const LEFT_OFFSET = 12;

const SEARCH_STRING_ONBOARDING = 'kernel';

function eqArrays<T extends any = any>(a: Array<T>, b: Array<T>): boolean {
    return a.every(ai => b.includes(ai)) && b.every(bi => a.includes(bi));
}

export const Demo: React.FC = () => {
    const theme = useThemeType();
    const hint = useWellKnownHint(OnboardingNames.Basics);
    const step = hint?.hint?.step;
    const { userSettings } = useUserSettings();
    const flameRef = React.useRef<ExternalFlamegraphApi | null>(null);
    const setRef = (val: ExternalFlamegraphApi | null) => {
        flameRef.current = val;
    };

    const { runPreset, resetPresetProgress } = useOnboardingPresets();
    const flamegraphOverviewStep = useOnboardingStep(FlamegraphSteps.FlamegraphOverview);
    const flamegraphClickStep = useOnboardingStep(FlamegraphSteps.FlamegraphClick);
    const flamegraphAltClickStep = useOnboardingStep(FlamegraphSteps.FlamegraphAltClick);
    const goBackStep = useOnboardingStep(FlamegraphSteps.GoBack);
    const resetOmitStep = useOnboardingStep(FlamegraphSteps.ResetOmit);
    const searchStep = useOnboardingStep(FlamegraphSteps.Search);
    const showMatchedStacksStep = useOnboardingStep(FlamegraphSteps.ShowMatchedStacks);
    const searchResetStep = useOnboardingStep(FlamegraphSteps.SearchReset);
    const leftHeavyStep = useOnboardingStep(FlamegraphSteps.LeftHeavy);
    const finalStep = useOnboardingStep(FlamegraphSteps.Final);

    const commonRef = useForkRef(
        flamegraphOverviewStep.ref,
        flamegraphClickStep.ref,
        flamegraphAltClickStep.ref,
        goBackStep.ref,
        resetOmitStep.ref,
        searchStep.ref,
        showMatchedStacksStep.ref,
        searchResetStep.ref,
        leftHeavyStep.ref,
        finalStep.ref,
    );
    const [_, setParams] = useSearchParams();

    const highlightedDomNode = useRef<HTMLElement | null>();
    useEffect(() => {
        if (step?.hintParams?.className) {
            const el = document.querySelector<HTMLElement>(step?.hintParams?.className);
            highlightedDomNode.current = el;
        }
    }, [step?.slug]);

    useEffect(() => {
        setParams({});
        resetPresetProgress(OnboardingNames.Basics);
        runPreset(OnboardingNames.Basics);
        return () => {
            const configSteps = (
                controller.options.config.presets
                    .basics as CommonPreset<
                        OnboardingNames, FlamegraphSteps
                    >
            ).steps.map((st) => st.slug);
            if (
                !eqArrays(
                    controller.state.progress?.presetPassedSteps.basics ?? [],
                    configSteps,
                )
            ) {
                resetPresetProgress(OnboardingNames.Basics);
            }
        };
    }, []);

    const prerenderedNewData = React.useMemo(() => {
        if (profileData) {
            return prerenderColors(profileData, { theme });
        }
        return null;
    }, [theme]);

    const enableHighlightVeil =
        Boolean(step?.hintParams?.highlightCoordinate) ||
        step?.slug === FlamegraphSteps.ResetOmit ||
        step?.slug === FlamegraphSteps.Search ||
        step?.slug === FlamegraphSteps.ShowMatchedStacks ||
        step?.slug === FlamegraphSteps.SearchReset ||
        step?.slug === FlamegraphSteps.LeftHeavy;

    const highlightCoordinate = step?.hintParams?.highlightCoordinate;


    const getHighlightVeilData = React.useCallback(() => {
        if (flameRef.current?.offsetter && highlightCoordinate) {
            const offsetter = flameRef.current.offsetter;
            const levelHeight = offsetter.levelHeight;
            const node = offsetter.rows[highlightCoordinate[0]][highlightCoordinate[1]];
            const x = node.x! + LEFT_OFFSET;
            const rect = flameRef.current.canvas.getBoundingClientRect();
            const baseTopOffset = rect.top + window.scrollY;
            const baseXOffset = 0;
            const y = highlightCoordinate[0] * levelHeight;
            const width = offsetter.countWidth(node);
            return {
                x: x + baseXOffset,
                y: y + baseTopOffset,
                width,
                height: levelHeight - 2,
                topOffset: 0,
            };
        }
        const el = highlightedDomNode.current;
        const rect = el?.getBoundingClientRect();
        if (el && rect) {
            return {
                x: el.offsetLeft + LEFT_OFFSET,
                y: rect.y,
                width: rect.width,
                height: rect.height,
                topOffset: 0,
            };
        }
        return undefined;
    }, [profileData, highlightCoordinate, step?.slug]);

    return (

        <FullscreenProvider initialEnalbed={true}>
            <Fullscreen>
                <div ref={commonRef}></div>
                <Visualisation
                    onFrameClick={(_ev, _frame) => {
                        if (hint?.open) {
                            const slug = hint.hint?.step.slug;
                            if (slug === FlamegraphSteps.FlamegraphClick) {
                                flamegraphClickStep.pass();
                            }
                            if (slug === FlamegraphSteps.GoBack) {
                                goBackStep.pass?.();
                            }
                        }
                    }}
                    onContextItemClick={() => flamegraphAltClickStep.pass()}
                    onFrameAltClick={() => flamegraphAltClickStep.pass()}
                    setOffsetterRef={setRef}
                    onSearch={(searchString) => {
                        if (searchString === SEARCH_STRING_ONBOARDING && step?.slug === FlamegraphSteps.Search) {
                            searchStep.pass();
                        }
                    }}
                    onSearchReset={() => searchResetStep.pass()}
                    onKeepOnlyFound={() => showMatchedStacksStep.pass()}
                    onResetOmitted={() => resetOmitStep.pass()}
                    onChangeLeftHeavy={() => leftHeavyStep.pass()}
                    disableHoverPopup={enableHighlightVeil}
                    loading={false} isDiff={false} profileData={prerenderedNewData} theme={theme} userSettings={userSettings} />

                {
                    enableHighlightVeil && <HighlightVeil getHighlightVeilData={getHighlightVeilData} />
                }
            </Fullscreen>
        </FullscreenProvider>
    );
};
