import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

import { setupVite } from "./src/vite/setup";

// https://vitejs.dev/config/
export default defineConfig(() => {
    const viteSettings = setupVite();

    return {
        plugins: [
            react(),
        ],
        resolve: {
            alias: [
                ...(viteSettings.aliases || []),
                {
                    find: /^~.+/,
                    replacement: val => val.replace(/^~/, ''),
                },
                {
                    find: 'src',
                    replacement: '/src',
                },
            ],
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
            rollupOptions: {
                output: {
                    entryFileNames: `assets/[name].js`,
                    chunkFileNames: `assets/[name].js`,
                    assetFileNames: `assets/[name].[ext]`,
                    inlineDynamicImports: true,
                    format: 'iife',
                },
            }
        },
    };
});
