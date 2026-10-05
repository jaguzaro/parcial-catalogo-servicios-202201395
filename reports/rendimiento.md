# Reporte de rendimiento

Repositorio: parcial-catalogo-servicios-202201395, commit 09b7ef9. Revisión de solo lectura.
Escala de referencia: 46 servicios de nivel 2, 12 de nivel 1, 2 clases, 5 criticidades y 11 tipos. Con ese volumen no hay ningún problema de rendimiento puro. Los hallazgos de severidad media son de robustez operativa, y los de severidad baja son de higiene que solo importaría si el catálogo creciera.

## Resumen

| Severidad | Cantidad |
|---|---|
| Crítica | 0 |
| Alta | 0 |
| Media | 3 |
| Baja | 5 |
| Total | 8 |

## Hallazgos

### REN-01 (media) Servidor HTTP sin timeouts de lectura, escritura ni inactividad
- Ubicación: `cmd/catalogo/main.go:117-121`
- Evidencia: `http.Server` solo define `ReadHeaderTimeout: 5 * time.Second`. No hay `ReadTimeout`, `WriteTimeout` ni `IdleTimeout`.
- Impacto: un cliente lento puede mantener abiertos el cuerpo de la petición o la respuesta, y las conexiones inactivas no se cierran nunca. `MaxBytesReader` limita el tamaño del cuerpo, pero no el tiempo.
- Propuesta: añadir `ReadTimeout`, `WriteTimeout` e `IdleTimeout` (por ejemplo 15 s, 30 s y 60 s).

### REN-02 (media) Login con Argon2id de 64 MiB y 4 hilos, sin límite de concurrencia ni de tasa
- Ubicación: `internal/auth/contrasena.go:20-22` (memoria 64 MiB, 3 pasadas, 4 hilos) y `:57`, `:88` (`argon2.IDKey`). Se invoca desde `internal/auth/sesion.go:105` (`verificarFicticio`, usuario inexistente) y `:111` (`VerificarContrasena`).
- Evidencia: el endpoint de login es público y cada intento reserva 64 MiB y usa 4 hilos de CPU. Incluso un usuario inexistente paga el costo completo, por el hash ficticio. No encontré semáforo, límite de tasa ni límite de intentos.
- Impacto: N logins concurrentes consumen N × 64 MiB. Unas decenas de peticiones simultáneas pueden agotar la memoria del contenedor o saturar la CPU.
- Propuesta: acotar las verificaciones Argon2 simultáneas con un semáforo (canal con buffer) y limitar la tasa de login por IP o usuario.

### REN-03 (media) Importación sin timeout y con fila `en_curso` que puede quedar huérfana
- Ubicación: `internal/importador/importador.go:83-96` (la fila `en_curso` se confirma en su propia transacción antes de ejecutar), `:98-105` (marcado como `fallida` con el mismo `ctx`). `cmd/catalogo/main.go:174` usa `context.Background()` y `:178` abre el pool sin timeout.
- Evidencia: el candado advisory se toma después de insertar la fila `en_curso`. Si el proceso muere entre ambos pasos, o si el `UPDATE ... 'fallida'` falla, la fila queda `en_curso` para siempre. Si la importación excede el contexto, el marcado de `fallida` también falla.
- Impacto: filas `en_curso` obsoletas en el historial. Ninguna consulta bloqueante las retiene, porque el candado es `xact` y se libera solo. La corrupción de datos no es posible, porque el trabajo va en una sola transacción.
- Propuesta: usar `context.WithTimeout` en el subcomando `importar`, y marcar `fallida` con un contexto nuevo (o reconciliar las `en_curso` antiguas al arrancar).

