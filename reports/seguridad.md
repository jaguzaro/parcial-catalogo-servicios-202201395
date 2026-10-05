# Reporte de seguridad

Alcance: repositorio `/Users/joelguzaro/Documents/GitHub/parcial-catalogo-servicios-202201395`, commit 09b7ef9, contra `docs/contexto/enunciado.md` sección 3.1 y `AGENTS.md`. Auditoría de solo lectura. No se leyó `.env`, solo `.env.example`. Cada hallazgo se verificó leyendo el código citado.

## Resumen de conteos por severidad

| Severidad | Cantidad |
|---|---|
| Crítica | 0 |
| Alta | 0 |
| Media | 1 |
| Baja | 5 |
| **Total** | **6** |

No se encontró ningún defecto que incumpla de forma directa los requisitos de 3.1. Los hallazgos son de endurecimiento.

## Hallazgos

### SEG-01 (media) Sin limitación de intentos en el inicio de sesión
- Ubicación: `internal/http/router.go:85` (ruta `POST /api/auth/login`), `internal/http/auth.go:61-95` y `internal/auth/sesion.go:91-133`.
- Problema: no hay límite de intentos fallidos, bloqueo temporal ni retardo progresivo por usuario o por IP. Un atacante puede probar contraseñas sin freno. Argon2id (64 MiB por verificación) encarece cada intento, pero también permite agotar CPU y memoria con peticiones concurrentes. Hay una búsqueda `grep` de rate limit sin resultados.
- Propuesta: añadir un límite de intentos por usuario e IP con bloqueo o retardo temporal. Acotar también la concurrencia de verificaciones argon2.

### SEG-02 (baja) La copia automática de `.env.example` deja credenciales conocidas y la cookie sin `Secure`
- Ubicación: `Makefile:16-17` (`setup` hace `cp .env.example .env`). Los valores están en `.env.example:8,16,22,24`.
- Problema: `make up` crea el `.env` con contraseñas de ejemplo públicas. Esto incluye la clave de la base y las cuentas demo, que son administrador y consulta. `COOKIE_SECURE=false` queda por defecto, también en `compose.yaml:27`. Si se despliega sin editarlas, las cuentas de demostración tienen claves publicadas en el repositorio. El `.env.example` en sí cumple: no tiene valores reales y los marca como EJEMPLO.
- Propuesta: hacer que `seed-demo` y `servir` se nieguen a arrancar con valores que empiecen por `EJEMPLO-` cuando `COOKIE_SECURE=true`, o avisar de forma visible en el arranque.

### SEG-03 (baja) Sin cabeceras de seguridad HTTP
- Ubicación: `internal/http/router.go:59-66` (construcción del mux) y `router.go:176-201` (servicio de la interfaz).
- Problema: ni la API ni la interfaz establecen `Content-Security-Policy`, `X-Content-Type-Options: nosniff`, `X-Frame-Options` o `frame-ancestors`, ni `Strict-Transport-Security` cuando se sirve por HTTPS. No hay coincidencias de esos nombres en ningún `.go`. Esto facilita clickjacking y reduce la defensa en profundidad frente a XSS.
- Propuesta: envolver el mux con un middleware que fije estas cabeceras, con HSTS solo si `COOKIE_SECURE` es true.

### SEG-04 (baja) Defensa CSRF apoyada solo en `SameSite=Lax`
- Ubicación: `internal/http/auth.go:17-35` (`exigirJSON`) y `internal/auth/sesion.go:210-220`.
- Problema: los POST sin cuerpo omiten la comprobación de `Content-Type`. Esto cubre `desactivar`, `activar` y `logout`, y el propio comentario del código lo acepta. La única barrera es `SameSite=Lax`. Esa barrera no protege frente a peticiones originadas en otro subdominio del mismo sitio, ni en navegadores que no apliquen Lax.
- Propuesta: exigir una cabecera personalizada, por ejemplo `X-Requested-With`, o validar `Origin` o `Sec-Fetch-Site` en toda escritura.

### SEG-05 (baja) Servidor HTTP sin `ReadTimeout`, `WriteTimeout` ni `IdleTimeout`
- Ubicación: `cmd/catalogo/main.go:117-120`.
- Problema: solo está definido `ReadHeaderTimeout: 5s`. Un cliente lento puede mantener abiertas conexiones durante la lectura del cuerpo, que admite hasta 1 MiB según `internal/http/auth.go:15`, y de la respuesta. Esto es un riesgo de agotamiento de recursos.
- Propuesta: definir `ReadTimeout`, `WriteTimeout` e `IdleTimeout` razonables.

