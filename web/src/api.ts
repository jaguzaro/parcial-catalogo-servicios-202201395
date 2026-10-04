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
  servicioCrear: { metodo: 'POST', ruta: '/api/servicios' },
  servicioActualizar: { metodo: 'PUT', ruta: '/api/servicios/{id}' },
  servicioResponsable: { metodo: 'PUT', ruta: '/api/servicios/{id}/responsable' },
  servicioDesactivar: { metodo: 'POST', ruta: '/api/servicios/{id}/desactivar' },
  servicioActivar: { metodo: 'POST', ruta: '/api/servicios/{id}/activar' },

  serviciosN1: { metodo: 'GET', ruta: '/api/servicios-n1' },
  servicioN1Crear: { metodo: 'POST', ruta: '/api/servicios-n1' },
  servicioN1Actualizar: { metodo: 'PUT', ruta: '/api/servicios-n1/{id}' },
  servicioN1Desactivar: { metodo: 'POST', ruta: '/api/servicios-n1/{id}/desactivar' },
  servicioN1Activar: { metodo: 'POST', ruta: '/api/servicios-n1/{id}/activar' },

  catalogo: { metodo: 'GET', ruta: '/api/catalogos/{c}' },

  usuarios: { metodo: 'GET', ruta: '/api/usuarios' },
  usuarioCrear: { metodo: 'POST', ruta: '/api/usuarios' },
  usuarioActualizar: { metodo: 'PUT', ruta: '/api/usuarios/{id}' },
  usuarioContrasena: { metodo: 'PUT', ruta: '/api/usuarios/{id}/contrasena' },
  usuarioDesactivar: { metodo: 'POST', ruta: '/api/usuarios/{id}/desactivar' },
  usuarioActivar: { metodo: 'POST', ruta: '/api/usuarios/{id}/activar' },

  importaciones: { metodo: 'GET', ruta: '/api/importaciones' },
  importacion: { metodo: 'GET', ruta: '/api/importaciones/{id}' },

  empresasListar: { metodo: 'GET', ruta: '/api/empresas' },
  empresasCrear: { metodo: 'POST', ruta: '/api/empresas' },
  empresasActualizar: { metodo: 'PUT', ruta: '/api/empresas/{id}' },
  empresasDesactivar: { metodo: 'POST', ruta: '/api/empresas/{id}/desactivar' },
  empresasActivar: { metodo: 'POST', ruta: '/api/empresas/{id}/activar' },

  areasListar: { metodo: 'GET', ruta: '/api/areas' },
  areasCrear: { metodo: 'POST', ruta: '/api/areas' },
  areasActualizar: { metodo: 'PUT', ruta: '/api/areas/{id}' },
  areasDesactivar: { metodo: 'POST', ruta: '/api/areas/{id}/desactivar' },
  areasActivar: { metodo: 'POST', ruta: '/api/areas/{id}/activar' },

  departamentosListar: { metodo: 'GET', ruta: '/api/departamentos' },
  departamentosCrear: { metodo: 'POST', ruta: '/api/departamentos' },
  departamentosActualizar: { metodo: 'PUT', ruta: '/api/departamentos/{id}' },
  departamentosDesactivar: { metodo: 'POST', ruta: '/api/departamentos/{id}/desactivar' },
  departamentosActivar: { metodo: 'POST', ruta: '/api/departamentos/{id}/activar' },

  seccionesListar: { metodo: 'GET', ruta: '/api/secciones' },
  seccionesCrear: { metodo: 'POST', ruta: '/api/secciones' },
  seccionesActualizar: { metodo: 'PUT', ruta: '/api/secciones/{id}' },
  seccionesDesactivar: { metodo: 'POST', ruta: '/api/secciones/{id}/desactivar' },
  seccionesActivar: { metodo: 'POST', ruta: '/api/secciones/{id}/activar' },

  puestosListar: { metodo: 'GET', ruta: '/api/puestos' },
  puestosCrear: { metodo: 'POST', ruta: '/api/puestos' },
  puestosActualizar: { metodo: 'PUT', ruta: '/api/puestos/{id}' },
  puestosDesactivar: { metodo: 'POST', ruta: '/api/puestos/{id}/desactivar' },
  puestosActivar: { metodo: 'POST', ruta: '/api/puestos/{id}/activar' },
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

