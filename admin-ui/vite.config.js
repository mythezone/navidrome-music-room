import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  base: './',
  plugins: [react()],
  build: {
    outDir: process.env.ADMIN_UI_OUT_DIR || '../gateway/internal/adminui/static',
    emptyOutDir: true,
    sourcemap: false,
  },
})
