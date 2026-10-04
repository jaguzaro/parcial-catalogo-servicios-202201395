import { useEffect, useState, type FormEvent } from 'react'
import { Link, useSearchParams } from 'react-router'
import { api, type FiltrosServicios, type NombreCatalogo } from '../api'
import { Estado, useConsulta, Valor } from './comunes'

const TAMANO = 20

// Filtros que viajan en la direccion del navegador, junto con la pagina.
const CLAVES = ['q', 'n1_id', 'activo', 'clase_id', 'criticidad_id', 'tipo_id'] as const

const ESTADOS = [
  ['S', 'Activo'],
  ['N', 'Inactivo'],
  ['DESCONOCIDO', 'Desconocido'],
] as const

const CATALOGOS: { clave: 'clase_id' | 'criticidad_id' | 'tipo_id'; catalogo: NombreCatalogo; etiqueta: string }[] = [
  { clave: 'clase_id', catalogo: 'clases', etiqueta: 'Clase' },
  { clave: 'criticidad_id', catalogo: 'criticidades', etiqueta: 'Criticidad' },
  { clave: 'tipo_id', catalogo: 'tipos', etiqueta: 'Tipo' },
]

export default function Catalogo() {
  const [params, setParams] = useSearchParams()
  const filtros: FiltrosServicios = { pagina: Math.max(1, Number(params.get('pagina')) || 1), tamano: TAMANO }
  for (const c of CLAVES) {
    const v = params.get(c)
    if (v) filtros[c] = v
  }

  // El texto de busqueda se edita en local y se aplica al enviar; el resto aplica al cambiar.
  const [texto, setTexto] = useState(filtros.q ?? '')
  useEffect(() => setTexto(params.get('q') ?? ''), [params])

  const n1 = useConsulta(() => api.serviciosN1(), 'n1')
  const clases = useConsulta(() => api.catalogo('clases'), 'clases')
  const criticidades = useConsulta(() => api.catalogo('criticidades'), 'criticidades')
  const tipos = useConsulta(() => api.catalogo('tipos'), 'tipos')
  const opciones = { clase_id: clases.datos, criticidad_id: criticidades.datos, tipo_id: tipos.datos }

  const lista = useConsulta(() => api.servicios(filtros), params.toString())

  function cambiar(clave: string, valor: string) {
    const sig = new URLSearchParams(params)
    if (valor) sig.set(clave, valor)
    else sig.delete(clave)
    sig.delete('pagina') // un filtro nuevo vuelve a la primera pagina
    setParams(sig)
  }

  function buscar(e: FormEvent) {
    e.preventDefault()
    cambiar('q', texto.trim())
  }

  function irAPagina(p: number) {
    const sig = new URLSearchParams(params)
    if (p > 1) sig.set('pagina', String(p))
    else sig.delete('pagina')
    setParams(sig)
  }

  const hayFiltros = CLAVES.some((c) => params.get(c))
  const total = lista.datos?.total ?? 0
  const paginas = Math.max(1, Math.ceil(total / TAMANO))
  const pagina = lista.datos?.pagina ?? filtros.pagina ?? 1

  return (
    <section>
      <h1>Servicios de nivel 2</h1>

      <form className="filtros" onSubmit={buscar} role="search">
        <label className="busqueda">
          Buscar por código o nombre
          <input value={texto} onChange={(e) => setTexto(e.target.value)} placeholder="SE.01 o nombre" />
        </label>
        <button type="submit">Buscar</button>

        <label>
          Nivel 1
          <select value={params.get('n1_id') ?? ''} onChange={(e) => cambiar('n1_id', e.target.value)}>
            <option value="">Todos</option>
            {n1.datos?.items.map((s) => (
              <option key={s.id} value={s.id}>{s.codigo} · {s.nombre}</option>
            ))}
          </select>
        </label>
        <label>
          Estado
          <select value={params.get('activo') ?? ''} onChange={(e) => cambiar('activo', e.target.value)}>
            <option value="">Todos</option>
            {ESTADOS.map(([v, et]) => (
              <option key={v} value={v}>{et}</option>
            ))}
          </select>
        </label>
        {CATALOGOS.map(({ clave, etiqueta }) => (
          <label key={clave}>
            {etiqueta}
            <select value={params.get(clave) ?? ''} onChange={(e) => cambiar(clave, e.target.value)}>
              <option value="">Todas</option>
              {opciones[clave]?.map((o) => (
                <option key={o.id} value={o.id}>{o.nombre}</option>
              ))}
            </select>
          </label>
        ))}
        {hayFiltros && (
          <button type="button" className="secundario" onClick={() => setParams(new URLSearchParams())}>
            Limpiar filtros
          </button>
        )}
      </form>

      {lista.error && <p className="error" role="alert">{lista.error}</p>}
      {lista.cargando && !lista.datos && <p className="aviso" role="status">Cargando…</p>}

      {lista.datos && (
        <>
          <p className="total" role="status">
            {total === 0 ? 'Sin resultados' : `${total} servicio${total === 1 ? '' : 's'}`}
            {total > 0 && ` · página ${pagina} de ${paginas}`}
          </p>
          <div className="tabla-contenedor">
            <table>
              <thead>
                <tr>
                  <th>Código</th>
                  <th>Nombre</th>
                  <th>Nivel 1</th>
                  <th>Estado</th>
                  <th>Clase</th>
                  <th>Criticidad</th>
                  <th>Tipo</th>
                </tr>
              </thead>
              <tbody>
                {lista.datos.items.map((s) => (
                  <tr key={s.id}>
                    <td><Link to={`/servicios/${s.id}`}>{s.codigo}</Link></td>
                    <td>{s.nombre}</td>
                    <td>{s.servicio_n1.codigo} · {s.servicio_n1.nombre}</td>
                    <td><Estado activo={s.activo} /></td>
                    <td><Valor v={s.clase?.nombre} /></td>
                    <td><Valor v={s.criticidad?.nombre} /></td>
                    <td><Valor v={s.tipo?.nombre} /></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <nav className="paginacion" aria-label="Paginación">
            <button type="button" disabled={pagina <= 1} onClick={() => irAPagina(pagina - 1)}>Anterior</button>
            <span>Página {pagina} de {paginas}</span>
            <button type="button" disabled={pagina >= paginas} onClick={() => irAPagina(pagina + 1)}>Siguiente</button>
          </nav>
        </>
      )}
    </section>
  )
}
