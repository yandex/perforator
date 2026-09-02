import type { TextShortener } from './TextShortener';


// Performance-sensitive implementation choices below were measured on the
// full LLVM demangler corpus and on production flamegraphs in SpiderMonkey and
// Chromium. In particular, keep the numeric UTF-16 scanner, indexed traversal,
// primitive parallel arrays, lazy engine-selected pair table, formatter fast path,
// declarative dispatcher guard, and operator-only conversion-separator post-pass.
// Int32Array was 19.44% faster in SpiderMonkey, while Array was 20.73% faster in
// Chrome, so choose the representation once per runtime. Conversely, moving the
// conversion post-pass into the main scanner improved one Firefox dataset by
// 1.61 percentage points but worsened Chrome by 15.27 points. Please benchmark
// both engines and both datasets before replacing these choices with code that
// merely looks simpler.

const ANONYMOUS_NAMESPACE = '(anonymous namespace)';
const ABI_TAG_PREFIX = '[abi:';
const ABI_TAG = /\[abi:[^\]]+\]/g;
const MAX_TEMPLATE_ARGUMENTS_LENGTH = 32;
const UNICODE_IDENTIFIER_CHAR = /[\p{L}\p{N}]/u;
const UNICODE_WHITESPACE = /\s/u;
const CPP_PARSER_TRIGGER = /[<()\s]/u;
const REJECTED_NAMES = ['decltype', 'noexcept', 'sizeof', 'alignof', 'typeid', 'requires', 'throw', 'const', 'volatile', 'override', 'final'];
const SHORT_GO_RECEIVER = /\.\([^)]*\)\./;
const USE_TYPED_PAIR_OFFSETS = typeof navigator !== 'undefined' && navigator.userAgent.includes('Firefox/');

type PairOffsets = Int32Array | number[];

const enum CharacterCode {
    HorizontalTab = 9,
    CarriageReturn = 13,
    Space = 32,
    DoubleQuote = 34,
    DollarSign = 36,
    SingleQuote = 39,
    LeftParenthesis = 40,
    RightParenthesis = 41,
    Digit0 = 48,
    Digit9 = 57,
    Colon = 58,
    LessThanSign = 60,
    GreaterThanSign = 62,
    UppercaseA = 65,
    UppercaseZ = 90,
    LeftSquareBracket = 91,
    Backslash = 92,
    RightSquareBracket = 93,
    Underscore = 95,
    LowercaseA = 97,
    LowercaseO = 111,
    LowercaseZ = 122,
    LeftCurlyBracket = 123,
    RightCurlyBracket = 125,
    AsciiMax = 127,
}


interface Structure {
    pairOffsets: PairOffsets | undefined;
    groupOpens: number[];
    groupParents: number[];
    allParenAncestors: number[];
    parenGroups: number[];
    separators: number[];
    whitespaces: number[];
    operators: number[];
    valid: boolean;
}

interface ComponentChains {
    chains: Int32Array;
    leadings: Int32Array;
}

function isWhitespace(char: string | undefined): boolean {
    if (char === undefined) {
        return false;
    }
    const code = char.charCodeAt(0);
    return code === CharacterCode.Space
        || (code >= CharacterCode.HorizontalTab && code <= CharacterCode.CarriageReturn)
        || (code > CharacterCode.AsciiMax && UNICODE_WHITESPACE.test(char));
}

function isIdentifierChar(char: string | undefined): boolean {
    if (char === undefined) {
        return false;
    }
    const code = char.charCodeAt(0);
    return code === CharacterCode.DollarSign
        || code === CharacterCode.Underscore
        || (code >= CharacterCode.Digit0 && code <= CharacterCode.Digit9)
        || (code >= CharacterCode.UppercaseA && code <= CharacterCode.UppercaseZ)
        || (code >= CharacterCode.LowercaseA && code <= CharacterCode.LowercaseZ)
        || (code > CharacterCode.AsciiMax && UNICODE_IDENTIFIER_CHAR.test(char));
}

function previousNonSpace(text: string, offset: number): number {
    let index = offset - 1;
    while (index >= 0 && isWhitespace(text[index])) {
        --index;
    }
    return index;
}

function nextNonSpace(text: string, offset: number, end = text.length): number {
    let index = offset;
    while (index < end && isWhitespace(text[index])) {
        ++index;
    }
    return index;
}

function isOperatorWordAt(text: string, offset: number): boolean {
    return offset >= 0 && text.startsWith('operator', offset) && !isIdentifierChar(text[offset - 1]);
}