### SEG-06 (baja) La política de contraseñas solo exige longitud mínima de 8
- Ubicación: `internal/auth/contrasena.go:26-28,38-45`.
- Problema: solo se valida la longitud, entre 8 y 256 caracteres. No se rechazan contraseñas triviales como `12345678`. El enunciado no exige más, y esto es un riesgo residual.
- Propuesta: añadir una lista corta de contraseñas comunes prohibidas, o subir el mínimo a 10 o 12 caracteres.

## Áreas revisadas sin hallazgos

- **Hash de contraseñas:** cumple. Se usa argon2id con sal aleatoria de 16 bytes y parámetros m=64 MiB, t=3, p=4 (`internal/auth/contrasena.go:19-60`). La comparación es en tiempo constante (línea 89). Existe un hash ficticio para igualar tiempos cuando el usuario no existe (líneas 97-109). La base lo refuerza con `CHECK (hash_contrasena LIKE '$argon2id$%')` (`internal/db/migraciones/00002_organizacion.sql:91`). No hay texto plano ni cifrado reversible.
- **Autorización en el servidor:** cumple. Toda ruta de `/api/` pasa por `lectura` o `escritura` (`internal/http/router.go:77-153`). Las 38 rutas de escritura usan `RequerirSesion` más `RequerirRol(administrador)`. Solo `login` es pública, como corresponde. El rol se lee de la base en cada petición (`internal/auth/sesion.go:145-150`), así que un cambio de rol o una baja tienen efecto inmediato. `RequerirRol` está pensado para ir dentro de `RequerirSesion`. La ruta comodín `/api/` responde 404 y no tiene ningún camino sin autenticar.
- **Rol consulta sin hashes ni secretos:** cumple. El tipo `usuarios.Usuario` (`internal/usuarios/usuarios.go:42-55`) y `auth.Usuario` (`internal/auth/sesion.go:53-61`) no tienen campo para el hash. Las consultas `selectUsuario` (`usuarios.go:100-109`) no seleccionan `hash_contrasena`. La única lectura del hash es la de `IniciarSesion` (`sesion.go:101`), que no se serializa. Los usuarios y su jerarquía son legibles por consulta, lo que coincide con `arquitectura.md:168`. El token tampoco sale en JSON: viaja solo en la cookie. Rol consulta sin escritura: cumple, por la cobertura de `escritura` mencionada arriba.
- **Usuarios desactivados y cierre de sesión:** cumple.
  - `Autenticar` exige `activo` y revoca la sesión si el usuario está inactivo (`internal/auth/sesion.go:157-163`).
  - `IniciarSesion` rechaza usuarios inactivos con el mismo error genérico (línea 115).
  - `Desactivar` revoca todas las sesiones en la misma transacción (`internal/usuarios/usuarios.go:541-546`).
  - `FijarContrasena` también revoca las sesiones (línea 511).
  - `logout` revoca el token y borra la cookie (`internal/http/auth.go:97-106`, `sesion.go:168-175`).
  - Los tokens son de 32 bytes aleatorios, en la base solo se guarda el SHA-256 (`sesion.go:76-87`) y expiran por TTL.
  - Se protege al último administrador (`usuarios.go:381-399`).
- **SQL parametrizada y validación de entrada:** cumple.
  - Todos los valores de usuario van como `$n`. Los `Sprintf` que arman SQL (`internal/organizacion/organizacion.go`, `internal/catalogo/opciones.go`, `n1.go`, `n2.go`) interpolan solo nombres de tabla o columna tomados de estructuras constantes del código (`Niveles`, `Clases`, `Criticidades`, `Tipos`), nunca de la petición. El recurso se resuelve por comparación exacta (`opciones.go:38`).
  - En `internal/auth/sesion.go:92-102`, la columna `usuario` o `correo` sale de un `if` fijo, no del input.
  - `LIKE` escapa comodines (`internal/db/listado.go:30-35`).
  - `leerJSON` limita el cuerpo a 1 MiB, rechaza campos desconocidos y varios objetos (`internal/http/auth.go:37-50`).
  - Hay validación de longitudes, rol y formato en `usuarios.validar` (`usuarios.go:231-271`).
- **Secretos y Git:** cumple. `.gitignore:2-4` excluye `.env` y `.env.*`, y permite solo `.env.example`. `.env.example` solo tiene valores marcados como EJEMPLO. Las contraseñas demo no se imprimen (`internal/demo/demo.go:79-83`). La variable `DATABASE_URL` se arma en `compose.yaml` a partir de `.env` y no hay secretos reales en el código. El Excel se monta como `:ro` (`compose.yaml:36`). La imagen final es distroless, sin shell y con usuario `nonroot` (`Dockerfile:19-25`).
- **Cookies:** cumple. La cookie lleva `HttpOnly`, `SameSite=Lax`, `Path=/`, `MaxAge` igual al TTL, y `Secure` configurable (`internal/auth/sesion.go:210-233`). El único punto débil es el valor por defecto `Secure=false` para `http://localhost`, tratado en SEG-02.
