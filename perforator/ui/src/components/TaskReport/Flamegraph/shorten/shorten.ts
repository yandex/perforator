import { TEXT_SHORTENERS } from './shorteners';


const FILE_NAME_REGEX = /(.*)@.*/;

const cutFileName = (text: string): string => (
    (text.match(FILE_NAME_REGEX)?.[1] ?? text).trim()
);

const applyShorteners = (text: string): string => {
    for (const shortener of TEXT_SHORTENERS) {
        const shortened = shortener.shorten(text);
        if (shortened && shortened !== text) {
            return shortened;
        }
    }
    return text;
};

export const shorten = (text: string): string => {
    let result = text;
    result = cutFileName(result);
    result = applyShorteners(result);
    return result;
};
