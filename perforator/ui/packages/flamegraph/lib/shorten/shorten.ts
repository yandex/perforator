import { TEXT_SHORTENERS } from './shorteners';


const applyShorteners = (text: string): string => {
    for (let i = 0; i < TEXT_SHORTENERS.length; i++) {
        const shortener = TEXT_SHORTENERS[i];
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
