import type { Coordinate } from '@perforator/flamegraph';

import type { PresetStep } from '@gravity-ui/onboarding';
import { createOnboarding, createPreset } from '@gravity-ui/onboarding';
import { Text } from '@gravity-ui/uikit';

import { ColorSwatch } from 'src/components/ColorSwatch/ColorSwatch';
import { LocalStorageKey } from 'src/const/localStorage';

import i18n from './onboarding/i18n';
import { createSuccessToast } from './toaster';


export const enum FlamegraphSteps {
    FlamegraphOverview = 'flamegraph-overview',
    FlamegraphClick = 'flamegraph-click',
    FlamegraphAltClick = 'flamegraph-alt-click',
    GoBack = 'go-back',
    ResetOmit = 'reset-omit',
    Search = 'search',
    SearchReset = 'search-reset',
    ShowMatchedStacks = 'show-matched-stacks',
    LeftHeavy = 'left-heavy',
    Final = 'final'
}

export const enum OnboardingNames {
    Basics = 'basics'
}

const progressSuccessToastHooks = {
    onStepPass: () => {
        createSuccessToast({ name: 'step', content: i18n('stepPassed') });
    },
};

const setIndexes: <
    I extends { hintParams?: H },
    H = I extends { hintParams?: infer Hint } ? Hint : never,
>(
    item: I,
    i: number,
    length: number,
) => I & { hintParams: H & { index?: string } } = (item, i, length) => ({
    ...item,
    hintParams: { ...item.hintParams, index: i !== 0 ? i18n('progress', { passed: i, total: length }) : undefined },
});


export const demoFlamegraphPreset = createPreset(
    ({ goNextStep, goPrevStep: _goPrevStep }) => {
        const steps = [
            {
                slug: FlamegraphSteps.FlamegraphOverview,
                name: i18n('overviewTitle'),
                description: '',
                hintParams: {
                    children:
                    <>{i18n('overviewIntro')}<ColorSwatch color={'rgb(205, 0, 0)'}/>{i18n('overviewOrange')}<ColorSwatch color={'rgb(96, 96, 205)'}/>{i18n('overviewBlue')}<ColorSwatch color="rgb(103, 178, 120)"/>{i18n('overviewGreen')}</>,
                    actions: [
                        {
                            children: i18n('next'),
                            view: 'action' as const,
                            onClick: () => {
                                goNextStep();
                            },
                        },
                    ],
                },
            },
            {
                slug: FlamegraphSteps.FlamegraphClick,
                name: i18n('clickTitle'),
                description: '',
                hintParams: {
                    children:
                        <>{i18n('clickIntro')}<Text variant={'code-1'}>inefficient_calc_sum</Text>{i18n('clickEnd')}</>,
                    highlightCoordinate: [8, 0] as Coordinate,
                },
                hooks: progressSuccessToastHooks,
            },
            {
                slug: FlamegraphSteps.GoBack,
                name: i18n('backTitle'),
                description: i18n('backDescription'),
                hintParams: {
                    highlightCoordinate: [0, 0] as Coordinate,
                },
                hooks: progressSuccessToastHooks,
            },
            {
                slug: FlamegraphSteps.FlamegraphAltClick,
                name: i18n('contextMenuTitle'),
                description: '',
                hintParams: {
                    children: <>{i18n('contextMenuIntro')}<Text variant={'code-1'}>shuffle_some_array</Text>{i18n('contextMenuEnd')}<br/>{i18n('contextMenuEfficiency')}<br/>{i18n('contextMenuAction')}</>,
                    highlightCoordinate: [8, 1] as Coordinate,
                },
                hooks: progressSuccessToastHooks,
            },
            {
                slug: FlamegraphSteps.ResetOmit,
                name: i18n('resetOmitTitle'),
                description: i18n('resetOmitDescription'),
                hintParams: {
                    className: '.flamegraph__clear-deletion',
                },
                hooks: progressSuccessToastHooks,
            },
            {
                slug: FlamegraphSteps.Search,
                name: i18n('searchTitle'),
                description: '',
                hintParams: {
                    children:
                        <>{i18n('searchIntro')}<Text variant="code-1">kernel</Text>{i18n('searchEnd')}</>,
                    className: '.flamegraph__button_search',
                },
                hooks: progressSuccessToastHooks,
            },
            {
                slug: FlamegraphSteps.ShowMatchedStacks,
                name: i18n('matchedStacksTitle'),
                hintParams: {
                    className: '.flamegraph__button_keep-only-found',
                },
                description: i18n('matchedStacksDescription'),
                hooks: progressSuccessToastHooks,
            },
            {
                slug: FlamegraphSteps.SearchReset,
                name: i18n('resetSearchTitle'),
                hintParams: {
                    className: '.flamegraph__clear',
                },
                description: i18n('resetSearchDescription'),
                hooks: progressSuccessToastHooks,
            },
            {
                slug: FlamegraphSteps.LeftHeavy,
                name: i18n('leftHeavyTitle'),
                description: i18n('leftHeavyDescription'),
                hintParams: {
                    className: '.flamegraph__switch_left-heavy',
                },
                hooks: progressSuccessToastHooks,
            },
            {
                slug: FlamegraphSteps.Final,
                name: i18n('completedTitle'),
                description: i18n('completedDescription'),
                hintParams: {
                    actions: [

                        {
                            children: i18n('finish'),
                            view: 'action' as const,
                            onClick: () => {
                                goNextStep();
                                controller.finishPreset(OnboardingNames.Basics);
                            },
                        },
                    ],
                },
                hooks: progressSuccessToastHooks,
            },
        ] satisfies PresetStep<FlamegraphSteps, any>[];

        return {
            enabled: true,
            name: OnboardingNames.Basics,
            visibility: 'visible',
            steps: steps.map((step, i) => setIndexes(step, i, steps.length - 1)),
        };
    },
);

function getFromLocalStorageAndParse(key: string) {
    try {
        return JSON.parse(localStorage.getItem(key) ?? '{}');
    } catch {
        return {};
    }
}

export const { useOnboardingHint, useOnboardingStep, useOnboardingPresets, presetsNames, useWizard, controller } = createOnboarding({
    baseState: {
        ...(getFromLocalStorageAndParse(LocalStorageKey.TutorialBase)),
        enabled: true,
    },
    config: {
        presets: {
            [OnboardingNames.Basics]: demoFlamegraphPreset,
        },
    },
    getProgressState: () => getFromLocalStorageAndParse(LocalStorageKey.TutorialProgress),
    progressState: getFromLocalStorageAndParse(LocalStorageKey.TutorialProgress),
    onSave: {
        state: async (state) => { localStorage.setItem(LocalStorageKey.TutorialBase, JSON.stringify(state)); },
        progress: async (progress) => { localStorage.setItem(LocalStorageKey.TutorialProgress, JSON.stringify(progress)); },
    },
    debugMode: true,
    logger: {
        level: 'debug',
    },

});

export function useWellKnownHint(preset: typeof presetsNames, step?: string) {
    const hint = useOnboardingHint();
    const currentPreset = hint.hint?.preset;
    const currentStep = hint.hint?.step;

    if (preset === currentPreset && (!step || currentStep?.slug === step)) {
        return hint;
    }
    return undefined;
}
