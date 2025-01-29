import { UIFactory } from './UIFactory';


export const buildFactory = async () => {};
const uiFactoryLocal = new UIFactory();
export const uiFactory = () => uiFactoryLocal;
