import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  base: './',
  plugins: [vue()],
  build: {
    outDir: process.env.ROOM_UI_OUT_DIR || '../gateway/internal/roomui/static',
    emptyOutDir: true,
    sourcemap: false,
  },
  test: {
    environment: 'jsdom',
  },
})
