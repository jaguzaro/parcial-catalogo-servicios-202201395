import { useState, type FormEvent } from 'react'
import { useSearchParams } from 'react-router'
import { api, type ServicioN1 } from '../api'
import { useConsulta } from './comunes'
import { CampoForm, ErrorForm, ErrorServidor, Paginacion, SoloAdministrador, useAccion } from './formularios'

const TAMANO = 20

function FormularioN1({
  servicio,
  alTerminar,
  alCancelar,
}: {
  servicio: ServicioN1 | null
  alTerminar: () => void
  alCancelar: () => void
}) {
  const [codigo, setCodigo] = useState(servicio?.codigo ?? '')
  const [nombre, setNombre] = useState(servicio?.nombre ?? '')
  const accion = useAccion()

  async function enviar(e: FormEvent) {
    e.preventDefault()
    const r = await accion.ejecutar(() =>
      servicio ? api.servicioN1Actualizar(servicio.id, { codigo, nombre }) : api.servicioN1Crear({ codigo, nombre }),
    )
    if (r) alTerminar()
  }

  return (
    <form className="panel" onSubmit={(e) => void enviar(e)}>
      <h2>{servicio ? `Editar ${servicio.codigo}` : 'Nuevo servicio de nivel 1'}</h2>
      <div className="rejilla-form">
        <CampoForm etiqueta="Código" campo="codigo" error={accion.error}>
          <input value={codigo} onChange={(e) => setCodigo(e.target.value)} />
        </CampoForm>
        <CampoForm etiqueta="Nombre" campo="nombre" error={accion.error}>
          <input value={nombre} onChange={(e) => setNombre(e.target.value)} />
        </CampoForm>
      </div>
      <ErrorForm error={accion.error} campos={['codigo', 'nombre']} />
      <div className="acciones">
        <button type="submit" disabled={accion.ocupado}>{servicio ? 'Guardar cambios' : 'Crear'}</button>
        <button type="button" className="secundario" onClick={alCancelar}>Cancelar</button>
      </div>
    </form>
  )
}

export default function ServiciosN1() {
  const [params, setParams] = useSearchParams()
  const [modo, setModo] = useState<'nuevo' | ServicioN1 | null>(null)
  const [version, setVersion] = useState(0)
  const [aviso, setAviso] = useState<string | null>(null)
  const accion = useAccion()

  const pagina = Math.max(1, Number(params.get('pagina')) || 1)
  const q = params.get('q') ?? ''
  const activo = params.get('activo') ?? ''
  const [texto, setTexto] = useState(q)
  const lista = useConsulta(
    () => api.serviciosN1({ q, activo, pagina, tamano: TAMANO }),
    `${params.toString()}|${version}`,
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

  async function cambiarEstado(s: ServicioN1) {
    setAviso(null)
    const r = await accion.ejecutar(() => (s.activo ? api.servicioN1Desactivar(s.id) : api.servicioN1Activar(s.id)))
    if (r) setVersion((v) => v + 1)
  }

  return (
    <section>
      <h1>Servicios de nivel 1</h1>

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
        <label>
          Estado
          <select value={activo} onChange={(e) => cambiar('activo', e.target.value)}>
            <option value="">Todos</option>
            <option value="true">Activos</option>
            <option value="false">Inactivos</option>
          </select>
        </label>
        <SoloAdministrador>
          <button type="button" onClick={() => setModo('nuevo')}>Nuevo servicio de nivel 1</button>
        </SoloAdministrador>
      </form>

      {modo === 'nuevo' && (
        <FormularioN1 servicio={null} alCancelar={() => setModo(null)} alTerminar={() => terminar('Servicio creado.')} />
      )}
      {modo !== null && modo !== 'nuevo' && (
        <FormularioN1
          key={modo.id}
          servicio={modo}
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
            {lista.datos.total === 0 ? 'Sin resultados' : `${lista.datos.total} servicios de nivel 1`}
          </p>
          <div className="tabla-contenedor">
            <table>
              <thead>
                <tr>
                  <th>Código</th>
                  <th>Nombre</th>
                  <th>Servicios de nivel 2</th>
                  <th>Estado</th>
                  <SoloAdministrador><th>Acciones</th></SoloAdministrador>
                </tr>
              </thead>
              <tbody>
                {lista.datos.items.map((s) => (
                  <tr key={s.id}>
                    <td>{s.codigo}</td>
                    <td>{s.nombre}</td>
                    <td>{s.servicios_n2}</td>
                    <td>
                      <span className={`estado ${s.activo ? 'estado-s' : 'estado-n'}`}>{s.activo ? 'Activo' : 'Inactivo'}</span>
                    </td>
                    <SoloAdministrador>
                      <td className="acciones-fila">
                        <button type="button" className="secundario" onClick={() => setModo(s)}>Editar</button>
                        <button type="button" className="secundario" disabled={accion.ocupado} onClick={() => void cambiarEstado(s)}>
                          {s.activo ? 'Dar de baja' : 'Dar de alta'}
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
    </section>
  )
}
