import path from 'node:path'
import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import { defineConfig, loadEnv } from 'vite'
import { viteSingleFile } from 'vite-plugin-singlefile'

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const backendUrl = env.VITE_DEFAULT_SERVER_URL || 'http://localhost:3000'

  return {
    plugins: [react(), tailwindcss(), viteSingleFile()],
    resolve: {
      alias: {
        '@': path.resolve(__dirname, './src'),
      },
    },
    server: {
      proxy: {
        '/gowa': {
          target: backendUrl,
          changeOrigin: true,
          ws: true,
          rewrite: (p) => p.replace(/^\/gowa/, ''),
        },
        '/devices': { target: backendUrl, changeOrigin: true },
        '/app': { target: backendUrl, changeOrigin: true },
        '/send': { target: backendUrl, changeOrigin: true },
        '/chat': { target: backendUrl, changeOrigin: true },
        '/group': { target: backendUrl, changeOrigin: true },
        '/user': { target: backendUrl, changeOrigin: true },
        '/message': { target: backendUrl, changeOrigin: true },
        '/newsletter': { target: backendUrl, changeOrigin: true },
        '/call': { target: backendUrl, changeOrigin: true },
        '/schedule': { target: backendUrl, changeOrigin: true },
        '/ws': { target: backendUrl, changeOrigin: true, ws: true },
      },
    },
  }
})
