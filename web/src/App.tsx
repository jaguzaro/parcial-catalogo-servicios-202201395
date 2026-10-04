import { BrowserRouter, Link, Navigate, NavLink, Outlet, Route, Routes } from 'react-router'
import { ProveedorSesion, useSesion } from './sesion'
import Login from './pantallas/Login'
import Catalogo from './pantallas/Catalogo'
import Ficha from './pantallas/Ficha'
import { EditarServicio, NuevoServicio } from './pantallas/ServicioForm'
import ServiciosN1 from './pantallas/ServiciosN1'
import Organizacion from './pantallas/Organizacion'
import Usuarios from './pantallas/Usuarios'
import { DetalleImportacion, ListaImportaciones } from './pantallas/Importaciones'

// Todo lo que no es el formulario exige sesion; sin ella se va a /login.
function Protegido() {
  const { usuario, cerrar } = useSesion()
  if (usuario === undefined) return <p className="aviso" role="status">Consultando sesión…</p>
  if (usuario === null) return <Navigate to="/login" replace />
  return (
    <>
      <header className="barra">
        <Link to="/" className="marca">Catálogo de servicios de TI</Link>
        <nav className="menu" aria-label="Secciones">
          <NavLink to="/" end>Servicios</NavLink>
          <NavLink to="/servicios-n1">Nivel 1</NavLink>
          <NavLink to="/organizacion">Organización</NavLink>
          <NavLink to="/usuarios">Usuarios</NavLink>
          <NavLink to="/importaciones">Importaciones</NavLink>
        </nav>
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
            <Route path="/servicios/nuevo" element={<NuevoServicio />} />
            <Route path="/servicios/:id" element={<Ficha />} />
            <Route path="/servicios/:id/editar" element={<EditarServicio />} />
            <Route path="/servicios-n1" element={<ServiciosN1 />} />
            <Route path="/organizacion" element={<Navigate to="/organizacion/empresas" replace />} />
            <Route path="/organizacion/:recurso" element={<Organizacion />} />
            <Route path="/usuarios" element={<Usuarios />} />
            <Route path="/importaciones" element={<ListaImportaciones />} />
            <Route path="/importaciones/:id" element={<DetalleImportacion />} />
            <Route path="*" element={<Navigate to="/" replace />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </ProveedorSesion>
  )
}
