// Unico modulo que habla con la API. Toda peticion de la interfaz pasa por aqui.
//
// Las rutas se declaran una sola vez en RUTAS, con su metodo y con {parametros} tal como
// las registra el servidor. TestWeb_RutasUsadasExisten (Go) lee este bloque y comprueba
// que cada una exista en RutasAPI(): no cambie el formato de las entradas.

export const RUTAS = {
  login: { metodo: 'POST', ruta: '/api/auth/login' },
  logout: { metodo: 'POST', ruta: '/api/auth/logout' },
  yo: { metodo: 'GET', ruta: '/api/auth/me' },
  servicios: { metodo: 'GET', ruta: '/api/servicios' },
  servicio: { metodo: 'GET', ruta: '/api/servicios/{id}' },
  serviciosN1: { metodo: 'GET', ruta: '/api/servicios-n1' },
  catalogo: { metodo: 'GET', ruta: '/api/catalogos/{c}' },
} as const

type Entrada = (typeof RUTAS)[keyof typeof RUTAS]

// ---------- tipos de la API ----------

export interface Usuario {
  id: number
  nombre: string
  usuario: string
  correo: string | null
  rol: 'administrador' | 'consulta'
  activo: boolean
}

export type EstadoServicio = 'S' | 'N' | 'DESCONOCIDO'

export interface Ref {
  id: number
  codigo: string
  nombre: string
}

export interface Opcion {
  id: number
  nombre: string
  activo: boolean
}

export interface Incidencia {
  id: number
  tipo: string
  regla: string | null
  hoja: string
  fila: number | null
  celdas: string | null
  mensaje: string
}

export interface Servicio {
  id: number
  codigo: string
  nombre: string
  activo: EstadoServicio
  servicio_n1: Ref
  clase: Opcion | null
  criticidad: Opcion | null
  tipo: Opcion | null
  descripcion: string | null
  metrica: string | null
  // null es "sin dato", nunca cero.
  minimo: number | null
  maximo: number | null
  requiere_revision: boolean
  seccion_responsable: (Ref & { activo: boolean; departamento: Ref; area: Ref; empresa: Ref }) | null
  usuario_responsable: { id: number; nombre: string; usuario: string; activo: boolean } | null
  incidencias?: Incidencia[]
}

export interface Listado<T> {
  items: T[]
  total: number
  pagina: number
  tamano: number
}

export interface OpcionCatalogo extends Opcion {
  orden: number
}

export type NombreCatalogo = 'clases' | 'criticidades' | 'tipos'

export interface FiltrosServicios {
  q?: string
  n1_id?: string
  activo?: string
  clase_id?: string
  criticidad_id?: string
  tipo_id?: string
  pagina?: number
  tamano?: number
}

// ---------- peticiones ----------

/** Error del servidor con su formato unico: {"error": {"codigo", "mensaje"}}. */
export class ErrorApi extends Error {
  constructor(
    readonly estado: number,
    readonly codigo: string,
    mensaje: string,
  ) {
    super(mensaje)
  }
}

function construir(ruta: string, params: Record<string, string | number> = {}, consulta?: object): string {
  let url = ruta.replace(/\{(\w+)\}/g, (_, nombre: string) => encodeURIComponent(String(params[nombre])))
  const qs = new URLSearchParams()
  for (const [k, v] of Object.entries(consulta ?? {})) {
    if (v !== undefined && v !== '') qs.set(k, String(v))
  }
  const texto = qs.toString()
  if (texto) url += '?' + texto
  return url
}

async function pedir<T>(
  entrada: Entrada,
  opciones: { params?: Record<string, string | number>; consulta?: object; cuerpo?: unknown } = {},
): Promise<T> {
  const init: RequestInit = { method: entrada.metodo, credentials: 'same-origin' }
  if (opciones.cuerpo !== undefined) {
    init.headers = { 'Content-Type': 'application/json' }
    init.body = JSON.stringify(opciones.cuerpo)
  } else if (entrada.metodo !== 'GET') {
    // El servidor exige application/json en todo POST y PUT, aunque no lleve datos.
    init.headers = { 'Content-Type': 'application/json' }
    init.body = '{}'
  }
  const resp = await fetch(construir(entrada.ruta, opciones.params, opciones.consulta), init)
  if (resp.status === 204) return undefined as T
  let cuerpo: unknown
  try {
    cuerpo = await resp.json()
  } catch {
    cuerpo = undefined
  }
  if (!resp.ok) {
    const e = (cuerpo as { error?: { codigo?: string; mensaje?: string } } | undefined)?.error
    throw new ErrorApi(resp.status, e?.codigo ?? 'DESCONOCIDO', e?.mensaje ?? `Error ${resp.status} del servidor.`)
  }
  return cuerpo as T
}

export const api = {
  login: (login: string, contrasena: string) =>
    pedir<{ usuario: Usuario }>(RUTAS.login, { cuerpo: { login, contrasena } }).then((r) => r.usuario),
  logout: () => pedir<void>(RUTAS.logout),
  yo: () => pedir<{ usuario: Usuario }>(RUTAS.yo).then((r) => r.usuario),
  servicios: (filtros: FiltrosServicios) => pedir<Listado<Servicio>>(RUTAS.servicios, { consulta: filtros }),
  servicio: (id: number) => pedir<Servicio>(RUTAS.servicio, { params: { id } }),
  serviciosN1: () => pedir<Listado<Ref>>(RUTAS.serviciosN1, { consulta: { tamano: 100 } }),
  catalogo: (c: NombreCatalogo) =>
    pedir<{ items: OpcionCatalogo[]; total: number }>(RUTAS.catalogo, { params: { c } }).then((r) => r.items),
}
