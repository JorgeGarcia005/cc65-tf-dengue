import { test, expect, type Page } from '@playwright/test'
import { mkdirSync } from 'node:fs'

// Flujos por rol de Vigía y capturas del informe TB2. Cada captura tiene un
// nombre fijo: el informe la busca por nombre (docs/guias/jose-70.md, paso 14).

const CAPTURAS = process.env.CAPTURAS ?? '../docs/informe/tb2/capturas'
const API = (process.env.VIGIA_API ?? 'http://localhost:8080').replace(/\/$/, '')
mkdirSync(CAPTURAS, { recursive: true })

function variable(nombre: string): string {
  const v = process.env[nombre]
  if (!v) throw new Error(`falta la variable ${nombre} (las de deploy/.env de esta corrida)`)
  return v
}
const admin = () => ({ correo: variable('VIGIA_ADMIN_CORREO'), clave: variable('VIGIA_ADMIN_CLAVE') })
const analista = () => ({ correo: variable('VIGIA_ANALISTA_CORREO'), clave: variable('VIGIA_ANALISTA_CLAVE') })

async function captura(page: Page, nombre: string, completa = true) {
  await page.screenshot({ path: `${CAPTURAS}/${nombre}.png`, fullPage: completa, animations: 'disabled' })
}

// Solo una sección: en el panel de administración la página completa mide
// casi 4 000 px y, a la altura de una hoja, no se leería.
async function capturaDe(region: ReturnType<Page['locator']>, nombre: string) {
  await region.screenshot({ path: `${CAPTURAS}/${nombre}.png`, animations: 'disabled' })
}

async function entrar(page: Page, c: { correo: string; clave: string }) {
  await page.goto('/login')
  await page.getByLabel('Correo').fill(c.correo)
  await page.getByLabel('Clave').fill(c.clave)
  await page.getByRole('button', { name: 'Entrar' }).click()
  await expect(page).not.toHaveURL(/\/login/)
}

const menu = (page: Page) => page.getByRole('navigation', { name: 'Principal' })

test.describe.configure({ mode: 'serial' })

test('login: una clave errada da un mensaje genérico y no entra', async ({ page }) => {
  await page.goto('/login')
  await page.getByLabel('Correo').fill(admin().correo)
  await page.getByLabel('Clave').fill('clave-incorrecta-e2e')
  await page.getByRole('button', { name: 'Entrar' }).click()
  await expect(page.getByRole('alert')).toContainText(/correo o clave incorrectos/i)
  await expect(page).toHaveURL(/\/login/)
  await captura(page, 'e2e-01-login')
})

test('analista: sin «Clúster» en el menú y 403 propio en /admin', async ({ page }) => {
  await entrar(page, analista())
  await expect(menu(page).getByRole('link', { name: 'Inicio' })).toBeVisible()
  await expect(menu(page).getByRole('link', { name: 'Clúster' })).toHaveCount(0)
  await page.goto('/admin')
  await expect(page.locator('#titulo-403')).toBeVisible()
  await captura(page, 'e2e-02-analista-403')
})

test('inicio: las 10 prioridades, con la evaluación del modelo que sirve la API', async ({ page }) => {
  await entrar(page, analista())
  const tabla = page.getByRole('region', { name: 'Las 10 prioridades' })
  await expect(tabla.getByRole('row')).toHaveCount(11) // encabezado + 10
  if (process.env.E2E_EXIGIR_VERSION === '1') {
    // Tras el paso 10, la evaluación es del modelo publicado: sin aviso de desfase.
    await expect(page.getByText(/Esta evaluación es del modelo/)).toHaveCount(0)
  }
  await captura(page, 'e2e-03-inicio')
})

test('mapa: semana de la primera prioridad y panel de su departamento', async ({ page }) => {
  await entrar(page, analista())
  const enlace = page.getByRole('region', { name: 'Las 10 prioridades' }).getByRole('link').first()
  const destino = new URL((await enlace.getAttribute('href')) ?? '', 'http://x')
  const semana = destino.searchParams.get('semana')
  expect(semana).toMatch(/^\d+$/)
  await page.goto(`/mapa?semana=${semana}&metrica=esperados`)
  await expect(page.getByRole('slider', { name: /Semana epidemiológica/ })).toHaveValue(semana!)
  await expect(page.getByRole('group', { name: 'Mapa de riesgo por departamento' }).getByRole('button').first()).toBeVisible()
  await captura(page, 'e2e-04-mapa')
  await page.goto('/')
  await page.getByRole('region', { name: 'Las 10 prioridades' }).getByRole('link').first().click()
  await expect(page).toHaveURL(/\/mapa\?.*depto=/)
  await expect(page.locator('aside.panel-depto')).toBeVisible()
  await expect(page.locator('aside.panel-depto')).not.toContainText('Elige un departamento')
  await captura(page, 'e2e-05-mapa-departamento')
})