export interface ServicioN1 extends Ref {
  activo: boolean
  servicios_n2: number
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

/** Cuerpo de alta y de modificacion de un servicio de nivel 2. */
export interface EntradaServicio {
  servicio_n1_id: number
  codigo: string
  nombre: string
  clase_id: number | null
  criticidad_id: number | null
  tipo_id: number | null
  descripcion: string | null
  metrica: string | null
  minimo: number | null
  maximo: number | null
  requiere_revision?: boolean
}

// ----- estructura organizacional -----

export type RecursoOrg = 'empresas' | 'areas' | 'departamentos' | 'secciones' | 'puestos'

export interface NivelOrg {
  recurso: RecursoOrg
  singular: string
  plural: string
  /** Campo del padre en el cuerpo y en la respuesta: area_id, seccion_id, ... */
  campoPadre?: string
  /** Clave con la que el servidor devuelve el padre resumido: empresa, area, ... */
  clavePadre?: string
  padre?: RecursoOrg
}

export const NIVELES_ORG: NivelOrg[] = [
  { recurso: 'empresas', singular: 'Empresa', plural: 'Empresas' },
  { recurso: 'areas', singular: 'Área', plural: 'Áreas', campoPadre: 'empresa_id', clavePadre: 'empresa', padre: 'empresas' },
  { recurso: 'departamentos', singular: 'Departamento', plural: 'Departamentos', campoPadre: 'area_id', clavePadre: 'area', padre: 'areas' },
  { recurso: 'secciones', singular: 'Sección', plural: 'Secciones', campoPadre: 'departamento_id', clavePadre: 'departamento', padre: 'departamentos' },
  { recurso: 'puestos', singular: 'Puesto', plural: 'Puestos', campoPadre: 'seccion_id', clavePadre: 'seccion', padre: 'secciones' },
]

export interface UnidadOrg {
  id: number
  codigo: string
  nombre: string
  activo: boolean
  /** El padre ya normalizado; null en la empresa. */
  padre: Ref | null
  padre_id: number | null
}

export interface EntradaUnidad {
  codigo: string
  nombre: string
  /** Id del padre; no se envia en la empresa. */
  padre_id?: number
}

export interface FiltrosOrg {
  q?: string
  activo?: string
  /** Id del padre por el que se filtra. */
  padre_id?: string
  pagina?: number
  tamano?: number
}

// ----- usuarios -----

export interface UsuarioFila extends Usuario {
  puesto_id: number
  jerarquia: { puesto: Ref; seccion: Ref; departamento: Ref; area: Ref; empresa: Ref }
}

export interface FiltrosUsuarios {
  q?: string
  rol?: string
  activo?: string
  seccion_id?: string
  pagina?: number
  tamano?: number
}

export interface EntradaUsuario {
  nombre: string
  usuario: string
  correo: string | null
  rol: string
  puesto_id: number
}

export interface Advertencia {
  codigo: string
  mensaje: string
  servicios: Dependencia[]
}

// ----- importaciones -----

export interface Corrida {
  id: number
  origen: 'cli' | 'web'
  estado: 'en_curso' | 'completada' | 'fallida'
  archivo_ruta: string
  archivo_sha256: string
  hoja: string
  creados: number
  actualizados: number
  omitidos: number
  observados: number
  detalle: Record<string, unknown>
  mensaje_error: string | null
  iniciada_en: string
  finalizada_en: string | null
  incidencias?: IncidenciaCorrida[]
}

export interface IncidenciaCorrida extends Incidencia {
  codigo: string | null
  valor_original: unknown
  valor_aplicado: unknown
}

// ---------- errores ----------

export interface Dependencia {
  entidad: string
  id: number
  codigo: string
  nombre?: string
}

/** Error del servidor con su formato unico: {"error": {"codigo", "mensaje", ...}}. */
export class ErrorApi extends Error {
  constructor(
    readonly estado: number,
    readonly codigo: string,
    mensaje: string,
    readonly campos: string[] = [],
    readonly dependencias: Dependencia[] = [],
    readonly totalDependencias = 0,
  ) {
    super(mensaje)
  }
}

// ---------- peticiones ----------

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

interface CuerpoError {
  error?: {
    codigo?: string
    mensaje?: string
    campos?: string[]
    dependencias?: Dependencia[]
    total_dependencias?: number
  }
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
    const e = (cuerpo as CuerpoError | undefined)?.error
    throw new ErrorApi(
      resp.status,
      e?.codigo ?? 'DESCONOCIDO',
      e?.mensaje ?? `Error ${resp.status} del servidor.`,
      e?.campos,
      e?.dependencias,
      e?.total_dependencias,
    )
  }
  return cuerpo as T
}

