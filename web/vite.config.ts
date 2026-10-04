import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// En desarrollo, la API y /healthz se piden al binario de Go. En produccion es el mismo origen.
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
      '/healthz': 'http://localhost:8080',
    },
  },
})
