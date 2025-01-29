export const SHARE_IFRAME_HEIGHT = 500;

export const makeEmbedUrl = (url: string): string => {
    const result = new URL(url);
    result.searchParams.append('embed', '1');
    return result.href;
};

export type ShareFormat = string;
export type ShareStringBuilder = (url: string) => string;

export const SHARE_FORMAT_LINK: [ShareFormat, ShareStringBuilder] = ['Link', url => url];
export const SHARE_FORMAT_IFRAME: [ShareFormat, ShareStringBuilder] = [
    'Iframe',
    url => `<iframe src="${makeEmbedUrl(url)}" height="${SHARE_IFRAME_HEIGHT}px" />`,
];

export const SHARE_FORMATS: [ShareFormat, ShareStringBuilder][] = [
    SHARE_FORMAT_LINK,
    SHARE_FORMAT_IFRAME,
];
