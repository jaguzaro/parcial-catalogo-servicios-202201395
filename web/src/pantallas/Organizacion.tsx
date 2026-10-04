import { useState, type FormEvent } from 'react'
import { Link, Navigate, useParams, useSearchParams } from 'react-router'
import { NIVELES_ORG, api, nivelOrg, type NivelOrg, type RecursoOrg, type UnidadOrg } from '../api'
import { useConsulta, Valor } from './comunes'
import { CampoForm, ErrorForm, ErrorServidor, Paginacion, SoloAdministrador, useAccion } from './formularios'

const TAMANO = 20

function FormularioUnidad({
  nivel,
  unidad,
  alTerminar,
  alCancelar,
}: {
  nivel: NivelOrg
  unidad: UnidadOrg | null
  alTerminar: () => void
  alCancelar: () => void
}) {
  const [codigo, setCodigo] = useState(unidad?.codigo ?? '')
  const [nombre, setNombre] = useState(unidad?.nombre ?? '')
  const [padreId, setPadreId] = useState(unidad?.padre_id ? String(unidad.padre_id) : '')
  const accion = useAccion()

  const padre = nivel.padre ? nivelOrg(nivel.padre) : null
  const padres = useConsulta(
    () => (padre ? api.org.listar(padre.recurso, { activo: 'true', tamano: 100 }) : Promise.resolve(null)),
    `padres-${nivel.recurso}`,
  )
  // Si el padre actual esta inactivo no viene en la lista; se agrega para no perderlo de vista.
  const opciones = padres.datos?.items ?? []
  const falta = unidad?.padre && !opciones.some((p) => p.id === unidad.padre!.id)

  async function enviar(e: FormEvent) {
    e.preventDefault()
    const cuerpo = { codigo, nombre, padre_id: padreId ? Number(padreId) : undefined }
    const r = await accion.ejecutar(() =>
      unidad ? api.org.actualizar(nivel.recurso, unidad.id, cuerpo) : api.org.crear(nivel.recurso, cuerpo),
    )
    if (r) alTerminar()
  }

  const campos = ['codigo', 'nombre', ...(nivel.campoPadre ? [nivel.campoPadre] : [])]

  return (
    <form className="panel" onSubmit={(e) => void enviar(e)}>
      <h2>{unidad ? `Editar ${nivel.singular.toLowerCase()} ${unidad.codigo}` : `Nueva ${nivel.singular.toLowerCase()}`}</h2>
      <div className="rejilla-form">
        <CampoForm etiqueta="Código" campo="codigo" error={accion.error}>
          <input value={codigo} onChange={(e) => setCodigo(e.target.value)} />
        </CampoForm>
        <CampoForm etiqueta="Nombre" campo="nombre" error={accion.error}>
          <input value={nombre} onChange={(e) => setNombre(e.target.value)} />
        </CampoForm>
        {padre && nivel.campoPadre && (
          <CampoForm etiqueta={padre.singular} campo={nivel.campoPadre} error={accion.error}>
            <select value={padreId} onChange={(e) => setPadreId(e.target.value)}>
              <option value="">Elija {padre.singular.toLowerCase()}</option>
              {falta && unidad?.padre && (
                <option value={unidad.padre.id}>{unidad.padre.codigo} · {unidad.padre.nombre} (inactivo)</option>
              )}
              {opciones.map((p) => (
                <option key={p.id} value={p.id}>{p.codigo} · {p.nombre}</option>
              ))}
            </select>
          </CampoForm>
        )}
      </div>
      {padres.error && <p className="error" role="alert">{padres.error}</p>}
      <ErrorForm error={accion.error} campos={campos} />
      <div className="acciones">
        <button type="submit" disabled={accion.ocupado}>{unidad ? 'Guardar cambios' : 'Crear'}</button>
        <button type="button" className="secundario" onClick={alCancelar}>Cancelar</button>
      </div>
    </form>
  )
}

