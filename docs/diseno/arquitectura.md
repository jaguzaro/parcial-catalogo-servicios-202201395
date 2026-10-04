# Arquitectura

El stack ya esta decidido en D14 a D19; aqui no se vuelve a abrir. Este documento explica
como encajan las piezas, la estructura del repositorio, la API y el empaquetado en Docker.

## Piezas

| Pieza | Decision | Papel |
|---|---|---|
| Backend | Go, `net/http` con los patrones de ruta de Go 1.22 (D14) | Un solo binario `catalogo` con subcomandos: sirve la API y la interfaz, migra, importa y siembra datos de demostracion. |
| Acceso a datos | `pgx/v5` con `pgxpool` (D14) | SQL escrito a mano, con parametros (`$1`), nunca concatenado. Las transacciones las delimita la capa de servicio. |
| Migraciones | `goose`, embebidas con `embed` (D14) | Archivos `.sql` dentro del binario. `servir` las aplica al arrancar y `migrar` las aplica sin servir. |
| Lectura del Excel | `excelize` (D14) | Solo en el paquete del importador. |
| Contrasenas | argon2id, `golang.org/x/crypto/argon2` (D15) | [reglas.md](reglas.md#autenticacion-y-sesiones). |
| Sesiones | Token opaco, cookie `HttpOnly`, SHA-256 en la base (D16) | Tabla `sesion`. |
| Interfaz | React, Vite, TypeScript, `react-router` (D17) | Se compila en Docker y queda embebida en el binario. Mismo origen que la API. |
| Base | PostgreSQL 17 con volumen y `healthcheck` (D18) | Unica fuente de verdad. |
| Pruebas | Integracion en Go contra `catalogo_test`, proyecto `catalogo-test` (D19) | `TestP01_...` a `TestP12_...`. |

Flujo de una peticion:

```text
navegador ── http://localhost:8080 ──> catalogo servir
                                        ├─ /healthz            salud (sin sesion)
                                        ├─ /api/...            middleware: JSON -> sesion -> rol -> handler -> servicio -> pgx -> PostgreSQL
                                        └─ /* (todo lo demas)  archivos de la interfaz embebidos; si no existe, index.html
```

Capas del backend, de afuera hacia adentro:

1. **`http`**: enrutador, middlewares, decodificar el JSON y traducir errores a respuestas.
   No tiene reglas de negocio.
2. **Servicios de dominio** (`organizacion`, `usuarios`, `catalogo`, `importador`):
   validan, abren la transaccion y aplican las reglas de [reglas.md](reglas.md).
3. **`db`**: pool, ayuda para transacciones y traduccion de errores de PostgreSQL
   (`23505` a `CODIGO_DUPLICADO`, `23503` a `REFERENCIA_INEXISTENTE`, `23514` a
   `VALIDACION`).

El importador lo usan dos entradas, `catalogo importar` y `POST /api/importaciones`, con
el mismo codigo.

## Estructura del repositorio

```text
.
├── AGENTS.md, CLAUDE.md         contexto para asistentes (existe)
├── README.md                    como levantar, probar y evaluar
├── Dockerfile                   construccion en tres etapas: web, go y final
├── .dockerignore                deja fuera .env, node_modules, logs y docs del contexto de construccion
├── compose.yaml                 servicios db y app, volumen y healthchecks
├── compose.test.yaml            ajustes para el proyecto catalogo-test y servicio de pruebas
├── .env.example                 variables documentadas, sin secretos reales
├── Makefile                     comandos del harness (pendiente, llega con su prompt)
├── go.mod, go.sum               modulo de Go
├── cmd/catalogo/                main y subcomandos: servir, migrar, importar, sembrar-demo, salud
├── internal/config/             lectura y validacion de variables de entorno
├── internal/db/                 pool pgx, transacciones, traduccion de errores de PostgreSQL
├── internal/db/migraciones/     archivos .sql de goose, embebidos
├── internal/auth/               argon2id, sesiones, middlewares de sesion y de rol
├── internal/http/               enrutador, handlers por recurso, formato de error, paginacion
├── internal/organizacion/       empresa, area, departamento, seccion, puesto: reglas y consultas
├── internal/usuarios/           usuarios, contrasenas, ultimo administrador
├── internal/catalogo/           servicios N1 y N2, catalogos, responsables, busqueda
├── internal/importador/         lectura del Excel y reglas D08 a D13
├── internal/demo/               sembrado de la estructura, las cuentas y las asignaciones de demostracion
├── internal/web/                embed de la interfaz compilada (dist/ se genera en Docker)
├── web/                         codigo de la interfaz: React, Vite, TypeScript
├── tests/integracion/           pruebas TestP01 a TestP12 contra catalogo_test
├── tests/fixtures/              libros Excel de ensayo generados por las pruebas (nunca el original)
├── scripts/analisis/            analisis del Excel (existe)
├── data/CatalogoServicios.xlsx  archivo original, solo lectura (existe)
├── docs/                        contexto, prompts, diseno, evidencias, RESOLUCION.md
├── hooks/, .claude/             harness del asistente (existe, no se modifica)
└── logs/                        registro de los hooks (existe)
```

`internal/web/dist/` no se versiona, salvo un `index.html` minimo de relleno para que
`go build` y `go vet` funcionen fuera de Docker. La construccion lo reemplaza por la
interfaz real.

## API

Todo cuerpo de peticion y de respuesta es JSON. Las fechas van en RFC 3339 y los
identificadores son numeros.

Abreviaturas de la columna "Quien": **P** es publico, **S** cualquier sesion valida
(administrador o consulta) y **A** solo administrador.

### Formato de error

```json
{"error": {"codigo": "CODIGO_DUPLICADO", "mensaje": "Ya existe un area con codigo TI en la empresa EMP01.", "campos": ["codigo"]}}
```

`campos`, `dependencias`, `total_dependencias` y `detalle` aparecen solo cuando aplican.

| HTTP | `codigo` | Cuando |
|---|---|---|
| 400 | `JSON_INVALIDO` | Cuerpo que no es JSON valido, o con campos desconocidos. |
| 401 | `NO_AUTENTICADO` | Sin sesion, sesion vencida o revocada, o usuario inactivo. |
| 401 | `CREDENCIALES_INVALIDAS` | Inicio de sesion fallido. |
| 403 | `PROHIBIDO` | El rol no alcanza para la operacion. |
| 404 | `NO_ENCONTRADO` | El `{id}` de la ruta no existe. |
| 409 | `CODIGO_DUPLICADO` | Violacion de unicidad. |
| 409 | `PADRE_INACTIVO` | Alta, movimiento o reactivacion bajo un padre o una referencia inactivos. |
| 409 | `DEPENDENCIAS_ACTIVAS` | Baja con dependientes activos, con la lista. |
| 409 | `ASIGNACIONES_AFECTADAS` | Movimiento que dejaria invalida la asignacion de un responsable. |
| 409 | `ULTIMO_ADMINISTRADOR` | Desactivar o degradar al ultimo administrador activo. |
| 409 | `SERVICIO_INACTIVO` | Asignar responsable a un servicio con `activo = 'N'`. |
| 409 | `IMPORTACION_EN_CURSO` | Otra importacion tiene el candado. |
| 415 | `TIPO_NO_SOPORTADO` | `POST` o `PUT` sin `Content-Type: application/json`. |
| 422 | `VALIDACION` | Campo obligatorio vacio, formato o longitud incorrectos, dominio cerrado. |
| 422 | `REFERENCIA_INEXISTENTE` | Un `*_id` del cuerpo no existe. |
| 422 | `MINIMO_MAYOR_QUE_MAXIMO` | `minimo > maximo`. |
| 422 | `RESPONSABLE_FUERA_DE_SECCION` | El usuario no pertenece a la seccion responsable. |
| 500 | `INTERNO` | Error no previsto. El mensaje es generico y el detalle va solo al log. |

Toda ruta que pide sesion puede responder 401, y toda ruta solo de administrador puede
responder 403. En las tablas siguientes eso no se repite.

### Listados

Todo `GET` de coleccion acepta `pagina` (desde 1, por defecto 1) y `tamano` (por defecto
20, maximo 100; un valor mayor es 422) y devuelve:

```json
{"items": [ ... ], "total": 46, "pagina": 1, "tamano": 20}
```

Orden por defecto: `codigo` ascendente. En usuarios, `usuario` ascendente.

### Autenticacion

| Metodo | Ruta | Quien | Recibe | Devuelve | Errores |
|---|---|---|---|---|---|
| `POST` | `/api/auth/login` | P | `{login, contrasena}` | 200 `{usuario}` (sin hash) y la cookie `catalogo_sesion` | 400, 401 `CREDENCIALES_INVALIDAS`, 415, 422 |
| `POST` | `/api/auth/logout` | S | `{}` | 204 y borra la cookie | 401, 415 |
| `GET` | `/api/auth/me` | S | | 200 `{usuario}` con rol y jerarquia | 401 |

### Estructura organizacional

Las cinco unidades usan el mismo juego de rutas:

| Recurso (`{r}`) | Campo padre | Filtro por padre |
|---|---|---|
| `empresas` | — | — |
| `areas` | `empresa_id` | `?empresa_id=` |
| `departamentos` | `area_id` | `?area_id=` |
| `secciones` | `departamento_id` | `?departamento_id=` |
| `puestos` | `seccion_id` | `?seccion_id=` |

| Metodo | Ruta | Quien | Recibe | Devuelve | Errores |
|---|---|---|---|---|---|
| `GET` | `/api/{r}` | S | Filtro por padre, `q` (codigo o nombre), `activo` (`true`/`false`), paginacion | 200 listado; cada item trae su padre `{id, codigo, nombre}` | 422 |
| `GET` | `/api/{r}/{id}` | S | | 200 registro con la ruta de ancestros | 404 |
| `POST` | `/api/{r}` | A | `{codigo, nombre, <padre>_id}` | 201 registro, cabecera `Location` | 409 `CODIGO_DUPLICADO`, `PADRE_INACTIVO`; 415; 422 `VALIDACION`, `REFERENCIA_INEXISTENTE` |
| `PUT` | `/api/{r}/{id}` | A | `{codigo, nombre, <padre>_id}`, completo | 200 registro | 404, 409 `CODIGO_DUPLICADO`, `PADRE_INACTIVO`, `ASIGNACIONES_AFECTADAS` (solo puestos), 415, 422 |
| `POST` | `/api/{r}/{id}/desactivar` | A | `{}` | 200 registro | 404, 409 `DEPENDENCIAS_ACTIVAS`, 415 |
| `POST` | `/api/{r}/{id}/activar` | A | `{}` | 200 registro | 404, 409 `PADRE_INACTIVO`, 415 |

`PUT` no cambia `activo`. El estado solo cambia por `desactivar` y `activar`, para que
siempre pase por las reglas de baja.

### Usuarios

| Metodo | Ruta | Quien | Recibe | Devuelve | Errores |
|---|---|---|---|---|---|
| `GET` | `/api/usuarios` | S | `q` (nombre, usuario o correo), `rol`, `activo`, `puesto_id`, `seccion_id`, paginacion | 200 listado sin `hash_contrasena` | 422 |
| `GET` | `/api/usuarios/{id}` | S | | 200 usuario con `jerarquia` (puesto, seccion, departamento, area, empresa) y servicios de los que es responsable | 404 |
| `POST` | `/api/usuarios` | A | `{nombre, usuario, correo?, contrasena, rol, puesto_id}` | 201 usuario | 409 `CODIGO_DUPLICADO` (usuario o correo), `PADRE_INACTIVO`; 415; 422 |
| `PUT` | `/api/usuarios/{id}` | A | `{nombre, usuario, correo?, rol, puesto_id}`, sin contrasena | 200 usuario | 404, 409 `CODIGO_DUPLICADO`, `PADRE_INACTIVO`, `ASIGNACIONES_AFECTADAS`, `ULTIMO_ADMINISTRADOR`; 415; 422 |
| `PUT` | `/api/usuarios/{id}/contrasena` | A | `{contrasena}` | 204; revoca las sesiones de ese usuario | 404, 415, 422 |
| `POST` | `/api/usuarios/{id}/desactivar` | A | `{}` | 200 `{usuario, advertencias}`; revoca sus sesiones | 404, 409 `ULTIMO_ADMINISTRADOR`, 415 |
| `POST` | `/api/usuarios/{id}/activar` | A | `{}` | 200 usuario | 404, 409 `PADRE_INACTIVO`, 415 |

### Catalogos

`{c}` es `clases`, `criticidades` o `tipos`.

| Metodo | Ruta | Quien | Recibe | Devuelve | Errores |
|---|---|---|---|---|---|
| `GET` | `/api/catalogos/{c}` | S | `activo` | 200 lista completa, sin paginar, ordenada por `orden`, con `valor_origen` y `origen_celda` | 404 (catalogo desconocido) |
| `POST` | `/api/catalogos/{c}` | A | `{nombre, orden}` | 201 opcion, con `valor_origen` nulo | 409 `CODIGO_DUPLICADO`, 415, 422 |
| `PUT` | `/api/catalogos/{c}/{id}` | A | `{nombre, orden}`; `valor_origen` no se acepta | 200 opcion | 404, 409, 415, 422 |
| `POST` | `/api/catalogos/{c}/{id}/desactivar` | A | `{}` | 200 opcion | 404, 409 `DEPENDENCIAS_ACTIVAS`, 415 |
| `POST` | `/api/catalogos/{c}/{id}/activar` | A | `{}` | 200 opcion | 404, 415 |
| `GET` | `/api/catalogos/mapeo` | S | | 200 opciones con `nombre <> valor_origen` (mapeo de etiquetas) | |

### Servicios

| Metodo | Ruta | Quien | Recibe | Devuelve | Errores |
|---|---|---|---|---|---|
| `GET` | `/api/servicios-n1` | S | `q`, `activo`, paginacion | 200 listado con el numero de servicios N2 de cada uno | 422 |
| `GET` | `/api/servicios-n1/{id}` | S | | 200 N1 con origen, sus N2 e incidencias de la ultima importacion | 404 |
| `POST` | `/api/servicios-n1` | A | `{codigo, nombre}` | 201 N1 | 409 `CODIGO_DUPLICADO`, 415, 422 |
| `PUT` | `/api/servicios-n1/{id}` | A | `{codigo, nombre}` | 200 N1 | 404, 409, 415, 422 |
| `POST` | `/api/servicios-n1/{id}/desactivar` | A | `{}` | 200 N1 | 404, 409 `DEPENDENCIAS_ACTIVAS`, 415 |
| `POST` | `/api/servicios-n1/{id}/activar` | A | `{}` | 200 N1 | 404, 415 |
| `GET` | `/api/servicios` | S | `q` (codigo o nombre, `ILIKE`), `n1_id`, `activo` (`S`/`N`/`DESCONOCIDO`), `clase_id`, `criticidad_id`, `tipo_id`, `seccion_id`, `requiere_revision`, paginacion. Los filtros se combinan con Y. | 200 listado con N1, catalogos y responsable resumidos | 422 (filtro con valor fuera de dominio) |
| `GET` | `/api/servicios/{id}` | S | | 200 ficha: todos los campos, N1, catalogos, seccion responsable con su jerarquia, usuario responsable (con su `activo`), origen e incidencias | 404 |
| `POST` | `/api/servicios` | A | `{servicio_n1_id, codigo, nombre, activo?, clase_id?, criticidad_id?, tipo_id?, descripcion?, metrica?, minimo?, maximo?}` | 201 ficha | 409 `CODIGO_DUPLICADO`, `PADRE_INACTIVO`; 415; 422 `VALIDACION`, `REFERENCIA_INEXISTENTE`, `MINIMO_MAYOR_QUE_MAXIMO` |
| `PUT` | `/api/servicios/{id}` | A | Los mismos campos, mas `requiere_revision`; no incluye responsable | 200 ficha | 404, 409, 415, 422 |
| `PUT` | `/api/servicios/{id}/responsable` | A | `{seccion_id, usuario_id}`, los dos admiten `null` | 200 ficha | 404, 409 `PADRE_INACTIVO`, `SERVICIO_INACTIVO`; 415; 422 `VALIDACION`, `REFERENCIA_INEXISTENTE`, `RESPONSABLE_FUERA_DE_SECCION` |
| `POST` | `/api/servicios/{id}/desactivar` | A | `{}` | 200 ficha (`activo = 'N'`) | 404, 415 |
| `POST` | `/api/servicios/{id}/activar` | A | `{}` | 200 ficha (`activo = 'S'`) | 404, 409 `PADRE_INACTIVO`, 415 |

### Importaciones

| Metodo | Ruta | Quien | Recibe | Devuelve | Errores |
|---|---|---|---|---|---|
| `POST` | `/api/importaciones` | A | `{}`; el archivo es siempre `EXCEL_PATH` | 201 importacion con resumen y `detalle`. Si la corrida fallo: 500 `INTERNO` y la corrida queda `fallida` | 409 `IMPORTACION_EN_CURSO`, 415 |
| `GET` | `/api/importaciones` | S | paginacion; orden `iniciada_en` descendente | 200 listado | 422 |
| `GET` | `/api/importaciones/{id}` | S | | 200 importacion con todas sus incidencias | 404 |

### Salud

| Metodo | Ruta | Quien | Recibe | Devuelve | Errores |
|---|---|---|---|---|---|
| `GET` | `/healthz` | P | | 200 `{"estado": "ok"}` si la base responde a `SELECT 1` | 503 `{"estado": "sin_base"}` |

### Subcomandos del binario

| Comando | Que hace | Salida |
|---|---|---|
| `catalogo servir` | Aplica las migraciones pendientes y escucha en `:8080`. | Sigue corriendo; distinto de 0 si no puede migrar. |
| `catalogo migrar` | Aplica las migraciones pendientes y termina. | 0 si quedo al dia. |
| `catalogo importar` | Corre el importador sobre `EXCEL_PATH` e imprime el resumen. | 0 completada; 1 fallida; 2 `IMPORTACION_EN_CURSO`. |
| `catalogo sembrar-demo` | [Cuentas y datos de demostracion](reglas.md#cuentas-de-evaluacion). | 0 si todo quedo creado o ya existia. |
| `catalogo salud` | Llama a `http://127.0.0.1:8080/healthz`. Lo usa el `healthcheck`, porque la imagen no tiene `curl`. | 0 si responde 200. |

Los comandos concretos de `docker compose` para cada uno los fija el `Makefile`.

## Interfaz servida desde el binario

- `web/` es una aplicacion Vite. `npm run build` deja los archivos en `web/dist/`.
- En Docker esos archivos se copian a `internal/web/dist/`, y `internal/web` los expone
  con `//go:embed all:dist` como un `fs.FS`.
- El enrutador atiende primero `/api/` y `/healthz`. Cualquier otra ruta busca el archivo
  en el `fs.FS`; si no existe, y la ruta no tiene extension, devuelve `index.html`, para
  que `react-router` resuelva rutas como `/servicios/12` al recargar. Una ruta `/api/...`
  desconocida da 404 JSON, nunca `index.html`.
- Los archivos con hash en el nombre (`/assets/*`) llevan `Cache-Control: max-age=31536000, immutable`
  e `index.html` lleva `no-cache`.
- La interfaz llama a la API con rutas relativas (`/api/...`) y `credentials: 'same-origin'`.
  No hay URL de API configurable ni CORS (D17).
- Pantallas: inicio de sesion; estructura organizacional (una lista por nivel con su
  formulario); usuarios; servicios (lista con busqueda, filtros y paginacion, y la ficha);
  catalogos; importaciones con su resumen e incidencias. Los botones de escritura se
  ocultan al rol `consulta`, pero la proteccion es la del servidor.

## Construccion en Docker

Un `Dockerfile` con tres etapas:

1. **`web`**, sobre una imagen `node` LTS: `npm ci` con `package-lock.json` y despues
   `npm run build`.
2. **`build`**, sobre una imagen `golang`: `go mod download`, copia del codigo, copia de
   `web/dist` desde la etapa 1 a `internal/web/dist`, y
   `CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/catalogo ./cmd/catalogo`.
   Esta etapa tiene la cadena de Go completa, y la usa tambien el servicio de pruebas.
3. **`final`**, sobre `gcr.io/distroless/static-debian12:nonroot`: solo el binario en
   `/app/catalogo`. Usuario sin privilegios, `EXPOSE 8080`,
   `ENTRYPOINT ["/app/catalogo"]` y `CMD ["servir"]`.

Las versiones exactas de las imagenes (`node`, `golang`) se fijan en el `Dockerfile` al
implementarlo, con etiqueta de version concreta y nunca `latest`. No se deciden aqui.

`.dockerignore` excluye `.env`, `.git`, `node_modules`, `web/dist`, `logs`, `docs`,
`data` (el Excel se monta, no se copia en la imagen) y `.claude`.

## Contenedores, volumen y healthcheck

`compose.yaml`, proyecto por defecto (el de evaluacion):

| Servicio | Imagen | Detalle |
|---|---|---|
| `db` | `postgres:17-alpine` | Variables `POSTGRES_DB`, `POSTGRES_USER` y `POSTGRES_PASSWORD` desde `.env`. Volumen `catalogo_pgdata:/var/lib/postgresql/data`. `healthcheck`: `pg_isready -U $POSTGRES_USER -d $POSTGRES_DB`, cada 5 s, `retries: 10`. Sin puerto publicado: solo la ve `app`. |
| `app` | Construida desde `Dockerfile` (etapa `final`) | `depends_on: db: condition: service_healthy` (D18). `DATABASE_URL` armada con las mismas variables. Puerto `${APP_PORT:-8080}:8080`. Volumen `./data:/app/data:ro`. `healthcheck`: `["CMD", "/app/catalogo", "salud"]`, cada 10 s, `start_period: 20s`. `restart: unless-stopped`. |

Volumen con nombre: `catalogo_pgdata`. `docker compose down` y `docker compose restart` lo
conservan, y eso es lo que comprueba P12. Solo `down -v` lo borra, y por los limites del
harness (AGENTS.md, seccion 4) eso solo se hace en el proyecto `catalogo-test`.

`compose.test.yaml`, que se usa con `-p catalogo-test` encima de `compose.yaml`:

- `db` con `POSTGRES_DB=catalogo_test`. El volumen queda como `catalogo-test_catalogo_pgdata`,
  separado del de evaluacion por el nombre de proyecto.
- `app` en otro puerto del anfitrion, para que las dos pilas convivan.
- `pruebas`, sobre la etapa `build`: corre `go test ./tests/integracion/... -count=1 -v`
  con `TEST_DATABASE_URL` hacia la base `catalogo_test`. Depende de `db` sano. Las pruebas
  de P12 se apoyan en el proyecto de prueba, no en el de evaluacion.

Variables de `.env.example`: `POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PASSWORD`,
`APP_PORT`, `SESSION_TTL`, `COOKIE_SECURE`, `EXCEL_PATH`, `DEMO_ADMIN_USUARIO`,
`DEMO_ADMIN_CONTRASENA`, `DEMO_CONSULTA_USUARIO` y `DEMO_CONSULTA_CONTRASENA`. Todas con
valores de ejemplo, no reales.

## Migraciones

- Archivos `internal/db/migraciones/NNNNN_descripcion.sql` con secciones `-- +goose Up` y
  `-- +goose Down`.
- Orden previsto: 1, organizacion; 2, usuarios y sesiones; 3, catalogos; 4, servicios;
  5, importacion e incidencias. Las FK de `importacion_id` en las tablas anteriores se
  agregan en la 5.
- Ninguna migracion inserta datos del Excel ni de demostracion: de eso se encargan
  `importar` y `sembrar-demo`.
- Las migraciones solo agregan. Al reiniciar no se recrea nada y los datos persisten (P12).
