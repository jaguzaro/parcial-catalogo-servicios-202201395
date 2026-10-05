import { useState, type FormEvent } from 'react'
import { api, type NombreCatalogo, type OpcionCatalogo } from '../api'
import { useConsulta, Valor } from './comunes'
import { CampoForm, ErrorForm, ErrorServidor, SoloAdministrador, useAccion } from './formularios'

const CATALOGOS: { clave: NombreCatalogo; titulo: string; singular: string }[] = [
  { clave: 'clases', titulo: 'Clases de servicio', singular: 'clase' },
  { clave: 'criticidades', titulo: 'Criticidades', singular: 'criticidad' },
  { clave: 'tipos', titulo: 'Tipos de servicio', singular: 'tipo' },
]

function FormularioOpcion({
  catalogo,
  singular,
  opcion,
  alTerminar,
  alCancelar,
}: {
  catalogo: NombreCatalogo
  singular: string
  opcion: OpcionCatalogo | null
  alTerminar: () => void
  alCancelar: () => void
}) {
  const [nombre, setNombre] = useState(opcion?.nombre ?? '')
  const [orden, setOrden] = useState(opcion ? String(opcion.orden) : '')
  const accion = useAccion()

  async function enviar(e: FormEvent) {
    e.preventDefault()
    const entrada = { nombre, orden: Number(orden) }
    const r = await accion.ejecutar(() =>
      opcion ? api.catalogoActualizar(catalogo, opcion.id, entrada) : api.catalogoCrear(catalogo, entrada),
    )
    if (r) alTerminar()
  }

  return (
    <form className="panel" onSubmit={(e) => void enviar(e)}>
      <h2>{opcion ? `Editar ${singular}: ${opcion.nombre}` : `Nueva ${singular}`}</h2>
      {opcion && (
        <p className="total">
          Valor de origen (no se edita): {opcion.valor_origen === null ? <Valor v={null} /> : <code>{opcion.valor_origen}</code>}
        </p>
      )}
      <div className="rejilla-form">
        <CampoForm etiqueta="Nombre" campo="nombre" error={accion.error}>
          <input value={nombre} onChange={(e) => setNombre(e.target.value)} />
        </CampoForm>
        <CampoForm etiqueta="Orden" campo="orden" error={accion.error}>
          <input type="number" min={1} value={orden} onChange={(e) => setOrden(e.target.value)} />
        </CampoForm>
      </div>
      <ErrorForm error={accion.error} campos={['nombre', 'orden']} />
      <div className="acciones">
        <button type="submit" disabled={accion.ocupado}>{opcion ? 'Guardar cambios' : 'Crear'}</button>
        <button type="button" className="secundario" onClick={alCancelar}>Cancelar</button>
      </div>
    </form>
  )
}

function Lista({ clave, titulo, singular }: { clave: NombreCatalogo; titulo: string; singular: string }) {
  const [version, setVersion] = useState(0)
  const [modo, setModo] = useState<'nuevo' | OpcionCatalogo | null>(null)
  const [aviso, setAviso] = useState<string | null>(null)
  const accion = useAccion()
  const lista = useConsulta(() => api.catalogo(clave), `${clave}|${version}`)

  function terminar(mensaje: string) {
    setModo(null)
    setAviso(mensaje)
    setVersion((v) => v + 1)
  }

  async function cambiarEstado(o: OpcionCatalogo) {
    setAviso(null)
    const r = await accion.ejecutar(() => (o.activo ? api.catalogoDesactivar(clave, o.id) : api.catalogoActivar(clave, o.id)))
    if (r) setVersion((v) => v + 1)
  }

  return (
    <section aria-label={titulo}>
      <h2>{titulo}</h2>
      <SoloAdministrador>
        <p><button type="button" onClick={() => setModo('nuevo')}>Nueva {singular}</button></p>
      </SoloAdministrador>

      {modo !== null && (
        <FormularioOpcion
          key={modo === 'nuevo' ? 'nuevo' : modo.id}
          catalogo={clave}
          singular={singular}
          opcion={modo === 'nuevo' ? null : modo}
          alCancelar={() => setModo(null)}
          alTerminar={() => terminar(modo === 'nuevo' ? 'Opción creada.' : 'Cambios guardados.')}
        />
      )}

      {aviso && <p className="exito" role="status">{aviso}</p>}
      {accion.error && <ErrorServidor error={accion.error} />}
      {lista.error && <p className="error" role="alert">{lista.error}</p>}
      {lista.cargando && !lista.datos && <p className="aviso" role="status">Cargando…</p>}
      {lista.datos && (
        <>
          <p className="total" role="status">{lista.datos.length} opciones</p>
          <div className="tabla-contenedor">
            <table>
              <thead>
                <tr>
                  <th>Orden</th>
                  <th>Valor de origen</th>
                  <th>Nombre</th>
                  <th>Estado</th>
                  <SoloAdministrador><th>Acciones</th></SoloAdministrador>
                </tr>
              </thead>
              <tbody>
                {lista.datos.map((o) => (
                  <tr key={o.id}>
                    <td>{o.orden}</td>
                    <td>{o.valor_origen === null ? <Valor v={null} /> : <code>{o.valor_origen}</code>}</td>
                    <td>{o.nombre}</td>
                    <td>
                      <span className={`estado ${o.activo ? 'estado-s' : 'estado-n'}`}>{o.activo ? 'Activo' : 'Inactivo'}</span>
                    </td>
                    <SoloAdministrador>
                      <td className="acciones-fila">
                        <button type="button" className="secundario" onClick={() => setModo(o)}>Editar</button>
                        <button type="button" className="secundario" disabled={accion.ocupado} onClick={() => void cambiarEstado(o)}>
                          {o.activo ? 'Dar de baja' : 'Dar de alta'}
                        </button>
                      </td>
                    </SoloAdministrador>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}
    </section>
  )
}

export default function Catalogos() {
  return (
    <section>
      <h1>Catálogos</h1>
      <p className="total">
        El valor de origen es el texto del Excel y no se edita; el nombre sí. La pareja deja el mapeo verificable.
      </p>
      {CATALOGOS.map((c) => (
        <Lista key={c.clave} {...c} />
      ))}
    </section>
  )
}
