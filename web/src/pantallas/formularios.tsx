import { useState, type ReactNode } from 'react'
import { ErrorApi } from '../api'
import { useEsAdministrador, useSesion } from '../sesion'

// Piezas de los formularios y de las acciones de escritura. El texto de todo error es el
// del servidor: aqui no se inventa ningun mensaje.

/**
 * Mensaje del servidor. Si trae `dependencias` (baja rechazada) las lista una por una, con
 * el total cuando el servidor mando solo una parte.
 */
export function ErrorServidor({ error }: { error: Error }) {
  const deps = error instanceof ErrorApi ? error.dependencias : []
  const total = error instanceof ErrorApi ? error.totalDependencias : 0
  return (
    <div className="error" role="alert">
      <p>{error.message}</p>
      {deps.length > 0 && (
        <ul className="dependencias">
          {deps.map((d) => (
            <li key={`${d.entidad}-${d.id}`}>
              {d.entidad} · {d.codigo}
              {d.nombre ? ` · ${d.nombre}` : ''}
            </li>
          ))}
          {total > deps.length && <li>…y {total - deps.length} más</li>}
        </ul>
      )}
    </div>
  )
}

/** El campo que nombra el error, cuando nombra exactamente uno. */
function campoUnico(error: Error | null): string | null {
  return error instanceof ErrorApi && error.campos.length === 1 ? error.campos[0] : null
}

/**
 * Error de un formulario que no se pudo colgar de un campo: no nombra ninguno, nombra
 * varios o nombra uno que el formulario no tiene. Cuando nombra un solo campo del
 * formulario, el mensaje sale junto a ese campo (CampoForm) y aqui no se repite.
 */
export function ErrorForm({ error, campos }: { error: Error | null; campos: string[] }) {
  if (!error) return null
  const unico = campoUnico(error)
  if (unico !== null && campos.includes(unico)) return null
  return <ErrorServidor error={error} />
}

/** Un campo de formulario, con el mensaje del servidor junto a el si es el que nombra. */
export function CampoForm({
  etiqueta,
  campo,
  error,
  children,
}: {
  etiqueta: string
  campo: string
  error: Error | null
  children: ReactNode
}) {
  const nombrado = error instanceof ErrorApi && error.campos.includes(campo)
  return (
    <label className={nombrado ? 'campo-invalido' : undefined}>
      {etiqueta}
      {children}
      {campoUnico(error) === campo && error && <span className="error-campo" role="alert">{error.message}</span>}
    </label>
  )
}

/**
 * Ejecuta una accion contra la API y guarda su error. Un 401 lleva al inicio de sesion.
 * `ejecutar` devuelve el resultado, o undefined si fallo (el error queda en `error`).
 */
export function useAccion() {
  const { sesionVencida } = useSesion()
  const [error, setError] = useState<Error | null>(null)
  const [ocupado, setOcupado] = useState(false)

  async function ejecutar<T>(f: () => Promise<T>): Promise<T | undefined> {
    setError(null)
    setOcupado(true)
    try {
      return await f()
    } catch (err) {
      if (err instanceof ErrorApi && err.estado === 401) sesionVencida()
      else setError(err instanceof Error ? err : new Error('Error desconocido.'))
      return undefined
    } finally {
      setOcupado(false)
    }
  }

  return { error, ocupado, ejecutar, limpiar: () => setError(null) }
}

export function Paginacion({
  pagina,
  total,
  tamano,
  onIr,
}: {
  pagina: number
  total: number
  tamano: number
  onIr: (p: number) => void
}) {
  const paginas = Math.max(1, Math.ceil(total / tamano))
  return (
    <nav className="paginacion" aria-label="Paginación">
      <button type="button" disabled={pagina <= 1} onClick={() => onIr(pagina - 1)}>Anterior</button>
      <span>Página {pagina} de {paginas}</span>
      <button type="button" disabled={pagina >= paginas} onClick={() => onIr(pagina + 1)}>Siguiente</button>
    </nav>
  )
}

/** Comodidad, no seguridad: el rol consulta no ve botones de escritura. */
export function SoloAdministrador({ children }: { children: ReactNode }) {
  return useEsAdministrador() ? <>{children}</> : null
}
