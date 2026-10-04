import type { ReactNode } from 'react'
import { Link, useParams } from 'react-router'
import { api, type Servicio } from '../api'
import { Estado, useConsulta, Valor } from './comunes'

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

export default function Ficha() {
  const { id } = useParams()
  const numero = Number(id)
  const { datos: s, error, cargando } = useConsulta(() => api.servicio(numero), String(numero))

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
