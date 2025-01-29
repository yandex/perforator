export const setPageTitle = (title: Optional<string>) => {
    document.title = (title ? `${title} | ` : '') + 'Perforator';
};
