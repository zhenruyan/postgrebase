import { defineConfig }           from 'vite';
import { svelte, vitePreprocess } from '@sveltejs/vite-plugin-svelte';

const backendProxyTarget = process.env.PB_BACKEND_PROXY_TARGET || 'http://127.0.0.1:8090';
const productionCsp = [
    "default-src 'self'",
    "style-src 'self' 'unsafe-inline'",
    "img-src 'self' http://127.0.0.1:* data: blob:",
    "connect-src 'self' http://127.0.0.1:*",
    "script-src 'self' 'sha256-GRUzBA7PzKYug7pqxv5rJaec5bwDCw1Vo6/IXwvD3Tc='",
].join('; ');
const productionCspPlugin = {
    name: 'postgrebase-production-csp',
    apply: 'build',
    transformIndexHtml(html) {
        return html.replace(
            '<meta name="viewport" content="width=device-width, initial-scale=1.0" />',
            `<meta name="viewport" content="width=device-width, initial-scale=1.0" />\n    <meta http-equiv="Content-Security-Policy" content="${productionCsp}" />`
        );
    },
};

// see https://vitejs.dev/config
export default defineConfig({
    server: {
        port: 3000,
        proxy: {
            '/api': {
                target: backendProxyTarget,
                changeOrigin: true,
                ws: true,
            },
        },
    },
    envPrefix: 'PB',
    base: './',
    build: {
        chunkSizeWarningLimit: 1000,
        reportCompressedSize: false,
    },
    plugins: [
        productionCspPlugin,
        svelte({
            preprocess: [vitePreprocess()],
            onwarn: (warning, handler) => {
                if (warning.code.startsWith('a11y-')) {
                    return; // silence a11y warnings
                }
                handler(warning);
            },
        }),
    ],
    resolve: {
        alias: {
            '@': __dirname + '/src',
            '@sdk': __dirname + '/src/pocketbase-sdk',
            'pocketbase': __dirname + '/src/pocketbase-sdk/index.ts',
        }
    },
})
