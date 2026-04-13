import type { Alias, PluginOption, UserConfig } from 'vite';
import { defineConfig } from 'vite';

import react from '@vitejs/plugin-react';

import { setupVite } from './src/vite/setup';


export default defineConfig(({ command }): UserConfig => {
    const uiHost = process.env.PERFORATOR_UI_HOST || '0.0.0.0';
    const uiPort = Number(process.env.PERFORATOR_UI_PORT || 1984);
    const uiUrl = `http://${uiHost}:${uiPort}`;

    const viteSettings = setupVite(command);

    const plugins: PluginOption[] = [
        ...(viteSettings?.plugins || []),
        react(),
    ];

    const aliases: Alias[] = [
        ...(viteSettings?.aliases || []),
        {
            find: /^~.+/,
            replacement: val => val.replace(/^~/, ''),
        },
        {
            find: 'src',
            replacement: '/src',
        },
    ];

    return {
        plugins,
        resolve: {
            alias: aliases,
            dedupe: [
                '@gravity-ui/uikit',
                '@gravity-ui/components',
                '@gravity-ui/icons',
                '@bem-react/classname',
                'react',
                'react-dom',
            ],
        },
        build: {
            sourcemap: true,
            ...(viteSettings.build || {}),
        },
        server: {
            host: uiHost,
            port: uiPort,
            proxy: {
                '/api/v0': {
                    target: process.env.PERFORATOR_API_URL || uiUrl,
                    changeOrigin: true,
                },
            },
            ...(viteSettings.https ? { https: viteSettings.https } : {}),
        },
    };
});