function endsWithOperatorWord(text: string, end: number): boolean {
    const wordEnd = previousNonSpace(text, end) + 1;
    return isOperatorWordAt(text, wordEnd - 'operator'.length);
}

function followsOperatorId(text: string, operatorOffset: number, offset: number): boolean {
    const tokenStart = operatorOffset + 'operator'.length;
    if (operatorOffset < 0 || tokenStart >= offset || offset - tokenStart > 3) {
        return false;
    }
    for (let index = tokenStart; index < offset; ++index) {
        if (isWhitespace(text[index]) || isIdentifierChar(text[index]) || text[index] === ':') {
            return false;
        }
    }
    if (text[offset - 1] === '<') {
        const next = text[nextNonSpace(text, offset + 1)];
        if (next === undefined || next === '(' || next === '=') {
            return false;
        }
    }
    // In operator<<<T>>, the second '<' still belongs to operator<<; the
    // third one opens the template argument list. The same longest-token rule
    // lets operator<<T> mean operator< followed by <T>.
    return text[offset - 1] !== '<' || text[offset + 1] !== '<';
}

function isTemplateOpen(text: string, offset: number): boolean {
    const previous = offset - 1;
    const next = nextNonSpace(text, offset + 1);
    if (previous < 0 || next >= text.length || isWhitespace(text[previous])) {
        return false;
    }
    if (isOperatorWordAt(text, offset - 'operator'.length)) {
        return false;
    }
    if (text[next] === '=' || text[next] === '<') {
        return false;
    }
    const previousChar = text[previous];
    return isIdentifierChar(previousChar)
        || previousChar === '>'
        || previousChar === ')'
        || previousChar === ']'
        || previousChar === '\''
        || previousChar === '"';
}

function isConversionOperatorAt(text: string, operatorOffset: number): boolean {
    if (operatorOffset < 0) {
        return false;
    }
    const target = nextNonSpace(text, operatorOffset + 'operator'.length);
    return target > operatorOffset + 'operator'.length && isIdentifierChar(text[target]);
}

function startsExpressionGrouping(text: string, open: number): boolean {
    const end = previousNonSpace(text, open) + 1;
    let start = end;
    while (start > 0 && isIdentifierChar(text[start - 1])) {
        --start;
    }
    for (let index = 0; index < REJECTED_NAMES.length; ++index) {
        if (rangeEquals(text, start, end, REJECTED_NAMES[index])) {
            return true;
        }
    }
    return false;
}

function filterConversionSeparators(
    text: string,
    pairOffsets: PairOffsets | undefined,
    groupOpens: number[],
    groupParents: number[],
    parenGroups: number[],
    separators: number[],
    operators: number[],
): void {
    const filteredSeparators = separators;
    let operatorPointer = 0;
    let parenPointer = 0;
    let lastOperator = -1;
    let conversionOperator = false;
    let parameterOpen = -1;
    let parameterClose = -1;
    let writeOffset = 0;
    for (let index = 0; index < separators.length; ++index) {
        const separator = separators[index];
        let operatorChanged = false;
        while (operatorPointer < operators.length && operators[operatorPointer] < separator) {
            lastOperator = operators[operatorPointer++];
            operatorChanged = true;
        }
        if (operatorChanged) {
            conversionOperator = isConversionOperatorAt(text, lastOperator);
            parameterOpen = -1;
            parameterClose = -1;
            while (parenPointer < parenGroups.length) {
                const group = parenGroups[parenPointer];
                const open = groupOpens[group];
                if (open <= lastOperator || groupParents[group] >= 0 || startsExpressionGrouping(text, open)) {
                    ++parenPointer;
                    continue;
                }
                parameterOpen = open;
                parameterClose = pairedOffset(pairOffsets, open);
                break;
            }
        }
        if (lastOperator < 0
            || !conversionOperator
            || (parameterOpen >= 0 && separator > parameterClose)) {
            filteredSeparators[writeOffset++] = separator;
        }
    }
    filteredSeparators.length = writeOffset;
}

