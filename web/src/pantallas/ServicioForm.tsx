import { useState, type FormEvent } from 'react'
import { Link, Navigate, useNavigate, useParams } from 'react-router'
import { api, type Opcion, type OpcionCatalogo, type Servicio } from '../api'
import { useEsAdministrador } from '../sesion'
import { useConsulta } from './comunes'
import { CampoForm, ErrorForm, useAccion } from './formularios'

const CAMPOS = [
  'servicio_n1_id', 'codigo', 'nombre', 'clase_id', 'criticidad_id', 'tipo_id',
  'descripcion', 'metrica', 'minimo', 'maximo', 'requiere_revision',
]

/** Lista de opciones de un catalogo. Si la opcion actual esta inactiva se agrega, para no perderla de vista. */
function Opciones({ lista, actual }: { lista: OpcionCatalogo[]; actual: Opcion | null }) {
  const falta = actual && !lista.some((o) => o.id === actual.id)
  return (
    <>
      <option value="">Sin dato</option>
      {falta && <option value={actual.id}>{actual.nombre} (inactiva)</option>}
      {lista.map((o) => <option key={o.id} value={o.id}>{o.nombre}</option>)}
    </>
  )
}

function Formulario({ servicio }: { servicio: Servicio | null }) {
  const navegar = useNavigate()
  const accion = useAccion()
  const [n1Id, setN1Id] = useState(servicio ? String(servicio.servicio_n1.id) : '')
  const [codigo, setCodigo] = useState(servicio?.codigo ?? '')
  const [nombre, setNombre] = useState(servicio?.nombre ?? '')
  const [claseId, setClaseId] = useState(servicio?.clase ? String(servicio.clase.id) : '')
  const [criticidadId, setCriticidadId] = useState(servicio?.criticidad ? String(servicio.criticidad.id) : '')
  const [tipoId, setTipoId] = useState(servicio?.tipo ? String(servicio.tipo.id) : '')
  const [descripcion, setDescripcion] = useState(servicio?.descripcion ?? '')
  const [metrica, setMetrica] = useState(servicio?.metrica ?? '')
  // Texto del campo: vacio es "sin dato" y viaja como null, nunca como cero.
  const [minimo, setMinimo] = useState(servicio?.minimo != null ? String(servicio.minimo) : '')
  const [maximo, setMaximo] = useState(servicio?.maximo != null ? String(servicio.maximo) : '')
  const [revision, setRevision] = useState(servicio?.requiere_revision ?? false)

  const n1 = useConsulta(() => api.serviciosN1(), 'n1-formulario')
  const clases = useConsulta(() => api.catalogo('clases'), 'clases')
  const criticidades = useConsulta(() => api.catalogo('criticidades'), 'criticidades')
  const tipos = useConsulta(() => api.catalogo('tipos'), 'tipos')

  const numero = (t: string) => (t.trim() === '' ? null : Number(t))
  const id = (t: string) => (t === '' ? null : Number(t))

  async function enviar(e: FormEvent) {
    e.preventDefault()
    const cuerpo = {
      servicio_n1_id: Number(n1Id),
      codigo,
      nombre,
      clase_id: id(claseId),
      criticidad_id: id(criticidadId),
      tipo_id: id(tipoId),
      descripcion,
      metrica,
      minimo: numero(minimo),
      maximo: numero(maximo),
    }
    const r = await accion.ejecutar(() =>
      servicio
        ? api.servicioActualizar(servicio.id, { ...cuerpo, requiere_revision: revision })
        : api.servicioCrear(cuerpo),
    )
    if (r) navegar(`/servicios/${r.id}`)
  }

  const n1Activos = (n1.datos?.items ?? []).filter((s) => s.activo || String(s.id) === n1Id)
  const errores = [n1.error, clases.error, criticidades.error, tipos.error].filter(Boolean)

  return (
    <form className="panel" onSubmit={(e) => void enviar(e)}>
      <div className="rejilla-form">
        <CampoForm etiqueta="Servicio de nivel 1" campo="servicio_n1_id" error={accion.error}>
          <select value={n1Id} onChange={(e) => setN1Id(e.target.value)}>
            <option value="">Elija un servicio de nivel 1</option>
            {n1Activos.map((s) => <option key={s.id} value={s.id}>{s.codigo} · {s.nombre}</option>)}
          </select>
        </CampoForm>
        <CampoForm etiqueta="Código" campo="codigo" error={accion.error}>
          <input value={codigo} onChange={(e) => setCodigo(e.target.value)} />
        </CampoForm>
        <CampoForm etiqueta="Nombre" campo="nombre" error={accion.error}>
          <input value={nombre} onChange={(e) => setNombre(e.target.value)} />
        </CampoForm>
        <CampoForm etiqueta="Clase" campo="clase_id" error={accion.error}>
          <select value={claseId} onChange={(e) => setClaseId(e.target.value)}>
            <Opciones lista={(clases.datos ?? []).filter((o) => o.activo)} actual={servicio?.clase ?? null} />
          </select>
        </CampoForm>
        <CampoForm etiqueta="Criticidad" campo="criticidad_id" error={accion.error}>
          <select value={criticidadId} onChange={(e) => setCriticidadId(e.target.value)}>
            <Opciones lista={(criticidades.datos ?? []).filter((o) => o.activo)} actual={servicio?.criticidad ?? null} />
          </select>
        </CampoForm>
        <CampoForm etiqueta="Tipo" campo="tipo_id" error={accion.error}>
          <select value={tipoId} onChange={(e) => setTipoId(e.target.value)}>
            <Opciones lista={(tipos.datos ?? []).filter((o) => o.activo)} actual={servicio?.tipo ?? null} />
          </select>
        </CampoForm>
        <CampoForm etiqueta="Métrica" campo="metrica" error={accion.error}>
          <input value={metrica} onChange={(e) => setMetrica(e.target.value)} />
        </CampoForm>
        <CampoForm etiqueta="Mínimo" campo="minimo" error={accion.error}>
          <input type="number" step="any" min="0" value={minimo} onChange={(e) => setMinimo(e.target.value)} />
        </CampoForm>
        <CampoForm etiqueta="Máximo" campo="maximo" error={accion.error}>
          <input type="number" step="any" min="0" value={maximo} onChange={(e) => setMaximo(e.target.value)} />
        </CampoForm>
      </div>
      <div className="rejilla-form">
        <CampoForm etiqueta="Descripción" campo="descripcion" error={accion.error}>
          <textarea rows={3} value={descripcion} onChange={(e) => setDescripcion(e.target.value)} />
        </CampoForm>
        {servicio && (
          <label className="casilla">
            <input type="checkbox" checked={revision} onChange={(e) => setRevision(e.target.checked)} />
            Marcado para revisión
          </label>
        )}
      </div>
      {errores.map((m) => <p key={m} className="error" role="alert">{m}</p>)}
      <ErrorForm error={accion.error} campos={CAMPOS} />
      <div className="acciones">
        <button type="submit" disabled={accion.ocupado}>{servicio ? 'Guardar cambios' : 'Crear servicio'}</button>
        <Link to={servicio ? `/servicios/${servicio.id}` : '/'}>Cancelar</Link>
      </div>
    </form>
  )
}

export function NuevoServicio() {
  if (!useEsAdministrador()) return <Navigate to="/" replace />
  return (
    <section>
      <p><Link to="/">← Volver al catálogo</Link></p>
      <h1>Nuevo servicio de nivel 2</h1>
      <Formulario servicio={null} />
    </section>
  )
}

export function EditarServicio() {
  const { id } = useParams()
  const numero = Number(id)
  const admin = useEsAdministrador()
  const { datos: s, error, cargando } = useConsulta(() => api.servicio(numero), `editar-${numero}`)
  if (!admin) return <Navigate to={`/servicios/${numero}`} replace />
  return (
    <section>
      <p><Link to={`/servicios/${numero}`}>← Volver a la ficha</Link></p>
      {error && <p className="error" role="alert">{error}</p>}
      {cargando && !s && !error && <p className="aviso" role="status">Cargando…</p>}
      {s && (
        <>
          <h1>Editar {s.codigo}</h1>
          <Formulario servicio={s} />
        </>
      )}
    </section>
  )
}
