import { useState, type FormEvent } from 'react'
import { Navigate, useNavigate } from 'react-router'
import { useSesion } from '../sesion'

export default function Login() {
  const { usuario, iniciar } = useSesion()
  const navegar = useNavigate()
  const [login, setLogin] = useState('')
  const [contrasena, setContrasena] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [enviando, setEnviando] = useState(false)

  if (usuario) return <Navigate to="/" replace />

  async function enviar(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setEnviando(true)
    try {
      await iniciar(login, contrasena)
      navegar('/', { replace: true })
    } catch (err) {
      // Se muestra tal cual el mensaje del servidor.
      setError(err instanceof Error ? err.message : 'No se pudo contactar al servidor.')
    } finally {
      setEnviando(false)
    }
  }

  return (
    <main className="login">
      <h1>Catálogo de servicios de TI</h1>
      <form onSubmit={(e) => void enviar(e)}>
        <label>
          Usuario o correo
          <input value={login} onChange={(e) => setLogin(e.target.value)} autoComplete="username" required autoFocus />
        </label>
        <label>
          Contraseña
          <input
            type="password"
            value={contrasena}
            onChange={(e) => setContrasena(e.target.value)}
            autoComplete="current-password"
            required
          />
        </label>
        {error && <p className="error" role="alert">{error}</p>}
        <button type="submit" disabled={enviando}>{enviando ? 'Entrando…' : 'Entrar'}</button>
      </form>
    </main>
  )
}
