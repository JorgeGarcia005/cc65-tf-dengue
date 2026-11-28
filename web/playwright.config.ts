import { defineConfig, devices } from '@playwright/test'

// Pruebas de punta a punta de Vigía contra el sistema desplegado (#70).
// Las corre scripts/evidencias-tb2.sh e2e (paso 14 de docs/guias/jose-70.md)
// con el modo servicio arriba y DEMO=true. Las credenciales llegan por
// variables de entorno (las de deploy/.env de la corrida), nunca en el código.
//
//   VIGIA_URL   SPA   (http://localhost:8081)
//   VIGIA_API   API   (http://localhost:8080)
//   CAPTURAS    carpeta de las capturas del informe (../docs/informe/tb2/capturas)
//   CHROME      ejecutable de Chrome si no se instaló el de Playwright
export default defineConfig({
  testDir: './e2e',
  testMatch: '**/*.e2e.ts', // no *.test.* ni *.spec.*: Vitest no las toma
  fullyParallel: false,
  workers: 1, // un solo clúster: los flujos van uno tras otro
  retries: 0,
  timeout: 180_000,
  expect: { timeout: 15_000 },
  reporter: [['list'], ['html', { open: 'never', outputFolder: 'playwright-report' }]],
  outputDir: 'test-results',
  use: {
    ...devices['Desktop Chrome'],
    baseURL: process.env.VIGIA_URL ?? 'http://localhost:8081',
    viewport: { width: 1280, height: 900 },
    locale: 'es-PE',
    timezoneId: 'America/Lima',
    trace: 'retain-on-failure',
    launchOptions: process.env.CHROME ? { executablePath: process.env.CHROME } : {},
  },
})
