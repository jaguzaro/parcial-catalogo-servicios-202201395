import { useState, type FormEvent } from 'react'
import { useSearchParams } from 'react-router'
import { api, type Advertencia, type UsuarioFila } from '../api'
import { useConsulta, Valor } from './comunes'
import { CampoForm, ErrorForm, ErrorServidor, Paginacion, SoloAdministrador, useAccion } from './formularios'

const TAMANO = 20
const ROLES = ['administrador', 'consulta'] as const
const CAMPOS_FORM = ['nombre', 'usuario', 'correo', 'rol', 'puesto_id', 'contrasena']

type Modo = { tipo: 'nuevo' } | { tipo: 'editar'; u: UsuarioFila } | { tipo: 'contrasena'; u: UsuarioFila } | null

function FormularioUsuario({
  usuario,
  alTerminar,
  alCancelar,
}: {
  usuario: UsuarioFila | null
  alTerminar: () => void
  alCancelar: () => void
}) {
  const [nombre, setNombre] = useState(usuario?.nombre ?? '')
  const [login, setLogin] = useState(usuario?.usuario ?? '')
  const [correo, setCorreo] = useState(usuario?.correo ?? '')
  const [rol, setRol] = useState<string>(usuario?.rol ?? 'consulta')
  const [puestoId, setPuestoId] = useState(usuario ? String(usuario.puesto_id) : '')
  const [clave, setClave] = useState('')
  const accion = useAccion()

  const puestos = useConsulta(() => api.org.listar('puestos', { activo: 'true', tamano: 100 }), 'puestos-activos')
  // Si el puesto actual esta inactivo no viene en la lista; se agrega para no perderlo de vista.
  const actual = usuario?.jerarquia.puesto
  const opciones = puestos.datos?.items ?? []
  const falta = actual && !opciones.some((p) => p.id === actual.id)

  async function enviar(e: FormEvent) {
    e.preventDefault()
    const base = {
      nombre,
      usuario: login,
      correo: correo.trim() === '' ? null : correo,
      rol,
      puesto_id: Number(puestoId),
    }
    const r = await accion.ejecutar(() =>
      usuario ? api.usuarioActualizar(usuario.id, base) : api.usuarioCrear({ ...base, contrasena: clave }),
    )
    if (r) alTerminar()
  }

  return (
    <form className="panel" onSubmit={(e) => void enviar(e)}>
      <h2>{usuario ? `Editar a ${usuario.usuario}` : 'Nuevo usuario'}</h2>
      <div className="rejilla-form">
        <CampoForm etiqueta="Nombre" campo="nombre" error={accion.error}>
          <input value={nombre} onChange={(e) => setNombre(e.target.value)} />
        </CampoForm>
        <CampoForm etiqueta="Usuario" campo="usuario" error={accion.error}>
          <input value={login} onChange={(e) => setLogin(e.target.value)} autoComplete="off" />
        </CampoForm>
        <CampoForm etiqueta="Correo (opcional)" campo="correo" error={accion.error}>
          <input type="email" value={correo} onChange={(e) => setCorreo(e.target.value)} />
        </CampoForm>
        <CampoForm etiqueta="Rol" campo="rol" error={accion.error}>
          <select value={rol} onChange={(e) => setRol(e.target.value)}>
            {ROLES.map((r) => <option key={r} value={r}>{r}</option>)}
          </select>
        </CampoForm>
        <CampoForm etiqueta="Puesto" campo="puesto_id" error={accion.error}>
          <select value={puestoId} onChange={(e) => setPuestoId(e.target.value)}>
            <option value="">Elija un puesto</option>
            {falta && <option value={actual.id}>{actual.codigo} · {actual.nombre} (inactivo)</option>}
            {opciones.map((p) => (
              <option key={p.id} value={p.id}>
                {p.codigo} · {p.nombre}{p.padre ? ` (${p.padre.codigo})` : ''}
              </option>
            ))}
          </select>
        </CampoForm>
        {!usuario && (
          <CampoForm etiqueta="Contraseña" campo="contrasena" error={accion.error}>
            <input
              type="password"
              value={clave}
              onChange={(e) => setClave(e.target.value)}
              autoComplete="new-password"
            />
          </CampoForm>
        )}
      </div>
      {puestos.error && <p className="error" role="alert">{puestos.error}</p>}
      <ErrorForm error={accion.error} campos={CAMPOS_FORM} />
      <div className="acciones">
        <button type="submit" disabled={accion.ocupado}>{usuario ? 'Guardar cambios' : 'Crear usuario'}</button>
        <button type="button" className="secundario" onClick={alCancelar}>Cancelar</button>
      </div>
    </form>
  )
}

