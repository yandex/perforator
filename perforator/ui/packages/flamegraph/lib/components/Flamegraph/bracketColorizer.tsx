import React from 'react';


const map = {
    '(': ')',
    '<': '>',
    '[': ']',
};

const reverseMap = Object.fromEntries(Object.entries(map).map(([a, b]) => ([b, a])));

const opens = Object.keys(map);

interface BracketPairs {
    firstChar: string;
    lastChar: string;
    firstIndex: number;
    lastIndex: number;
    depth: number;
}

// func - some function name with generics, like std::function<void ()>::operator()() const
// every pair should be colorized
export function bracketColorizer(func: string) {
    const stack: { char: string; index: number }[] = [];
    const pairs: BracketPairs[] = [];
    function peek(): { char: string; index: number } | undefined {
        return stack[stack.length - 1];
    }
    for (let i = 0; i < func.length; i++) {
        const char = func[i];
        if (opens.includes(char)) {
            stack.push({ char: map[char], index: i });
        }
        else if (char === peek()?.char) {
            const other = stack.pop();
            pairs.push({ firstChar: reverseMap[char], firstIndex: other.index, lastChar: char, lastIndex: i, depth: stack.length });
        }
    }

    return pairs;
}

const MAX_COLORS = 5;

export function colorize(func: string): React.ReactNode {
    const pairs = bracketColorizer(func);
    const pairsByFirstIndex = Object.fromEntries(pairs.flatMap((pair) => ([[pair.firstIndex, pair.depth % MAX_COLORS], [pair.lastIndex, pair.depth % MAX_COLORS]])));

    const res: React.ReactNode[] = [];
    for (let i = 0; i < func.length; i++) {
        const char = func[i];
        if (i in pairsByFirstIndex) {
            res.push(<span key={`${char}-${i}`} className={`color-${pairsByFirstIndex[i]}`}>{char}</span>);
        } else {
            res.push(char);
        }
    }

    return res.reduce<React.ReactNode[]>((acc, iter) => {
        if (typeof iter === 'string' && acc.length > 0 && typeof acc[acc.length - 1] === 'string') {
            acc[acc.length - 1] = (acc[acc.length - 1] as string).concat(iter);
        } else {
            acc.push(iter);
        }
        return acc;
    }, []);
}