function ListaNivel({ nivel }: { nivel: NivelOrg }) {
  const [params, setParams] = useSearchParams()
  const [modo, setModo] = useState<'nuevo' | UnidadOrg | null>(null)
  const [version, setVersion] = useState(0)
  const [aviso, setAviso] = useState<string | null>(null)
  const accion = useAccion()

  const pagina = Math.max(1, Number(params.get('pagina')) || 1)
  const filtros = {
    q: params.get('q') ?? '',
    activo: params.get('activo') ?? '',
    padre_id: params.get('padre_id') ?? '',
  }
  const [texto, setTexto] = useState(filtros.q)
  const padre = nivel.padre ? nivelOrg(nivel.padre) : null
  const padres = useConsulta(
    () => (padre ? api.org.listar(padre.recurso, { tamano: 100 }) : Promise.resolve(null)),
    `filtro-padres-${nivel.recurso}`,
  )
  const lista = useConsulta(
    () => api.org.listar(nivel.recurso, { ...filtros, pagina, tamano: TAMANO }),
    `${nivel.recurso}|${params.toString()}|${version}`,
  )

  function cambiar(clave: string, valor: string) {
    const sig = new URLSearchParams(params)
    if (valor) sig.set(clave, valor)
    else sig.delete(clave)
    sig.delete('pagina')
    setParams(sig)
  }

  function irAPagina(p: number) {
    const sig = new URLSearchParams(params)
    if (p > 1) sig.set('pagina', String(p))
    else sig.delete('pagina')
    setParams(sig)
  }

  function terminar(mensaje: string) {
    setModo(null)
    setAviso(mensaje)
    setVersion((v) => v + 1)
  }

  async function cambiarEstado(u: UnidadOrg, desactivar: boolean) {
    setAviso(null)
    const r = await accion.ejecutar(() =>
      desactivar ? api.org.desactivar(nivel.recurso, u.id) : api.org.activar(nivel.recurso, u.id),
    )
    if (r) setVersion((v) => v + 1)
  }

  return (
    <>
      <form
        className="filtros"
        role="search"
        onSubmit={(e) => {
          e.preventDefault()
          cambiar('q', texto.trim())
        }}
      >
        <label className="busqueda">
          Buscar por código o nombre
          <input value={texto} onChange={(e) => setTexto(e.target.value)} />
        </label>
        <button type="submit">Buscar</button>
        {padre && (
          <label>
            {padre.singular}
            <select value={filtros.padre_id} onChange={(e) => cambiar('padre_id', e.target.value)}>
              <option value="">Todas</option>
              {padres.datos?.items.map((p) => (
                <option key={p.id} value={p.id}>{p.codigo} · {p.nombre}</option>
              ))}
            </select>
          </label>
        )}
        <label>
          Estado
          <select value={filtros.activo} onChange={(e) => cambiar('activo', e.target.value)}>
            <option value="">Todos</option>
            <option value="true">Activos</option>
            <option value="false">Inactivos</option>
          </select>
        </label>
        <SoloAdministrador>
          <button type="button" onClick={() => setModo('nuevo')}>Nueva {nivel.singular.toLowerCase()}</button>
        </SoloAdministrador>
      </form>

      {modo === 'nuevo' && (
        <FormularioUnidad nivel={nivel} unidad={null} alCancelar={() => setModo(null)} alTerminar={() => terminar('Registro creado.')} />
      )}
      {modo !== null && modo !== 'nuevo' && (
        <FormularioUnidad
          key={modo.id}
          nivel={nivel}
          unidad={modo}
          alCancelar={() => setModo(null)}
          alTerminar={() => terminar('Cambios guardados.')}
        />
      )}

      {aviso && <p className="exito" role="status">{aviso}</p>}
      {accion.error && <ErrorServidor error={accion.error} />}
      {lista.error && <p className="error" role="alert">{lista.error}</p>}
      {lista.cargando && !lista.datos && <p className="aviso" role="status">Cargando…</p>}
      {lista.datos && (
        <>
          <p className="total" role="status">
            {lista.datos.total === 0 ? 'Sin resultados' : `${lista.datos.total} registros`}
          </p>
          <div className="tabla-contenedor">
            <table>
              <thead>
                <tr>
                  <th>Código</th>
                  <th>Nombre</th>
                  {padre && <th>Depende de ({padre.singular.toLowerCase()})</th>}
                  <th>Estado</th>
                  <SoloAdministrador><th>Acciones</th></SoloAdministrador>
                </tr>
              </thead>
              <tbody>
                {lista.datos.items.map((u) => (
                  <tr key={u.id}>
                    <td>{u.codigo}</td>
                    <td>{u.nombre}</td>
                    {padre && <td>{u.padre ? `${u.padre.codigo} · ${u.padre.nombre}` : <Valor v={null} />}</td>}
                    <td>
                      <span className={`estado ${u.activo ? 'estado-s' : 'estado-n'}`}>{u.activo ? 'Activo' : 'Inactivo'}</span>
                    </td>
                    <SoloAdministrador>
                      <td className="acciones-fila">
                        <button type="button" className="secundario" onClick={() => setModo(u)}>Editar</button>
                        <button
                          type="button"
                          className="secundario"
                          disabled={accion.ocupado}
                          onClick={() => void cambiarEstado(u, u.activo)}
                        >
                          {u.activo ? 'Dar de baja' : 'Dar de alta'}
                        </button>
                      </td>
                    </SoloAdministrador>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <Paginacion pagina={lista.datos.pagina} total={lista.datos.total} tamano={TAMANO} onIr={irAPagina} />
        </>
      )}
    </>
  )
}

export default function Organizacion() {
  const { recurso } = useParams()
  const nivel = NIVELES_ORG.find((n) => n.recurso === (recurso as RecursoOrg))
  if (!nivel) return <Navigate to="/organizacion/empresas" replace />

  return (
    <section>
      <h1>Estructura organizacional</h1>
      <nav className="pestanas" aria-label="Niveles de la estructura">
        {NIVELES_ORG.map((n) => (
          <Link key={n.recurso} to={`/organizacion/${n.recurso}`} aria-current={n.recurso === nivel.recurso ? 'page' : undefined}>
            {n.plural}
          </Link>
        ))}
      </nav>
      {/* La clave reinicia filtros, formulario y mensajes al cambiar de nivel. */}
      <ListaNivel key={nivel.recurso} nivel={nivel} />
    </section>
  )
}
