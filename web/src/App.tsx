import { useEffect, useState } from 'react'

type Estado = 'consultando' | 'ok' | 'sin_base' | 'sin_respuesta'

const etiquetas: Record<Estado, string> = {
  consultando: 'Consultando…',
  ok: 'Servidor y base de datos operativos',
  sin_base: 'El servidor responde, pero la base de datos no',
  sin_respuesta: 'No hay respuesta del servidor',
}

export default function App() {
  const [estado, setEstado] = useState<Estado>('consultando')

  useEffect(() => {
    fetch('/healthz', { credentials: 'same-origin' })
      .then(async (resp) => {
        const cuerpo = (await resp.json()) as { estado?: string }
        setEstado(cuerpo.estado === 'ok' ? 'ok' : 'sin_base')
      })
      .catch(() => setEstado('sin_respuesta'))
  }, [])

  return (
    <main style={{ fontFamily: 'system-ui, sans-serif', maxWidth: 640, margin: '4rem auto' }}>
      <h1>Catálogo de servicios de TI</h1>
      <p role="status">{etiquetas[estado]}</p>
    </main>
  )
}
