// La URL de la API se decide al desplegar, no al compilar: cmd/web entrega
// /config.json a partir de la variable API_URL (en desarrollo, public/config.json).
export interface Config {
  apiUrl: string
}

export async function cargarConfig(): Promise<Config> {
  const r = await fetch('/config.json', { cache: 'no-store' })
  if (!r.ok) throw new Error(`/config.json respondió ${r.status}`)
  const c: unknown = await r.json()
  if (!c || typeof c !== 'object' || typeof (c as { api_url?: unknown }).api_url !== 'string') {
    throw new Error('/config.json sin api_url')
  }
  return { apiUrl: (c as { api_url: string }).api_url }
}
