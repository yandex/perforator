import React from 'react';

import { SegmentedRadioGroup } from '@gravity-ui/uikit';

import { QueryLanguageHelpPopover } from 'src/components/QueryLanguageEditor';
import { QueryLanguageEditorImpl, QueryLanguageFallback } from 'src/components/QueryLanguageEditor/lazy';
import { QuerySuggestProvider } from 'src/providers/QuerySuggestProvider/QuerySuggestProvider';
import { cn } from 'src/utils/cn';
import { EMPTY_SELECTOR } from 'src/utils/selector';

import { TokensInput } from './TokensInput';
import { makeSelectorFromTokensString, parseSelectorToTokensString } from './utils';

import './SwitchableSelectorInput.scss';


const b = cn('switchable-selector-input');

type InputMode = 'tokens' | 'selector';


export interface SwitchableSelectorInputProps {
    selector?: string;
    onUpdate: (selector: Optional<string>) => void;
    onSelectorChange?: (selector: Optional<string>) => void;
    height?: string;
    className?: string;
    wrapperClassName?: string;
}

export const SwitchableSelectorInput: React.FC<SwitchableSelectorInputProps> = ({
    onUpdate,
    onSelectorChange,
    selector,
    height = '32px',
    wrapperClassName,
    className,
}) => {
    const [mode, setMode] = React.useState<InputMode>('tokens');

    const handleTokensUpdate = React.useCallback((tokens: Optional<string>) => {
        const newSelector = makeSelectorFromTokensString(tokens);
        onUpdate(newSelector);
        onSelectorChange?.(newSelector);
    }, [onUpdate, onSelectorChange]);

    const handleSelectorUpdate = React.useCallback((newSelector: Optional<string>) => {
        onUpdate(newSelector);
    }, [onUpdate]);

    const handleSelectorChange = React.useCallback((newSelector: Optional<string>) => {
        onSelectorChange?.(newSelector);
    }, [onSelectorChange]);

    return (
        <QuerySuggestProvider>
            <div className={b(null, className)}>
                <div className={b('wrapper', wrapperClassName)}>
                    {mode === 'tokens' ? (
                        <TokensInput
                            tokens={parseSelectorToTokensString(selector ?? EMPTY_SELECTOR)}
                            initialTokens={parseSelectorToTokensString(selector ?? EMPTY_SELECTOR)}
                            onUpdate={handleTokensUpdate}
                        />
                    ) : (
                        <React.Suspense fallback={<QueryLanguageFallback className={className} selector={selector}/>}>
                            <QueryLanguageEditorImpl
                                selector={selector}
                                onUpdate={handleSelectorUpdate}
                                onSelectorChange={handleSelectorChange}
                                height={height}
                                className={b('editor')}
                                wrapperClassName={b('editor-wrapper')}
                            />
                        </React.Suspense>
                    )}
                </div>
                <SegmentedRadioGroup
                    value={mode}
                    onUpdate={setMode}
                    className={b('switcher')}
                >
                    <SegmentedRadioGroup.Option value="tokens">
                        Tokens
                    </SegmentedRadioGroup.Option>
                    <SegmentedRadioGroup.Option value="selector">
                        Selector
                    </SegmentedRadioGroup.Option>
                </SegmentedRadioGroup>
                <QueryLanguageHelpPopover/>
            </div>
        </QuerySuggestProvider>
    );
};
