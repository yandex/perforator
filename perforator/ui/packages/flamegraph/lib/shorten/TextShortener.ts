export interface TextShortenerTestCase {
    input: string;
    expected: string;
}

export interface TextShortener {
    shorten: (text: string) => Optional<string>;
    testCases?: TextShortenerTestCase[];
}