// C++ [expr.prim.id] describes a function name as an unqualified-id or a
// qualified-id. Its components may be identifiers, template-ids, destructors,
// operator-function-ids, conversion-function-ids, or literal-operator-ids.
// Record balanced groups and the grammar-relevant tokens in one linear pass.
// Primitive parallel arrays avoid per-delimiter objects and Map entries.
function scanStructure(text: string): Structure {
    let pairOffsets: PairOffsets | undefined;
    const groupOpens: number[] = [];
    const groupParents: number[] = [];
    const allParenAncestors: number[] = [];
    const parenGroups: number[] = [];
    const separators: number[] = [];
    const whitespaces: number[] = [];
    const operators: number[] = [];
    const stack: number[] = [];
    let quote = 0;
    let escaped = false;
    let valid = true;

    for (let index = 0; index < text.length; ++index) {
        // Keep the hot scanner on numeric UTF-16 code units. SpiderMonkey
        // otherwise spends substantial time materializing/comparing one-char strings.
        const code = text.charCodeAt(index);
        if (quote !== 0) {
            if (escaped) {
                escaped = false;
            } else if (code === CharacterCode.Backslash) {
                escaped = true;
            } else if (code === quote) {
                quote = 0;
            }
            continue;
        }
        if (code === CharacterCode.SingleQuote || code === CharacterCode.DoubleQuote) {
            quote = code;
            continue;
        }
        if (stack.length === 0 && code === CharacterCode.LowercaseO && isOperatorWordAt(text, index)) {
            operators.push(index);
        }
        if (stack.length === 0 && (code === CharacterCode.Space
            || (code >= CharacterCode.HorizontalTab && code <= CharacterCode.CarriageReturn)
            || (code > CharacterCode.AsciiMax && UNICODE_WHITESPACE.test(text[index])))) {
            whitespaces.push(index);
        }
        if (code === CharacterCode.Colon && text.charCodeAt(index + 1) === CharacterCode.Colon && stack.length === 0) {
            separators.push(index);
            ++index;
            continue;
        }

        let opening = code === CharacterCode.LeftParenthesis
            || code === CharacterCode.LeftSquareBracket
            || code === CharacterCode.LeftCurlyBracket;
        if (code === CharacterCode.LessThanSign
            && (isTemplateOpen(text, index)
                || (stack.length === 0
                    && operators.length > 0
                    && followsOperatorId(text, operators[operators.length - 1], index)))) {
            opening = true;
        }
        if (opening) {
            pairOffsets ??= USE_TYPED_PAIR_OFFSETS ? new Int32Array(text.length) : new Array<number>(text.length);
            const group = groupOpens.length;
            const parent = stack.length === 0 ? -1 : stack[stack.length - 1];
            groupOpens.push(index);
            groupParents.push(parent);
            allParenAncestors.push(parent < 0 || (text.charCodeAt(groupOpens[parent]) === CharacterCode.LeftParenthesis && allParenAncestors[parent]) ? 1 : 0);
            stack.push(group);
            if (code === CharacterCode.LeftParenthesis) {
                parenGroups.push(group);
            }
            continue;
        }

        const expected = code === CharacterCode.RightParenthesis
            ? CharacterCode.LeftParenthesis
            : code === CharacterCode.RightSquareBracket
                ? CharacterCode.LeftSquareBracket
                : code === CharacterCode.RightCurlyBracket
                    ? CharacterCode.LeftCurlyBracket
                    : code === CharacterCode.GreaterThanSign
                        ? CharacterCode.LessThanSign
                        : 0;
        if (expected !== 0) {
            const group = stack[stack.length - 1];
            if (group !== undefined && text.charCodeAt(groupOpens[group]) === expected) {
                stack.pop();
                const start = groupOpens[group];
                pairOffsets![start] = index + 1;
                pairOffsets![index] = start + 1;
            } else if (code !== CharacterCode.GreaterThanSign) {
                valid = false;
            }
        }
    }

    if (operators.length > 0 && separators.length > 0) {
        filterConversionSeparators(text, pairOffsets, groupOpens, groupParents, parenGroups, separators, operators);
    }

    return {
        pairOffsets,
        groupOpens,
        groupParents,
        allParenAncestors,
        parenGroups,
        separators,
        whitespaces,
        operators,
        valid: valid && quote === 0 && stack.length === 0,
    };
}

function pairedOffset(pairOffsets: PairOffsets | undefined, offset: number): number {
    const encoded = pairOffsets?.[offset] ?? 0;
    return encoded === 0 ? -1 : encoded - 1;
}

function rangeEquals(text: string, start: number, end: number, expected: string): boolean {
    return end - start === expected.length && text.startsWith(expected, start);
}

function rangeStartsWith(text: string, start: number, end: number, expected: string): boolean {
    return end - start >= expected.length && text.startsWith(expected, start);
}

function startsSpecialName(text: string, start: number, end: number): boolean {
    return rangeEquals(text, start, end, ANONYMOUS_NAMESPACE)
        || rangeStartsWith(text, start, end, '\'lambda')
        || rangeStartsWith(text, start, end, '\'unnamed')
        || rangeStartsWith(text, start, end, '\'block-literal')
        || rangeStartsWith(text, start, end, '$_')
        || rangeStartsWith(text, start, end, '-[')
        || rangeStartsWith(text, start, end, '+[')
        || rangeStartsWith(text, start, end, 'friend ');
}

