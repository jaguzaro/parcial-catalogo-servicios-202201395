import { useState, type FormEvent, type ReactNode } from 'react'
import { Link, useParams } from 'react-router'
import { api, type Servicio } from '../api'
import { Estado, useConsulta, Valor } from './comunes'
import { CampoForm, ErrorForm, ErrorServidor, SoloAdministrador, useAccion } from './formularios'

function Campo({ nombre, children }: { nombre: string; children: ReactNode }) {
  return (
    <>
      <dt>{nombre}</dt>
      <dd>{children}</dd>
    </>
  )
}

function Responsable({ s }: { s: Servicio }) {
  const sec = s.seccion_responsable
  const usu = s.usuario_responsable
  return (
    <>
      <Campo nombre="Sección responsable">
        {sec ? (
          <>
            {sec.codigo} · {sec.nombre}
            {!sec.activo && ' (inactiva)'}
            <br />
            <small>{sec.empresa.nombre} › {sec.area.nombre} › {sec.departamento.nombre}</small>
          </>
        ) : (
          <Valor v={null} />
        )}
      </Campo>
      <Campo nombre="Usuario responsable">
        {usu ? (
          <>
            {usu.nombre} ({usu.usuario}){!usu.activo && ' (inactivo)'}
          </>
        ) : (
          <Valor v={null} />
        )}
      </Campo>
    </>
  )
}

/**
 * Asignacion del responsable. La lista de usuarios se limita a los de la seccion elegida y
 * se vuelve a pedir cada vez que la seccion cambia; la regla la comprueba el servidor.
 */
function FormularioResponsable({ s, alTerminar }: { s: Servicio; alTerminar: () => void }) {
  const [seccionId, setSeccionId] = useState(s.seccion_responsable ? String(s.seccion_responsable.id) : '')
  const [usuarioId, setUsuarioId] = useState(s.usuario_responsable ? String(s.usuario_responsable.id) : '')
  const accion = useAccion()

  const secciones = useConsulta(() => api.org.listar('secciones', { activo: 'true', tamano: 100 }), 'secciones-activas')
  const usuarios = useConsulta(
    () => (seccionId ? api.usuarios({ seccion_id: seccionId, activo: 'true', tamano: 100 }) : Promise.resolve(null)),
    `usuarios-de-${seccionId}`,
  )

  const listaSecciones = secciones.datos?.items ?? []
  const actual = s.seccion_responsable
  const faltaSeccion = actual && !listaSecciones.some((x) => x.id === actual.id)
  const listaUsuarios = usuarios.datos?.items ?? []
  const actualUsuario = s.usuario_responsable
  const faltaUsuario =
    actualUsuario && String(s.seccion_responsable?.id) === seccionId && !listaUsuarios.some((u) => u.id === actualUsuario.id)

  async function enviar(e: FormEvent) {
    e.preventDefault()
    const r = await accion.ejecutar(() =>
      api.servicioResponsable(s.id, seccionId ? Number(seccionId) : null, usuarioId ? Number(usuarioId) : null),
    )
    if (r) alTerminar()
  }

  return (
    <form className="panel" onSubmit={(e) => void enviar(e)}>
      <h2>Asignar responsable</h2>
      <div className="rejilla-form">
        <CampoForm etiqueta="Sección responsable" campo="seccion_id" error={accion.error}>
          <select
            value={seccionId}
            onChange={(e) => {
              setSeccionId(e.target.value)
              setUsuarioId('') // el usuario tiene que ser de la seccion elegida
            }}
          >
            <option value="">Sin sección</option>
            {faltaSeccion && <option value={actual.id}>{actual.codigo} · {actual.nombre} (inactiva)</option>}
            {listaSecciones.map((x) => (
              <option key={x.id} value={x.id}>{x.codigo} · {x.nombre}{x.padre ? ` (${x.padre.codigo})` : ''}</option>
            ))}
          </select>
        </CampoForm>
        <CampoForm etiqueta="Usuario responsable" campo="usuario_id" error={accion.error}>
          <select value={usuarioId} onChange={(e) => setUsuarioId(e.target.value)} disabled={!seccionId}>
            <option value="">Sin usuario</option>
            {faltaUsuario && <option value={actualUsuario.id}>{actualUsuario.nombre} ({actualUsuario.usuario}) (inactivo)</option>}
            {listaUsuarios.map((u) => (
              <option key={u.id} value={u.id}>{u.nombre} ({u.usuario})</option>
            ))}
          </select>
        </CampoForm>
      </div>
      {secciones.error && <p className="error" role="alert">{secciones.error}</p>}
      {usuarios.error && <p className="error" role="alert">{usuarios.error}</p>}
      <ErrorForm error={accion.error} campos={['seccion_id', 'usuario_id']} />
      <div className="acciones">
        <button type="submit" disabled={accion.ocupado}>Guardar responsable</button>
      </div>
    </form>
  )
}