// ---------- estructura organizacional ----------

const ORG = {
  empresas: {
    listar: RUTAS.empresasListar, crear: RUTAS.empresasCrear, actualizar: RUTAS.empresasActualizar,
    desactivar: RUTAS.empresasDesactivar, activar: RUTAS.empresasActivar,
  },
  areas: {
    listar: RUTAS.areasListar, crear: RUTAS.areasCrear, actualizar: RUTAS.areasActualizar,
    desactivar: RUTAS.areasDesactivar, activar: RUTAS.areasActivar,
  },
  departamentos: {
    listar: RUTAS.departamentosListar, crear: RUTAS.departamentosCrear, actualizar: RUTAS.departamentosActualizar,
    desactivar: RUTAS.departamentosDesactivar, activar: RUTAS.departamentosActivar,
  },
  secciones: {
    listar: RUTAS.seccionesListar, crear: RUTAS.seccionesCrear, actualizar: RUTAS.seccionesActualizar,
    desactivar: RUTAS.seccionesDesactivar, activar: RUTAS.seccionesActivar,
  },
  puestos: {
    listar: RUTAS.puestosListar, crear: RUTAS.puestosCrear, actualizar: RUTAS.puestosActualizar,
    desactivar: RUTAS.puestosDesactivar, activar: RUTAS.puestosActivar,
  },
} as const

export function nivelOrg(recurso: RecursoOrg): NivelOrg {
  return NIVELES_ORG.find((n) => n.recurso === recurso)!
}

/** El servidor devuelve el padre con el nombre de su tabla; aqui queda siempre en `padre`. */
function normalizar(nivel: NivelOrg, bruto: Record<string, unknown>): UnidadOrg {
  return {
    id: bruto.id as number,
    codigo: bruto.codigo as string,
    nombre: bruto.nombre as string,
    activo: bruto.activo as boolean,
    padre: nivel.clavePadre ? ((bruto[nivel.clavePadre] as Ref | null | undefined) ?? null) : null,
    padre_id: nivel.campoPadre ? ((bruto[nivel.campoPadre] as number | undefined) ?? null) : null,
  }
}

function cuerpoUnidad(nivel: NivelOrg, e: EntradaUnidad): Record<string, unknown> {
  const c: Record<string, unknown> = { codigo: e.codigo, nombre: e.nombre }
  if (nivel.campoPadre) c[nivel.campoPadre] = e.padre_id ?? 0
  return c
}

const org = {
  listar: async (recurso: RecursoOrg, f: FiltrosOrg): Promise<Listado<UnidadOrg>> => {
    const nivel = nivelOrg(recurso)
    const { padre_id, ...resto } = f
    const consulta: Record<string, unknown> = { ...resto }
    if (nivel.campoPadre && padre_id) consulta[nivel.campoPadre] = padre_id
    const r = await pedir<Listado<Record<string, unknown>>>(ORG[recurso].listar, { consulta })
    return { ...r, items: r.items.map((b) => normalizar(nivel, b)) }
  },
  crear: async (recurso: RecursoOrg, e: EntradaUnidad) => {
    const nivel = nivelOrg(recurso)
    return normalizar(nivel, await pedir(ORG[recurso].crear, { cuerpo: cuerpoUnidad(nivel, e) }))
  },
  actualizar: async (recurso: RecursoOrg, id: number, e: EntradaUnidad) => {
    const nivel = nivelOrg(recurso)
    return normalizar(nivel, await pedir(ORG[recurso].actualizar, { params: { id }, cuerpo: cuerpoUnidad(nivel, e) }))
  },
  desactivar: async (recurso: RecursoOrg, id: number) =>
    normalizar(nivelOrg(recurso), await pedir(ORG[recurso].desactivar, { params: { id } })),
  activar: async (recurso: RecursoOrg, id: number) =>
    normalizar(nivelOrg(recurso), await pedir(ORG[recurso].activar, { params: { id } })),
}