function lastOperatorInRange(structure: Structure, start: number, end: number, pointer = structure.operators.length - 1): number {
    while (pointer >= 0 && structure.operators[pointer] >= end) {
        --pointer;
    }
    const offset = structure.operators[pointer];
    return offset !== undefined && offset >= start ? offset : -1;
}

function skipDeclaratorPrefix(text: string, start: number, end: number): number {
    while (start < end && (isWhitespace(text[start]) || text[start] === '*' || text[start] === '&')) {
        ++start;
    }
    while (text[start] === '(') {
        const token = nextNonSpace(text, start + 1, end);
        if (text[token] !== '*' && text[token] !== '&') {
            break;
        }
        start = token + 1;
        while (start < end && (isWhitespace(text[start]) || text[start] === '*' || text[start] === '&')) {
            ++start;
        }
    }
    return start;
}

function isIdentifierRange(text: string, start: number, end: number): boolean {
    if (start >= end) {
        return false;
    }
    for (let index = start; index < end; ++index) {
        if (!isIdentifierChar(text[index])) {
            return false;
        }
    }
    return true;
}

function nonFinalComponentStart(text: string, start: number, end: number, structure: Structure, lastOperator: number): number {
    const leading = nextNonSpace(text, start, end);
    const trimmedEnd = previousNonSpace(text, end) + 1;
    if (leading >= trimmedEnd) {
        return -1;
    }
    if (lastOperator >= leading) {
        return lastOperator;
    }

    let lastSpace = -1;
    for (let index = leading; index < trimmedEnd; ++index) {
        const close = pairedOffset(structure.pairOffsets, index);
        if (close >= 0) {
            index = close;
            continue;
        }
        if (isWhitespace(text[index])) {
            lastSpace = index;
        }
    }
    if (startsSpecialName(text, leading, trimmedEnd)) {
        const suffix = nextNonSpace(text, lastSpace + 1, trimmedEnd);
        if (!rangeStartsWith(text, leading, trimmedEnd, 'friend ')
            && isIdentifierRange(text, suffix, trimmedEnd)
            && !isRejectedName(text, suffix, trimmedEnd)) {
            return suffix;
        }
        return leading;
    }
    return lastSpace < 0 ? leading : skipDeclaratorPrefix(text, lastSpace + 1, trimmedEnd);
}

function isFunctionQualifierSuffix(text: string, start: number, end: number): boolean {
    let offset = nextNonSpace(text, start, end);
    while (offset < end) {
        if (text.startsWith('const', offset) && !isIdentifierChar(text[offset + 5])) {
            offset = nextNonSpace(text, offset + 5, end);
            continue;
        }
        if (text.startsWith('volatile', offset) && !isIdentifierChar(text[offset + 8])) {
            offset = nextNonSpace(text, offset + 8, end);
            continue;
        }
        if (text[offset] === '&') {
            offset = nextNonSpace(text, offset + (text[offset + 1] === '&' ? 2 : 1), end);
            continue;
        }
        return false;
    }
    return true;
}

function buildComponentChains(text: string, structure: Structure): ComponentChains {
    const count = structure.separators.length + 1;
    const chains = new Int32Array(count);
    const leadings = new Int32Array(count);
    let operatorPointer = 0;
    let lastOperator = -1;
    for (let component = 0; component < count; ++component) {
        const start = component === 0 ? 0 : structure.separators[component - 1] + 2;
        const end = component < structure.separators.length ? structure.separators[component] : text.length;
        while (operatorPointer < structure.operators.length && structure.operators[operatorPointer] < end) {
            lastOperator = structure.operators[operatorPointer++];
        }
        const leading = nextNonSpace(text, start, end);
        leadings[component] = leading;
        let localStart = nonFinalComponentStart(text, start, end, structure, lastOperator >= start ? lastOperator : -1);
        const localStartChar = text[localStart];
        if (component > 0
            && localStart > leading
            && (localStartChar === 'c' || localStartChar === 'v' || localStartChar === '&')
            && isFunctionQualifierSuffix(text, localStart, end)) {
            localStart = leading;
        }
        if (localStart < 0) {
            chains[component] = -1;
        } else if (component === 0 || localStart > leading) {
            chains[component] = localStart;
        } else {
            chains[component] = chains[component - 1];
        }
    }
    return { chains, leadings };
}