### REN-04 (baja) Tabla `sesion` sin purga y sin índice sobre `expira_en`
- Ubicación: `internal/db/migraciones/00001...` no aplica; ver `00002_organizacion.sql:98-111`, y `internal/auth/sesion.go:123-126` y `:145-149`.
- Evidencia: cada login inserta una fila. Las sesiones vencidas o revocadas nunca se borran. `Autenticar` busca por `token_hash` (cubierto por `sesion_token_uk`), así que la consulta por petición es eficiente. En cambio, no hay purga ni índice por `expira_en` o `revocada_en`.
- Impacto: crecimiento lento e indefinido de la tabla. A esta escala es irrelevante.
- Propuesta: añadir una purga periódica o al arrancar de las filas con `expira_en < now()`, con un índice parcial si crece.

### REN-05 (baja) Importación con consultas por fila (patrón N+1 acotado)
- Ubicación: `internal/importador/importador.go:268-273` y `:273-275` (un SELECT por opción de catálogo, y un EXISTS adicional si no existe), `:430` (SELECT del nivel 1 por cada n2 cuando no está en el mapa), `:537-538` (un SELECT por servicio) y `:694-739` (un INSERT por incidencia).
- Evidencia: bucles `for _, s := range plan.N2` con consultas dentro de `servicio()`. Son unas 46 filas de nivel 2, 12 de nivel 1, 18 opciones de catálogo y unas 10 incidencias, es decir, unos cientos de viajes de ida y vuelta dentro de una sola transacción.
- Impacto: despreciable a esta escala (milisegundos en red local). Escalaría linealmente si el Excel creciera a miles de filas.
- Propuesta: precargar los servicios existentes por código con un solo `SELECT` y usar `pgx.Batch` o `CopyFrom` para las incidencias, solo si el volumen crece.

### REN-06 (baja) Búsqueda `ILIKE '%q%'` sin índice trigram y doble consulta count + página
- Ubicación: `internal/catalogo/n2.go:220` y `:246-251`. El mismo patrón está en `internal/catalogo/n1.go:155`, `internal/usuarios/usuarios.go:154` y `internal/organizacion/organizacion.go:160`.
- Evidencia: el patrón `%...%` no puede usar los btree `servicio_n2_nombre_idx` (`00004_catalogo.sql:129`) ni `servicio_n2_codigo_uk`. Se hace un `count(*)` y luego un `SELECT` con `LIMIT/OFFSET`.
- Impacto: recorrido secuencial sobre 46 filas, irrelevante. El índice `servicio_n2_nombre_idx` es en la práctica inútil para el filtro de contiene.
- Propuesta: si el catálogo crece, usar `pg_trgm` con índice GIN, o `count(*) OVER()` para ahorrar la segunda consulta.

### REN-07 (baja) Subconsulta de incidencias con `max(id)` sobre `importacion` sin índice por `estado`
- Ubicación: `internal/catalogo/n1.go:54`
- Evidencia: `importacion_id = (SELECT max(id) FROM importacion WHERE estado = 'completada')` se ejecuta en cada ficha de n1 y n2. La tabla `importacion` solo tiene índice por `usuario_id` (`00003_importacion.sql:31`). La subconsulta se resuelve con un recorrido hacia atrás del PK, sin problema con pocas corridas.
- Impacto: despreciable. Crece con el número de importaciones, que es pequeño.
- Propuesta: opcional, un índice parcial `(id) WHERE estado = 'completada'` si el historial crece.

### REN-08 (baja) Listados sin paginación y `pagina` sin tope superior
- Ubicación: `internal/catalogo/opciones.go:112-133` (`ListarOpciones`, sin paginar) y `:136-157` (`Mapeo`, un Query por catálogo, 3 en total). `internal/http/parametros.go:21-24` (`pagina` sin máximo).
- Evidencia: los catálogos de opciones están acotados (2, 5 y 11 filas más lo que cree el administrador). `Mapeo` abre 3 consultas, una por catálogo, no por fila. Un `pagina=999999999` genera un `OFFSET` enorme que no devuelve nada pero es barato.
- Impacto: despreciable. `tamano` sí está acotado a 100 (`parametros.go:12-13`).
- Propuesta: opcionalmente, acotar `pagina` o devolver vacío cuando `OFFSET >= total`.

