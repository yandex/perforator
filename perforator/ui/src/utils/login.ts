import Cookies from 'js-cookie';

import { uiFactory } from 'src/factory';


export const getUserLogin = (): Optional<string> => {
    if (!uiFactory().authorizationSupported()) {
        return undefined;
    }
    const cookie = uiFactory().loginCookie();
    const defaultUser = uiFactory().defaultUser();
    if (!cookie) {
        return defaultUser;
    }
    return Cookies.get(cookie) || defaultUser;
};
