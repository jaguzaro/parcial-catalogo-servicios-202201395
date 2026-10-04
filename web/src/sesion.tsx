import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from 'react'
import { api, type Usuario } from './api'

interface ContextoSesion {
  /** undefined mientras se consulta quien soy; null si no hay sesion. */
  usuario: Usuario | null | undefined
  iniciar: (login: string, contrasena: string) => Promise<void>
  cerrar: () => Promise<void>
  /** Para cualquier pantalla: un 401 de la API lleva al inicio de sesion. */
  sesionVencida: () => void
}

const Ctx = createContext<ContextoSesion | null>(null)

export function ProveedorSesion({ children }: { children: ReactNode }) {
  const [usuario, setUsuario] = useState<Usuario | null | undefined>(undefined)

  useEffect(() => {
    api
      .yo()
      .then(setUsuario)
      .catch(() => setUsuario(null)) // 401 u otro fallo: sin sesion, a iniciar sesion
  }, [])

  const iniciar = useCallback(async (login: string, contrasena: string) => {
    setUsuario(await api.login(login, contrasena))
  }, [])

  const cerrar = useCallback(async () => {
    try {
      await api.logout()
    } finally {
      setUsuario(null)
    }
  }, [])

  const sesionVencida = useCallback(() => setUsuario(null), [])

  return <Ctx.Provider value={{ usuario, iniciar, cerrar, sesionVencida }}>{children}</Ctx.Provider>
}

export function useSesion(): ContextoSesion {
  const c = useContext(Ctx)
  if (!c) throw new Error('useSesion fuera de ProveedorSesion')
  return c
}

/**
 * Comodidad, no seguridad: solo decide si se dibujan los botones de escritura. La
 * autorizacion real la aplica el servidor (403 para el rol consulta en toda ruta de
 * escritura), asi que quitar esta comprobacion no abre nada.
 */
export function useEsAdministrador(): boolean {
  return useSesion().usuario?.rol === 'administrador'
}