function finalComponentStart(
    text: string,
    leading: number,
    end: number,
    lastOperator: number,
    lastWhitespace: number,
    whitespacePointer: number,
    decoratorStarts: Int32Array,
): number {
    if (leading >= end) {
        return -1;
    }
    if (startsSpecialName(text, leading, end)) {
        return leading;
    }
    if (lastOperator >= leading) {
        return lastOperator;
    }
    if (lastWhitespace < leading) {
        return leading;
    }
    let cached = decoratorStarts[whitespacePointer];
    if (cached === 0) {
        cached = skipDeclaratorPrefix(text, lastWhitespace + 1, text.length) + 1;
        decoratorStarts[whitespacePointer] = cached;
    }
    const start = cached - 1;
    return start < end ? start : -1;
}

function isRejectedName(text: string, start: number, end: number): boolean {
    const length = end - start;
    for (let wordIndex = 0; wordIndex < REJECTED_NAMES.length; ++wordIndex) {
        const word = REJECTED_NAMES[wordIndex];
        if (length === word.length && text.startsWith(word, start)) {
            return true;
        }
    }
    return endsWithOperatorWord(text, end);
}

function topLevelRequiresOffset(text: string, structure: Structure): number {
    let parenPointer = structure.parenGroups.length - 1;
    for (let index = structure.whitespaces.length - 1; index >= 0; --index) {
        const whitespace = structure.whitespaces[index];
        while (parenPointer >= 0) {
            const group = structure.parenGroups[parenPointer];
            if (structure.groupParents[group] >= 0
                || pairedOffset(structure.pairOffsets, structure.groupOpens[group]) >= whitespace) {
                --parenPointer;
                continue;
            }
            break;
        }
        const start = nextNonSpace(text, whitespace + 1);
        if (!text.startsWith('requires', start) || isIdentifierChar(text[start + 'requires'.length])) {
            continue;
        }
        if (parenPointer >= 0) {
            const group = structure.parenGroups[parenPointer];
            const close = pairedOffset(structure.pairOffsets, structure.groupOpens[group]);
            if (isFunctionQualifierSuffix(text, close + 1, whitespace)) {
                return whitespace;
            }
        }
    }
    return -1;
}

// Itanium local names embed the enclosing function encoding before ::. Its
// parameters and qualifiers are part of the qualified local name and are also
// retained by llvm-cxxfilt-18 --no-params.
function hasTopLevelLocalTail(text: string, structure: Structure): boolean {
    for (let groupIndex = 0; groupIndex < structure.parenGroups.length; ++groupIndex) {
        const group = structure.parenGroups[groupIndex];
        if (structure.groupParents[group] >= 0) {
            continue;
        }
        const open = structure.groupOpens[group];
        const close = pairedOffset(structure.pairOffsets, open);
        if (rangeEquals(text, open, close + 1, ANONYMOUS_NAMESPACE)) {
            continue;
        }
        let offset = nextNonSpace(text, close + 1);
        while (offset < text.length) {
            if (text.startsWith('const', offset) && !isIdentifierChar(text[offset + 5])) {
                offset = nextNonSpace(text, offset + 5);
                continue;
            }
            if (text.startsWith('volatile', offset) && !isIdentifierChar(text[offset + 8])) {
                offset = nextNonSpace(text, offset + 8);
                continue;
            }
            if (text[offset] === '&') {
                offset = nextNonSpace(text, offset + (text[offset + 1] === '&' ? 2 : 1));
                continue;
            }
            break;
        }
        if (text.startsWith('::', offset) && nextNonSpace(text, offset + 2) < text.length) {
            return true;
        }
    }
    return false;
}

function isParameterlessCallOperator(text: string, start: number, end: number, structure: Structure): boolean {
    const leading = nextNonSpace(text, start, end);
    const trimmedEnd = previousNonSpace(text, end) + 1;
    const operator = lastOperatorInRange(structure, leading, trimmedEnd);
    if (operator < 0) {
        return false;
    }
    const atomStart = operator + 'operator'.length;
    if (!text.startsWith('()', atomStart) && !text.startsWith('[]', atomStart)) {
        return false;
    }
    const templateStart = atomStart + 2;
    return templateStart === trimmedEnd || (text[templateStart] === '<' && pairedOffset(structure.pairOffsets, templateStart) === trimmedEnd - 1);
}

