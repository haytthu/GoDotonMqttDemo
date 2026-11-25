import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

const proxy = {
  '/api': process.env.DEMO_API_URL || 'http://127.0.0.1:8080',
}

export default defineConfig({
  plugins: [vue()],
  server: { proxy },
  preview: { proxy },
})

