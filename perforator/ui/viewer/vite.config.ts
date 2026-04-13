import { defineConfig, searchForWorkspaceRoot } from "vite";
import react from "@vitejs/plugin-react";

// https://vitejs.dev/config/
export default defineConfig({
    plugins: [
        react(),
    ],
    resolve: {
        alias: [
            {
                find: /^~.+/,
                replacement: val => val.replace(/^~/, ''),
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
            }
        }
    },
});