function extractQualifiedFunctionName(text: string): string | undefined {
    const sourceSuffix = text.indexOf(' @');
    const input = (sourceSuffix >= 0 ? text.slice(0, sourceSuffix) : text).trim();
    const structure = scanStructure(input);
    if (!structure.valid) {
        return undefined;
    }
    const requiresOffset = input.includes('requires') ? topLevelRequiresOffset(input, structure) : -1;
    if (requiresOffset >= 0) {
        return extractQualifiedFunctionName(input.slice(0, requiresOffset));
    }
    const hasLocalTail = hasTopLevelLocalTail(input, structure);
    const finalStart = (structure.separators[structure.separators.length - 1] ?? -2) + 2;
    if (isParameterlessCallOperator(input, finalStart, input.length, structure)) {
        return input;
    }
    const { chains, leadings } = buildComponentChains(input, structure);
    if (hasLocalTail) {
        const trailingStart = nextNonSpace(input, finalStart);
        const trailingEnd = previousNonSpace(input, input.length) + 1;
        const trailingParenthesis = input.indexOf('(', trailingStart);
        if ((trailingParenthesis < 0 || trailingParenthesis >= trailingEnd) && trailingStart < trailingEnd) {
            const localNameStart = chains[chains.length - 1];
            return input.slice(localNameStart >= 0 ? localNameStart : 0);
        }
    }

    // Candidate parentheses, namespace components, spaces, and operator tokens
    // are all visited by monotonic cursors. No candidate rescans its prefix.
    const decoratorStarts = new Int32Array(structure.whitespaces.length);
    let component = structure.separators.length;
    let whitespacePointer = structure.whitespaces.length - 1;
    let operatorPointer = structure.operators.length - 1;

    for (let index = structure.parenGroups.length - 1; index >= 0; --index) {
        const group = structure.parenGroups[index];
        const open = structure.groupOpens[group];
        if (isWhitespace(input[open - 1])) {
            continue;
        }
        while (component > 0 && structure.separators[component - 1] >= open) {
            --component;
        }
        while (whitespacePointer >= 0 && structure.whitespaces[whitespacePointer] >= open) {
            --whitespacePointer;
        }
        while (operatorPointer >= 0 && structure.operators[operatorPointer] >= open) {
            --operatorPointer;
        }
        if (input[open - 1] === ')') {
            const matchingOpen = pairedOffset(structure.pairOffsets, open - 1);
            if (!endsWithOperatorWord(input, matchingOpen)) {
                continue;
            }
        }
        const lastWhitespace = structure.whitespaces[whitespacePointer] ?? -1;
        const lastOperator = structure.operators[operatorPointer] ?? -1;
        const leading = leadings[component];
        const localStart = finalComponentStart(input, leading, open, lastOperator, lastWhitespace, whitespacePointer, decoratorStarts);
        if (localStart < 0) {
            continue;
        }
        const candidateStart = localStart > leading || component === 0 ? localStart : chains[component - 1];
        if (candidateStart < 0 || candidateStart >= open || isRejectedName(input, candidateStart, open)) {
            continue;
        }
        const parent = structure.groupParents[group];
        if (parent >= 0 && (!structure.allParenAncestors[group] || structure.groupOpens[parent] > candidateStart)) {
            continue;
        }
        if (hasLocalTail) {
            return input.slice(candidateStart, open).trimEnd();
        }
        return input.slice(candidateStart, open);
    }

    if (structure.whitespaces.length > 0 && !hasLocalTail && !/::(?:operator|friend)\s/.test(input)) {
        return input;
    }
    const lastWhitespace = structure.whitespaces[structure.whitespaces.length - 1] ?? -1;
    const lastOperator = structure.operators[structure.operators.length - 1] ?? -1;
    const leading = leadings[leadings.length - 1];
    const fallbackLocal = finalComponentStart(input, leading, input.length, lastOperator, lastWhitespace, structure.whitespaces.length - 1, decoratorStarts);
    const fallback = fallbackLocal > leading || structure.separators.length === 0 ? fallbackLocal : chains[chains.length - 2];
    return fallback >= 0 && !isRejectedName(input, fallback, input.length) ? input.slice(fallback) : input;
}

function compactComponent(text: string, start: number, end: number, structure: Structure): string {
    const pieces: string[] = [];
    let segmentStart = start;
    for (let index = start; index < end; ++index) {
        if (text[index] !== '<') {
            continue;
        }
        const close = pairedOffset(structure.pairOffsets, index);
        if (close < 0 || close >= end) {
            continue;
        }
        if (close - index - 1 >= MAX_TEMPLATE_ARGUMENTS_LENGTH) {
            pieces.push(text.slice(segmentStart, index), '<...>');
            segmentStart = close + 1;
        }
        index = close;
    }
    if (pieces.length === 0) {
        return text.slice(start, end);
    }
    pieces.push(text.slice(segmentStart, end));
    return pieces.join('');
}

