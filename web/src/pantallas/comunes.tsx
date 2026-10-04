import { useEffect, useRef, useState } from 'react'
import { ErrorApi } from '../api'
import { useSesion } from '../sesion'

/** Lo que no tiene valor se muestra como "sin dato": ni cero ni vacio. */
export function SinDato() {
  return <span className="sin-dato">sin dato</span>
}

export function Valor({ v }: { v: string | number | null | undefined }) {
  return v === null || v === undefined || v === '' ? <SinDato /> : <>{v}</>
}

const ESTADOS = { S: 'Activo', N: 'Inactivo', DESCONOCIDO: 'Desconocido' } as const

export function Estado({ activo }: { activo: keyof typeof ESTADOS }) {
  return <span className={`estado estado-${activo.toLowerCase()}`}>{ESTADOS[activo]}</span>
}

export interface Carga<T> {
  datos: T | null
  error: string | null
  cargando: boolean
}

/**
 * Ejecuta una consulta cada vez que cambia `clave`. Un 401 lleva al inicio de sesion;
 * cualquier otro error se muestra con el mensaje del servidor. Descarta respuestas viejas.
 */
export function useConsulta<T>(consulta: () => Promise<T>, clave: string): Carga<T> {
  const { sesionVencida } = useSesion()
  const [estado, setEstado] = useState<Carga<T>>({ datos: null, error: null, cargando: true })
  const ultima = useRef(consulta)
  ultima.current = consulta

  useEffect(() => {
    let vigente = true
    setEstado((e) => ({ ...e, error: null, cargando: true }))
    ultima.current().then(
      (datos) => vigente && setEstado({ datos, error: null, cargando: false }),
      (err: unknown) => {
        if (!vigente) return
        if (err instanceof ErrorApi && err.estado === 401) {
          sesionVencida()
          return
        }
        setEstado({ datos: null, cargando: false, error: err instanceof Error ? err.message : 'Error desconocido.' })
      },
    )
    return () => {
      vigente = false
    }
  }, [clave, sesionVencida])

  return estado
}
