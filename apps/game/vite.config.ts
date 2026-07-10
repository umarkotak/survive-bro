/// <reference types="vitest/config" />

import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 3702,
    strictPort: true,
    proxy: {
      '/api': {
        target: 'http://localhost:3701',
        changeOrigin: false,
      },
      '/health': {
        target: 'http://localhost:3701',
        changeOrigin: false,
      },
      '/metrics': {
        target: 'http://localhost:3701',
        changeOrigin: false,
      },
      '/ws': {
        target: 'http://localhost:3701',
        ws: true,
        changeOrigin: false,
      },
    },
  },
  preview: {
    port: 3702,
    strictPort: true,
  },
  test: {
    environment: 'node',
    include: ['src/**/*.test.ts'],
  },
})