export default function Ficha() {
  const { id } = useParams()
  const numero = Number(id)
  const [version, setVersion] = useState(0)
  const [asignando, setAsignando] = useState(false)
  const accion = useAccion()
  const { datos: s, error, cargando } = useConsulta(() => api.servicio(numero), `${numero}|${version}`)

  async function cambiarEstado(desactivar: boolean) {
    const r = await accion.ejecutar(() => (desactivar ? api.servicioDesactivar(numero) : api.servicioActivar(numero)))
    if (r) setVersion((v) => v + 1)
  }

  return (
    <section>
      <p><Link to="/">← Volver al catálogo</Link></p>
      {error && <p className="error" role="alert">{error}</p>}
      {cargando && !s && !error && <p className="aviso" role="status">Cargando…</p>}
      {s && (
        <>
          <h1>{s.codigo} · {s.nombre}</h1>
          {s.requiere_revision && (
            <p className="aviso-revision">Este servicio está marcado para revisión: puede tener datos ausentes.</p>
          )}

          <SoloAdministrador>
            <div className="acciones">
              <Link className="boton" to={`/servicios/${s.id}/editar`}>Editar</Link>
              <button type="button" className="secundario" onClick={() => setAsignando((a) => !a)}>
                {asignando ? 'Cerrar asignación' : 'Asignar responsable'}
              </button>
              {s.activo !== 'N' && (
                <button type="button" className="secundario" disabled={accion.ocupado} onClick={() => void cambiarEstado(true)}>
                  Dar de baja
                </button>
              )}
              {s.activo !== 'S' && (
                <button type="button" className="secundario" disabled={accion.ocupado} onClick={() => void cambiarEstado(false)}>
                  Dar de alta
                </button>
              )}
            </div>
          </SoloAdministrador>
          {accion.error && <ErrorServidor error={accion.error} />}
          {asignando && (
            <FormularioResponsable
              key={version}
              s={s}
              alTerminar={() => {
                setAsignando(false)
                setVersion((v) => v + 1)
              }}
            />
          )}

          <dl className="ficha">
            <Campo nombre="Código">{s.codigo}</Campo>
            <Campo nombre="Nombre">{s.nombre}</Campo>
            <Campo nombre="Estado"><Estado activo={s.activo} /></Campo>
            <Campo nombre="Nivel 1">{s.servicio_n1.codigo} · {s.servicio_n1.nombre}</Campo>
            <Campo nombre="Clase"><Valor v={s.clase?.nombre} /></Campo>
            <Campo nombre="Criticidad"><Valor v={s.criticidad?.nombre} /></Campo>
            <Campo nombre="Tipo"><Valor v={s.tipo?.nombre} /></Campo>
            <Campo nombre="Descripción"><Valor v={s.descripcion} /></Campo>
            <Campo nombre="Métrica"><Valor v={s.metrica} /></Campo>
            <Campo nombre="Mínimo"><Valor v={s.minimo} /></Campo>
            <Campo nombre="Máximo"><Valor v={s.maximo} /></Campo>
            <Responsable s={s} />
          </dl>

          <h2>Incidencias de la importación</h2>
          {s.incidencias && s.incidencias.length > 0 ? (
            <ul className="incidencias">
              {s.incidencias.map((i) => (
                <li key={i.id}>
                  <strong>{i.tipo}</strong>
                  {i.regla && ` · ${i.regla}`}
                  {i.fila !== null && ` · fila ${i.fila}`}
                  <br />
                  {i.mensaje}
                </li>
              ))}
            </ul>
          ) : (
            <p>Sin incidencias.</p>
          )}
        </>
      )}
    </section>
  )
}