function FormularioContrasena({
  usuario,
  alTerminar,
  alCancelar,
}: {
  usuario: UsuarioFila
  alTerminar: () => void
  alCancelar: () => void
}) {
  const [clave, setClave] = useState('')
  const accion = useAccion()

  async function enviar(e: FormEvent) {
    e.preventDefault()
    const r = await accion.ejecutar(async () => {
      await api.usuarioContrasena(usuario.id, clave)
      return true
    })
    if (r) alTerminar()
  }

  return (
    <form className="panel" onSubmit={(e) => void enviar(e)}>
      <h2>Cambiar la contraseña de {usuario.usuario}</h2>
      <div className="rejilla-form">
        <CampoForm etiqueta="Contraseña nueva" campo="contrasena" error={accion.error}>
          <input
            type="password"
            value={clave}
            onChange={(e) => setClave(e.target.value)}
            autoComplete="new-password"
          />
        </CampoForm>
      </div>
      <ErrorForm error={accion.error} campos={['contrasena']} />
      <div className="acciones">
        <button type="submit" disabled={accion.ocupado}>Cambiar contraseña</button>
        <button type="button" className="secundario" onClick={alCancelar}>Cancelar</button>
      </div>
    </form>
  )
}

export default function Usuarios() {
  const [params, setParams] = useSearchParams()
  const [modo, setModo] = useState<Modo>(null)
  const [version, setVersion] = useState(0)
  const [advertencias, setAdvertencias] = useState<Advertencia[]>([])
  const [aviso, setAviso] = useState<string | null>(null)
  const accion = useAccion()

  const pagina = Math.max(1, Number(params.get('pagina')) || 1)
  const filtros = {
    q: params.get('q') ?? '',
    rol: params.get('rol') ?? '',
    activo: params.get('activo') ?? '',
  }
  const [texto, setTexto] = useState(filtros.q)
  const lista = useConsulta(
    () => api.usuarios({ ...filtros, pagina, tamano: TAMANO }),
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

  async function desactivar(u: UsuarioFila) {
    setAviso(null)
    setAdvertencias([])
    const r = await accion.ejecutar(() => api.usuarioDesactivar(u.id))
    if (r) {
      setAdvertencias(r.advertencias)
      setVersion((v) => v + 1)
    }
  }

  async function activar(u: UsuarioFila) {
    setAviso(null)
    setAdvertencias([])
    if (await accion.ejecutar(() => api.usuarioActivar(u.id))) setVersion((v) => v + 1)
  }

  return (
    <section>
      <h1>Usuarios</h1>

      <form
        className="filtros"
        role="search"
        onSubmit={(e) => {
          e.preventDefault()
          cambiar('q', texto.trim())
        }}
      >
        <label className="busqueda">
          Buscar por nombre, usuario o correo
          <input value={texto} onChange={(e) => setTexto(e.target.value)} />
        </label>
        <button type="submit">Buscar</button>
        <label>
          Rol
          <select value={filtros.rol} onChange={(e) => cambiar('rol', e.target.value)}>
            <option value="">Todos</option>
            {ROLES.map((r) => <option key={r} value={r}>{r}</option>)}
          </select>
        </label>
        <label>
          Estado
          <select value={filtros.activo} onChange={(e) => cambiar('activo', e.target.value)}>
            <option value="">Todos</option>
            <option value="true">Activos</option>
            <option value="false">Inactivos</option>
          </select>
        </label>
        <SoloAdministrador>
          <button type="button" onClick={() => setModo({ tipo: 'nuevo' })}>Nuevo usuario</button>
        </SoloAdministrador>
      </form>

      {modo?.tipo === 'nuevo' && (
        <FormularioUsuario usuario={null} alCancelar={() => setModo(null)} alTerminar={() => terminar('Usuario creado.')} />
      )}
      {modo?.tipo === 'editar' && (
        <FormularioUsuario
          key={modo.u.id}
          usuario={modo.u}
          alCancelar={() => setModo(null)}
          alTerminar={() => terminar('Cambios guardados.')}
        />
      )}
      {modo?.tipo === 'contrasena' && (
        <FormularioContrasena
          key={modo.u.id}
          usuario={modo.u}
          alCancelar={() => setModo(null)}
          alTerminar={() => terminar('Contraseña cambiada.')}
        />
      )}

      {aviso && <p className="exito" role="status">{aviso}</p>}
      {accion.error && <ErrorServidor error={accion.error} />}
      {advertencias.map((a) => (
        <div key={a.codigo} className="aviso-revision" role="status">
          <p>{a.mensaje}</p>
          {a.servicios.length > 0 && (
            <ul>
              {a.servicios.map((s) => <li key={s.id}>{s.codigo}{s.nombre ? ` · ${s.nombre}` : ''}</li>)}
            </ul>
          )}
        </div>
      ))}

      {lista.error && <p className="error" role="alert">{lista.error}</p>}
      {lista.cargando && !lista.datos && <p className="aviso" role="status">Cargando…</p>}
      {lista.datos && (
        <>
          <p className="total" role="status">
            {lista.datos.total === 0 ? 'Sin resultados' : `${lista.datos.total} usuarios`}
          </p>
          <div className="tabla-contenedor">
            <table>
              <thead>
                <tr>
                  <th>Usuario</th>
                  <th>Nombre</th>
                  <th>Correo</th>
                  <th>Rol</th>
                  <th>Puesto</th>
                  <th>Estado</th>
                  <SoloAdministrador><th>Acciones</th></SoloAdministrador>
                </tr>
              </thead>
              <tbody>
                {lista.datos.items.map((u) => (
                  <tr key={u.id}>
                    <td>{u.usuario}</td>
                    <td>{u.nombre}</td>
                    <td><Valor v={u.correo} /></td>
                    <td>{u.rol}</td>
                    <td>
                      {u.jerarquia.puesto.codigo} · {u.jerarquia.puesto.nombre}
                      <br />
                      <small>{u.jerarquia.empresa.nombre} › {u.jerarquia.area.nombre} › {u.jerarquia.departamento.nombre} › {u.jerarquia.seccion.nombre}</small>
                    </td>
                    <td>
                      <span className={`estado ${u.activo ? 'estado-s' : 'estado-n'}`}>{u.activo ? 'Activo' : 'Inactivo'}</span>
                    </td>
                    <SoloAdministrador>
                      <td className="acciones-fila">
                        <button type="button" className="secundario" onClick={() => setModo({ tipo: 'editar', u })}>Editar</button>
                        <button type="button" className="secundario" onClick={() => setModo({ tipo: 'contrasena', u })}>Contraseña</button>
                        {u.activo ? (
                          <button type="button" className="secundario" disabled={accion.ocupado} onClick={() => void desactivar(u)}>Dar de baja</button>
                        ) : (
                          <button type="button" className="secundario" disabled={accion.ocupado} onClick={() => void activar(u)}>Dar de alta</button>
                        )}
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
