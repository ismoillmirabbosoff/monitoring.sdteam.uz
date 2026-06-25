import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// Build natijasi Go server xizmat qiladigan web/dist ga chiqadi.
export default defineConfig({
  plugins: [vue()],
  build: { outDir: 'dist', emptyOutDir: true },
  server: {
    port: 5173,
    // dev rejimida API so'rovlarini Go backendga yo'naltirish
    proxy: {
      '/api': 'http://localhost:8080',
      '/health': 'http://localhost:8080',
    },
  },
})
