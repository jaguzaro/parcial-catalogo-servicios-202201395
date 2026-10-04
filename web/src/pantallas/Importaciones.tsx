import { useState } from 'react'
import { Link, useParams } from 'react-router'
import { api, type Corrida } from '../api'
import { useConsulta, Valor } from './comunes'
import { Paginacion } from './formularios'

const TAMANO = 20

function fecha(iso: string | null): string {
  return iso ? new Date(iso).toLocaleString('es-GT') : ''
}

/** Valor leido o aplicado: JSON compacto; null es "sin dato", no una cadena vacia. */
function Json({ v }: { v: unknown }) {
  if (v === null || v === undefined) return <Valor v={null} />
  return <code>{typeof v === 'string' ? v : JSON.stringify(v)}</code>
}

function Resumen({ c }: { c: Corrida }) {
  return (
    <dl className="resumen" aria-label="Resumen de la importación">
      <div><dt>Creados</dt><dd>{c.creados}</dd></div>
      <div><dt>Actualizados</dt><dd>{c.actualizados}</dd></div>
      <div><dt>Omitidos</dt><dd>{c.omitidos}</dd></div>
      <div><dt>Observados</dt><dd>{c.observados}</dd></div>
    </dl>
  )
}

export function ListaImportaciones() {
  const [pagina, setPagina] = useState(1)
  const lista = useConsulta(() => api.importaciones({ pagina, tamano: TAMANO }), String(pagina))

  return (
    <section>
      <h1>Importaciones</h1>
      {lista.error && <p className="error" role="alert">{lista.error}</p>}
      {lista.cargando && !lista.datos && <p className="aviso" role="status">Cargando…</p>}
      {lista.datos && (
        <>
          <p className="total" role="status">
            {lista.datos.total === 0 ? 'Todavía no hay importaciones.' : `${lista.datos.total} importaciones`}
          </p>
          {lista.datos.total > 0 && (
            <div className="tabla-contenedor">
              <table>
                <thead>
                  <tr>
                    <th>Corrida</th>
                    <th>Inicio</th>
                    <th>Origen</th>
                    <th>Estado</th>
                    <th>Creados</th>
                    <th>Actualizados</th>
                    <th>Omitidos</th>
                    <th>Observados</th>
                  </tr>
                </thead>
                <tbody>
                  {lista.datos.items.map((c) => (
                    <tr key={c.id}>
                      <td><Link to={`/importaciones/${c.id}`}>#{c.id}</Link></td>
                      <td>{fecha(c.iniciada_en)}</td>
                      <td>{c.origen}</td>
                      <td><span className={`estado estado-corrida-${c.estado}`}>{c.estado}</span></td>
                      <td>{c.creados}</td>
                      <td>{c.actualizados}</td>
                      <td>{c.omitidos}</td>
                      <td>{c.observados}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
          <Paginacion pagina={lista.datos.pagina} total={lista.datos.total} tamano={TAMANO} onIr={setPagina} />
        </>
      )}
    </section>
  )
}

export function DetalleImportacion() {
  const { id } = useParams()
  const numero = Number(id)
  const { datos: c, error, cargando } = useConsulta(() => api.importacion(numero), String(numero))
  const [tipo, setTipo] = useState('')

  const incidencias = c?.incidencias ?? []
  const porTipo = new Map<string, number>()
  for (const i of incidencias) porTipo.set(i.tipo, (porTipo.get(i.tipo) ?? 0) + 1)
  const visibles = tipo ? incidencias.filter((i) => i.tipo === tipo) : incidencias

  return (
    <section>
      <p><Link to="/importaciones">← Volver a las importaciones</Link></p>
      {error && <p className="error" role="alert">{error}</p>}
      {cargando && !c && !error && <p className="aviso" role="status">Cargando…</p>}
      {c && (
        <>
          <h1>Importación #{c.id}</h1>
          <p>
            <span className={`estado estado-corrida-${c.estado}`}>{c.estado}</span> · {c.origen} · inicio{' '}
            {fecha(c.iniciada_en)}
            {c.finalizada_en && <> · fin {fecha(c.finalizada_en)}</>}
          </p>
          {c.mensaje_error && <p className="error" role="alert">{c.mensaje_error}</p>}
          <Resumen c={c} />
          <p className="total">
            Hoja <strong>{c.hoja}</strong> · SHA-256 <code>{c.archivo_sha256}</code>
          </p>

          <h2>Incidencias ({incidencias.length})</h2>
          {incidencias.length === 0 ? (
            <p>Esta corrida no registró incidencias.</p>
          ) : (
            <>
              <label className="filtro-tipo">
                Tipo
                <select value={tipo} onChange={(e) => setTipo(e.target.value)}>
                  <option value="">Todos ({incidencias.length})</option>
                  {[...porTipo].map(([t, n]) => (
                    <option key={t} value={t}>{t} ({n})</option>
                  ))}
                </select>
              </label>
              <div className="tabla-contenedor">
                <table>
                  <thead>
                    <tr>
                      <th>Tipo</th>
                      <th>Fila</th>
                      <th>Celdas</th>
                      <th>Código</th>
                      <th>Valor leído</th>
                      <th>Valor aplicado</th>
                      <th>Mensaje</th>
                    </tr>
                  </thead>
                  <tbody>
                    {visibles.map((i) => (
                      <tr key={i.id}>
                        <td>
                          <strong>{i.tipo}</strong>
                          {i.regla && <><br /><small>{i.regla}</small></>}
                        </td>
                        <td><Valor v={i.fila} /></td>
                        <td><Valor v={i.celdas} /></td>
                        <td><Valor v={i.codigo} /></td>
                        <td><Json v={i.valor_original} /></td>
                        <td><Json v={i.valor_aplicado} /></td>
                        <td>{i.mensaje}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </>
          )}
        </>
      )}
    </section>
  )
}
