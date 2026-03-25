import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'
import removeConsole from 'vite-plugin-remove-console'

export default defineConfig(({ mode }) => {
    const env = loadEnv(mode, `${process.cwd()}/env`)

    return {
        plugins: [
            vue(),
            env.VITE_APP_USE_SCRIPT === 'prod' ? removeConsole({
                external: ['src/assets/iconfont/iconfont.js'],  // 排除 iconfont.js 文件
            }) : null,
        ],
        resolve: {
            alias: {
                '@': fileURLToPath(new URL('./src', import.meta.url)),
            },
        },
        server: {
            host: '0.0.0.0',
            port: 3011,
            proxy: {
                '/devs': {
                    target: 'https://106.55.158.199/',
                    secure: false,
                },
                '/api': {
                    target: 'https://106.55.158.199/',
                    secure: false,
                    changeOrigin: true,
                },
                '/signout': {
                    target: 'https://106.55.158.199/',
                    secure: false,
                },
                '/alive': {
                    target: 'https://106.55.158.199/',
                    secure: false,
                },
                '/get': {
                    target: 'https://106.55.158.199/',
                    secure: false,
                },
                '^/cmd/.*': {
                    target: 'https://106.55.158.199/',
                    secure: false,
                },
                '^/connect/.*': {
                    ws: true,
                    target: 'https://106.55.158.199/',
                },
                '^/web/*': {
                    target: 'https://106.55.158.199/',
                },
                '/auth-config': {
                    target: 'https://106.55.158.199/',
                    secure: false,
                    changeOrigin: true,
                },
            },
        },
        // 关闭 css 警告
        css: {
            preprocessorOptions: {
                scss: {
                    api: 'modern-compiler',
                },
            },
        },
    }
})