async function predecir(page: Page) {
  await page.goto('/prediccion')
  const form = page.getByRole('form', { name: 'Predicción de un caso' })
  await form.getByLabel('Departamento').selectOption({ label: 'Piura' })
  await form.getByLabel('Semana epidemiológica (1–53)').fill('20')
  await form.getByLabel('Edad').fill('35')
  await form.getByRole('radio', { name: 'Femenino' }).check()
  await form.getByRole('button', { name: 'Calcular riesgo' }).click()
  await expect(page.getByText('✓ cuadra')).toBeVisible()
}

test('predicción: riesgo y contribuciones que suman el logit', async ({ page }) => {
  await entrar(page, analista())
  await predecir(page)
  await captura(page, 'e2e-06-prediccion')
})

test('evaluación: cada gráfico dice de qué reporte sale', async ({ page }) => {
  await entrar(page, analista())
  await page.goto('/evaluacion')
  const pies = page.locator('figcaption')
  await expect(pies.first()).toBeVisible()
  const n = await pies.count()
  expect(n).toBeGreaterThanOrEqual(4)
  for (let i = 0; i < n; i++) await expect(pies.nth(i)).toContainText('reports/')
  await captura(page, 'e2e-07-evaluacion')
})

test('responsive: a 375 px no hay desplazamiento horizontal', async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 812 })
  await entrar(page, analista())
  await predecir(page)
  const desborde = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)
  expect(desborde).toBeLessThanOrEqual(0)
  await captura(page, 'e2e-12-responsive-375')
})

test('admin: entrenamiento en vivo, caída de un nodo, sello bit a bit y métricas', async ({ page }) => {
  await entrar(page, admin())
  await expect(menu(page).getByRole('link', { name: 'Clúster' })).toBeVisible()
  await page.goto('/admin')
  await expect(page.getByRole('status').filter({ hasText: 'En vivo' })).toBeVisible()
  const ent = page.getByRole('region', { name: 'Entrenamiento' })
  // Los parámetros del modelo inicial: el modelo publicado no cambia de versión.
  await ent.getByLabel('Épocas').fill('600')
  await ent.getByLabel('Tasa').fill('2')
  await ent.getByLabel('Pausa (ms, demo)').fill('20')
  await ent.getByRole('button', { name: 'Lanzar entrenamiento' }).click()
  const epoca = ent.locator('dt:text-is("Época") + dd')
  await expect.poll(async () => Number((await epoca.textContent())?.replace(/\D/g, '') || 0), { timeout: 60_000 }).toBeGreaterThan(40)
  await captura(page, 'e2e-08-admin-entrenando', false)

  // Si el corte cae a mitad de una época, el coordinador reasigna el shard
  // (línea «reasignacion»); si cae entre dos, el nodo vuelve a registrarse y
  // recibe su shard en la siguiente. En los dos casos la bitácora muestra la
  // caída y el entrenamiento sigue hasta el sello bit a bit.
  await page.getByRole('button', { name: /^Tumbar / }).first().click()
  await expect(page.getByRole('log')).toContainText('salió del clúster', { timeout: 60_000 })
  await capturaDe(page.locator('section').filter({ has: page.getByRole('log') }), 'e2e-09-admin-nodo-caido')

  await expect(page.getByText('bit a bit ✓')).toBeVisible({ timeout: 150_000 })
  await expect(page.locator('[data-metrica="cpu"]').first()).toContainText('%')
  await capturaDe(page.getByRole('region', { name: 'Topología' }), 'e2e-10-admin-metricas')
})

// Swagger UI se descarga de cdn.jsdelivr.net: necesita internet.
test('swagger: contrato OpenAPI con autorización Bearer', async ({ page }) => {
  await page.goto(`${API}/docs`)
  await expect(page.locator('.swagger-ui .opblock').first()).toBeVisible({ timeout: 30_000 })
  await expect(page.locator('.swagger-ui .authorization__btn').first()).toBeVisible()
  await captura(page, 'e2e-11-swagger')
})