function stripFunctionNameDecorations(name: string): string {
    let stripped = name;
    if (stripped.includes(ANONYMOUS_NAMESPACE)) {
        stripped = stripped.split(`${ANONYMOUS_NAMESPACE}::`).join('');
    }
    if (stripped.includes(ABI_TAG_PREFIX)) {
        stripped = stripped.replace(ABI_TAG, '');
    }
    return stripped;
}

function formatFunctionName(name: string): string {
    const hasAnonymousNamespace = name.includes(ANONYMOUS_NAMESPACE);
    const strippedName = stripFunctionNameDecorations(name);
    if (!hasAnonymousNamespace && (strippedName.length < MAX_TEMPLATE_ARGUMENTS_LENGTH + 2 || !strippedName.includes('<'))) {
        return strippedName;
    }
    const structure = scanStructure(strippedName);
    let changed = false;
    for (let component = 0; component <= structure.separators.length; ++component) {
        const start = component === 0 ? 0 : structure.separators[component - 1] + 2;
        const end = component < structure.separators.length ? structure.separators[component] : strippedName.length;
        const leading = nextNonSpace(strippedName, start, end);
        const trimmedEnd = previousNonSpace(strippedName, end) + 1;
        if (rangeEquals(strippedName, leading, trimmedEnd, ANONYMOUS_NAMESPACE)) {
            changed = true;
            break;
        }
        for (let index = start; index < end; ++index) {
            if (strippedName[index] === '<') {
                const close = pairedOffset(structure.pairOffsets, index);
                if (close >= 0 && close < end) {
                    if (close - index - 1 >= MAX_TEMPLATE_ARGUMENTS_LENGTH) {
                        changed = true;
                        break;
                    }
                    index = close;
                }
            }
        }
        if (changed) {
            break;
        }
    }
    if (!changed) {
        return strippedName;
    }

    const components: string[] = [];
    for (let component = 0; component <= structure.separators.length; ++component) {
        const start = component === 0 ? 0 : structure.separators[component - 1] + 2;
        const end = component < structure.separators.length ? structure.separators[component] : strippedName.length;
        const leading = nextNonSpace(strippedName, start, end);
        const trimmedEnd = previousNonSpace(strippedName, end) + 1;
        if (!rangeEquals(strippedName, leading, trimmedEnd, ANONYMOUS_NAMESPACE)) {
            components.push(compactComponent(strippedName, start, end, structure));
        }
    }
    return components.join('::');
}

function shortenCppName(text: string): string {
    if (text.includes('.(') && SHORT_GO_RECEIVER.test(text)) {
        return text;
    }
    // Go package paths are handled by the next shortener in the chain. C++
    // division expressions in template arguments must still reach this parser.
    if (!text.includes(' @') && text.includes('/') && !text.includes('::') && !text.includes('<') && !text.includes(' ') && !text.includes('operator/')) {
        return text;
    }
    const extracted = extractQualifiedFunctionName(text);
    return extracted === undefined ? text : formatFunctionName(extracted);
}