// ---------- el resto ----------

export const api = {
  login: (login: string, contrasena: string) =>
    pedir<{ usuario: Usuario }>(RUTAS.login, { cuerpo: { login, contrasena } }).then((r) => r.usuario),
  logout: () => pedir<void>(RUTAS.logout),
  yo: () => pedir<{ usuario: Usuario }>(RUTAS.yo).then((r) => r.usuario),

  servicios: (filtros: FiltrosServicios) => pedir<Listado<Servicio>>(RUTAS.servicios, { consulta: filtros }),
  servicio: (id: number) => pedir<Servicio>(RUTAS.servicio, { params: { id } }),
  servicioCrear: (e: EntradaServicio) => pedir<Servicio>(RUTAS.servicioCrear, { cuerpo: e }),
  servicioActualizar: (id: number, e: EntradaServicio) =>
    pedir<Servicio>(RUTAS.servicioActualizar, { params: { id }, cuerpo: e }),
  servicioResponsable: (id: number, seccionId: number | null, usuarioId: number | null) =>
    pedir<Servicio>(RUTAS.servicioResponsable, {
      params: { id },
      cuerpo: { seccion_id: seccionId, usuario_id: usuarioId },
    }),
  servicioDesactivar: (id: number) => pedir<Servicio>(RUTAS.servicioDesactivar, { params: { id } }),
  servicioActivar: (id: number) => pedir<Servicio>(RUTAS.servicioActivar, { params: { id } }),

  serviciosN1: (consulta: { q?: string; activo?: string; pagina?: number; tamano?: number } = { tamano: 100 }) =>
    pedir<Listado<ServicioN1>>(RUTAS.serviciosN1, { consulta }),
  servicioN1Crear: (e: { codigo: string; nombre: string }) => pedir<ServicioN1>(RUTAS.servicioN1Crear, { cuerpo: e }),
  servicioN1Actualizar: (id: number, e: { codigo: string; nombre: string }) =>
    pedir<ServicioN1>(RUTAS.servicioN1Actualizar, { params: { id }, cuerpo: e }),
  servicioN1Desactivar: (id: number) => pedir<ServicioN1>(RUTAS.servicioN1Desactivar, { params: { id } }),
  servicioN1Activar: (id: number) => pedir<ServicioN1>(RUTAS.servicioN1Activar, { params: { id } }),

  catalogo: (c: NombreCatalogo) =>
    pedir<{ items: OpcionCatalogo[]; total: number }>(RUTAS.catalogo, { params: { c } }).then((r) => r.items),

  org,

  usuarios: (f: FiltrosUsuarios) => pedir<Listado<UsuarioFila>>(RUTAS.usuarios, { consulta: f }),
  usuarioCrear: (e: EntradaUsuario & { contrasena: string }) => pedir<UsuarioFila>(RUTAS.usuarioCrear, { cuerpo: e }),
  usuarioActualizar: (id: number, e: EntradaUsuario) =>
    pedir<UsuarioFila>(RUTAS.usuarioActualizar, { params: { id }, cuerpo: e }),
  usuarioContrasena: (id: number, contrasena: string) =>
    pedir<void>(RUTAS.usuarioContrasena, { params: { id }, cuerpo: { contrasena } }),
  usuarioDesactivar: (id: number) =>
    pedir<{ usuario: UsuarioFila; advertencias: Advertencia[] }>(RUTAS.usuarioDesactivar, { params: { id } }),
  usuarioActivar: (id: number) => pedir<UsuarioFila>(RUTAS.usuarioActivar, { params: { id } }),

  importaciones: (consulta: { pagina?: number; tamano?: number } = {}) =>
    pedir<Listado<Corrida>>(RUTAS.importaciones, { consulta }),
  importacion: (id: number) => pedir<Corrida>(RUTAS.importacion, { params: { id } }),
}
