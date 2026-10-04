import { expect, test, type Page } from '@playwright/test'

// Las credenciales llegan por variables de entorno (las mismas cuentas de evaluacion
// que siembra `make seed-demo`); aqui no hay ninguna escrita.
const usuario = process.env.DEMO_CONSULTA_USUARIO ?? ''
const contrasena = process.env.DEMO_CONSULTA_CONTRASENA ?? ''

async function entrar(page: Page) {
  await page.goto('/login')
  await page.getByLabel('Usuario o correo').fill(usuario)
  await page.getByLabel('Contraseña').fill(contrasena)
  await page.getByRole('button', { name: 'Entrar' }).click()
}

// "46 servicios · página 1 de 3" -> 46; "Sin resultados" -> 0
async function totalMostrado(page: Page): Promise<number> {
  const texto = (await page.locator('p.total').innerText()).trim()
  if (texto.startsWith('Sin resultados')) return 0
  const m = texto.match(/^(\d+) servicios?/)
  expect(m, `texto de total inesperado: "${texto}"`).not.toBeNull()
  return Number(m![1])
}

test.beforeAll(() => {
  expect(usuario, 'falta DEMO_CONSULTA_USUARIO en el entorno').not.toBe('')
  expect(contrasena, 'falta DEMO_CONSULTA_CONTRASENA en el entorno').not.toBe('')
})

test('inicio de sesion: entrar con una cuenta sembrada y llegar al catalogo', async ({ page }) => {
  await entrar(page)
  await expect(page).toHaveURL(/\/$/)
  await expect(page.getByRole('heading', { name: 'Servicios de nivel 2' })).toBeVisible()
  await expect(page.locator('table tbody tr').first()).toBeVisible()
})

test('filtro: al filtrar por nivel 1 cambian la tabla y el total', async ({ page }) => {
  await entrar(page)
  await expect(page.locator('table tbody tr').first()).toBeVisible()
  const totalInicial = await totalMostrado(page)
  expect(totalInicial).toBeGreaterThan(0)

  const selector = page.getByLabel('Nivel 1')
  const codigo = 'SE.12'
  const opcion = selector.locator('option', { hasText: codigo })
  await expect(opcion).toHaveCount(1)
  await selector.selectOption({ label: (await opcion.innerText()).trim() })

  // El filtro viaja en la URL; esperar a que el total deje de ser el inicial.
  await expect(page).toHaveURL(/n1_id=\d+/)
  await expect(page.locator('p.total')).not.toContainText(`${totalInicial} servicios`)

  const total = await totalMostrado(page)
  expect(total).toBeGreaterThan(0)
  expect(total).toBeLessThan(totalInicial)

  // Toda fila visible pertenece al nivel 1 elegido, y el total coincide con las filas
  // (cabe en una sola pagina de 20).
  const filas = page.locator('table tbody tr')
  await expect(filas).toHaveCount(total)
  for (const celda of await filas.locator('td:nth-child(3)').allInnerTexts()) {
    expect(celda.startsWith(codigo)).toBe(true)
  }

  // El total mostrado coincide con lo que responde la API con el mismo filtro.
  const n1Id = new URL(page.url()).searchParams.get('n1_id')
  const r = await page.request.get(`/api/servicios?n1_id=${n1Id}&tamano=1`)
  expect(r.ok()).toBe(true)
  expect((await r.json()).total).toBe(total)
})
