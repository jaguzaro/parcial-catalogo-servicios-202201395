import { BrowserRouter, Link, Navigate, Outlet, Route, Routes } from 'react-router'
import { ProveedorSesion, useSesion } from './sesion'
import Login from './pantallas/Login'
import Catalogo from './pantallas/Catalogo'
import Ficha from './pantallas/Ficha'

// Todo lo que no es el formulario exige sesion; sin ella se va a /login.
function Protegido() {
  const { usuario, cerrar } = useSesion()
  if (usuario === undefined) return <p className="aviso" role="status">Consultando sesión…</p>
  if (usuario === null) return <Navigate to="/login" replace />
  return (
    <>
      <header className="barra">
        <Link to="/" className="marca">Catálogo de servicios de TI</Link>
        <span className="barra-usuario">
          {usuario.nombre} · <strong>{usuario.rol}</strong>
        </span>
        <button type="button" onClick={() => void cerrar()}>Cerrar sesión</button>
      </header>
      <main>
        <Outlet />
      </main>
    </>
  )
}

export default function App() {
  return (
    <ProveedorSesion>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route element={<Protegido />}>
            <Route path="/" element={<Catalogo />} />
            <Route path="/servicios/:id" element={<Ficha />} />
            <Route path="*" element={<Navigate to="/" replace />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </ProveedorSesion>
  )
}
