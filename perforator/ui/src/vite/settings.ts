import type { Alias, PluginOption } from 'vite';


export interface ViteSettings {
    plugins?: PluginOption[];
    aliases?: Alias[];
}