const testCases = [
    {
        input: 'name_space::class::method',
        expected: 'name_space::class::method',
    },
    {
        input: 'namespace1::namespace2::class::method',
        expected: 'namespace1::namespace2::class::method',
    },
    {
        input: '(anonymous namespace)::Class::Method',
        expected: 'Class::Method',
    },
    {
        input: 'class::method',
        expected: 'class::method',
    },
    {
        input: '(anonymous namespace)::function',
        expected: 'function',
    },
    {
        input: 'namespace::Class::method()',
        expected: 'namespace::Class::method',
    },
    {
        input: 'namespace::Class::method(int, std::string)',
        expected: 'namespace::Class::method',
    },
    {
        input: 'class::method @/main.cpp',
        expected: 'class::method',
    },
    {
        input: 'namespace1::namespace2::Class::method<int>() @/main.cpp',
        expected: 'namespace1::namespace2::Class::method<int>',
    },
    {
        input: 'namespace::Class::method()::$_42::operator+',
        expected: 'namespace::Class::method()::$_42::operator+',
    },
    {
        input: 'namespace::Class::~Class',
        expected: 'namespace::Class::~Class',
    },
    {
        input: 'namespace::Class<float>::method()',
        expected: 'namespace::Class<float>::method',
    },

    // Minimal cases retained from failures found by the standalone corpus loop.
    {
        input: 'grpc::Status grpc::internal::CallOpSendMessage::SendMessage<NUnifiedAgentProto::Request>(NUnifiedAgentProto::Request const&)',
        expected: 'grpc::internal::CallOpSendMessage::SendMessage<NUnifiedAgentProto::Request>',
    },
    {
        input: 'namespace::Class<12345678901234567890123456789012>::method<12345678901234567890123456789012>()',
        expected: 'namespace::Class<...>::method<...>',
    },
    {
        input: 'int (*h<>(int (*)() throw(int)))() throw(int)',
        expected: 'h<>',
    },
    {
        input: 'auto inline_func()::\'lambda\'<typename $T, typename $T0>($T, $T0)::operator()<int, int>($T, $T0) const',
        expected: 'inline_func()::\'lambda\'<typename $T, typename $T0>($T, $T0)::operator()<int, int>',
    },
    {
        input: 'decltype(foo<int>()) test8::X<int>::bar<int>() const',
        expected: 'test8::X<int>::bar<int>',
    },
    {
        input: 'test2::A<int>::friend f(...) requires True<T>',
        expected: 'test2::A<int>::friend f',
    },
    {
        input: 'Outer<Marp>::operator Muncher<Merp>()::S::operator Merp()',
        expected: 'Outer<Marp>::operator Muncher<Merp>()::S::operator Merp',
    },
    {
        input: 'void (anonymous namespace)::f()',
        expected: 'f',
    },
    {
        input: 'void f<int>(int) requires requires { (T)(); }',
        expected: 'f<int>',
    },
    {
        input: 'void Casts::implicit<4u>(enable_if<4u / 4, void>::type*)',
        expected: 'Casts::implicit<4u>',
    },
    {
        input: 'void std::__y1::__invoke_r[abi:ne210000]<void, NYaIO::TIOLoop::ForkLoop()::$_0&>(NYaIO::TIOLoop::ForkLoop()::$_0&)',
        expected: 'std::__y1::__invoke_r<...>',
    },
    {
        input: 'net.(*conn).Read',
        expected: 'net.(*conn).Read',
    },
    {
        input: 'bool ns::operator==<char, std::__y1::char_traits<char>, 1>(int)',
        expected: 'ns::operator==<...>',
    },
    {
        input: 'bool ns::operator==[abi:x]<char, std::__y1::char_traits<char>, 1>(int)',
        expected: 'ns::operator==<...>',
    },
    {
        input: '__add_pointer(T) std::any_cast<T>(std::any*)',
        expected: 'std::any_cast<T>',
    },
    {
        input: 'A::operator n::B<C, D> const&() const',
        expected: 'A::operator n::B<C, D> const&',
    },
    {
        input: 'decltype(fp()) N::RunNoExcept<N::G::operator()()::\'lambda\'()>(N::G::operator()()::\'lambda\'()&&)',
        expected: 'N::RunNoExcept<N::G::operator()()::\'lambda\'()>',
    },
    {
        input: 'auto N::T::f<0>() requires I < sizeof...(T) - 1',
        expected: 'N::T::f<0>',
    },
    {
        input: 'void N::f()::\'lambda\'()::operator()()',
        expected: 'N::f()::\'lambda\'()::operator()',
    },
    {
        input: 'N::R N::T::f()::\'lambda\'()::operator()()',
        expected: 'N::T::f()::\'lambda\'()::operator()',
    },
    {
        input: 'N::make()::$_0 N::f<N::make()::$_0, N::SomeLongTypeName>(N::make()::$_0)',
        expected: 'N::f<...>',
    },
    {
        input: 'N::make() const::$_0 N::f<N::make() const::$_0, N::SomeLongType>(N::make() const::$_0)',
        expected: 'N::f<...>',
    },
    {
        input: 'auto N::T::f<0>() requires N::C<T>',
        expected: 'N::T::f<0>',
    },
    {
        input: 'auto N::T::f<0>() requires N::C::\'lambda\'()::operator()() const',
        expected: 'N::T::f<0>',
    },
    {
        input: 'N::A& N::A::operator=<(N::E)0>(N::A&&)',
        expected: 'N::A::operator=<(N::E)0>',
    },
    {
        input: '(anonymous namespace)::operator<<',
        expected: 'operator<<',
    },
    {
        input: 'void A::B::Emit<A::operator()() const::\'lambda\'(T)>(U)',
        expected: 'A::B::Emit<...>',
    },
];

export const cpp: TextShortener = {
    mayShorten: CPP_PARSER_TRIGGER,
    shorten: shortenCppName,
    testCases,
};
