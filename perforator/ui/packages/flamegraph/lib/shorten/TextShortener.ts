export interface TextShortenerTestCase {
    input: string;
    expected: string;
}

export interface TextShortener {
    /** If provided, a failed test guarantees that shorten() would return the input unchanged. */
    mayShorten?: RegExp;
    shorten: (text: string) => Optional<string>;
    testCases?: TextShortenerTestCase[];
}
