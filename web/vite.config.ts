/// <reference types="vitest/config" />
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// En producción la SPA la sirve cmd/web (Go) con CSP estricta y /config.json;
// en desarrollo `npm run dev` usa public/config.json (API en localhost:8080).
export default defineConfig({
  plugins: [react()],
  // assetsInlineLimit 0: ninguna fuente o imagen se incrusta como data: URI;
  // la CSP de cmd/web (font-src 'self', img-src 'self' data:) bloquearía las fuentes.
  build: { outDir: 'dist', sourcemap: false, assetsInlineLimit: 0 },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/pruebas/preparar.ts'],
  },
})
