import { TEXT_SHORTENERS } from './shorteners';


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
    result = applyShorteners(result);
    return result;
};