## Áreas revisadas sin hallazgos

- **N+1 en listados de la API.** El listado de n2 (`n2.go:115-132`) resuelve clase, criticidad, tipo, sección, departamento, área, empresa y usuario responsable con JOIN en una sola consulta. El conteo de hijos de n1 (`n1.go:115`) es una subconsulta correlacionada cubierta por `servicio_n2_n1_idx`. La jerarquía de ancestros se lee con una sola consulta de JOIN (`organizacion.go:208-230`). No hay consultas dentro de bucles en los paths HTTP.
- **Paginación.** Todos los GET de colección de n1, n2, organización, usuarios e importaciones usan `LIMIT/OFFSET` con `tamano` máximo 100 (`parametros.go:11-34`, `db/listado.go`). Las dependencias en bajas se acotan con `db.MaxDependencias` (`organizacion.go:502`, `catalogo.go:236`).
- **Índices sobre filtros y claves foráneas del catálogo.** `servicio_n2` tiene índices en `servicio_n1_id`, `nombre`, `activo`, `clase_id`, `criticidad_id`, `tipo_id`, `seccion_responsable_id`, `usuario_responsable_id` e `importacion_id`, más `codigo` único (`00004_catalogo.sql:128-136`). `servicio_n1` tiene `nombre`, `activo` e `importacion_id`, más `codigo` único. `incidencia` tiene índices en `importacion_id`, `servicio_n1_id` y `servicio_n2_id`. Las claves foráneas de organización quedan cubiertas por las restricciones `UNIQUE (padre_id, codigo)`, y `usuario.puesto_id` y `sesion.usuario_id` tienen índice. Los índices de `activo`, `clase`, `criticidad` y `tipo` son de baja selectividad pero inocuos.
- **Importador, transacciones e idempotencia.** Todo el trabajo va en una sola transacción con `defer tx.Rollback` (`importador.go:124-128`, `:195`), más un candado `pg_try_advisory_xact_lock` para una corrida a la vez (`:131-136`). Es idempotente por llave natural `codigo` con huella `origen_hash` (`:537-574`): si el hash coincide cuenta `SinCambios`, y la segunda corrida no crea nada. Los códigos de catálogo usan `valor_origen` único.
- **Lecturas repetidas del archivo.** El Excel se lee una sola vez con `io.ReadAll` (`importador.go:110-117`) y se pasa como `[]byte` a `Analizar` y al SHA-256. `excelize` cierra con `defer f.Close()` (`lectura.go:159`).
- **Fugas de recursos.** Todos los `Query` revisados cierran las filas con `defer filas.Close()` o con cierre explícito en cada rama (`n1.go:196-209`, `opciones.go:143-151`, `organizacion.go:507`, `usuarios.go:211`, `importador/consulta.go:97`, `:130`). `EnTx` y el importador hacen `Rollback` diferido (`db/errores.go:80`). El pool se cierra con `defer pool.Close()` (`main.go:107`, `:159`, `:182`).
- **Goroutines.** Solo existe la goroutine de `ListenAndServe` (`main.go:123`), con canal con buffer 1 y apagado ordenado con `Shutdown` y timeout de 10 s (`:132-136`). No hay otras goroutines ni `time.Sleep`.
- **Bloqueos de base y arranque.** `/healthz` y el healthcheck del contenedor tienen timeouts (`main.go:192` con `Timeout: 3 * time.Second`, `compose.yaml:10-14` y `:37-41`). `depends_on` espera `service_healthy` (`compose.yaml:21-23`), así que el arranque no depende de reintentos en el código. El bloqueo `FOR UPDATE` / `FOR SHARE` sobre filas individuales es corto y sigue un orden consistente (padre antes que hijo).
- **Límite del cuerpo de las peticiones.** `leerJSON` usa `http.MaxBytesReader` (`internal/http/auth.go:38-39`).
