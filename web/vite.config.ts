import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  // Relative base: the built bundle works at the root and behind /matcharr
  base: './',
  plugins: [vue()],
  server: {
    proxy: {
      '/api': 'http://localhost:8585',
    },
  },
})
