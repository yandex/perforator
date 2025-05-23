import type { Alias, BuildOptions, PluginOption, ServerOptions } from 'vite';


export interface ViteSettings {
    plugins?: PluginOption[];
    aliases?: Alias[];
    build?: BuildOptions;
    https?: ServerOptions['https'];
}
