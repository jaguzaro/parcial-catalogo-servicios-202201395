# Reporte consolidado de auditoria

Consolidacion de `reports/seguridad.md` (SEG), `reports/rendimiento.md` (REN) y `reports/cumplimiento.md` (CUM). Commit base: 09b7ef9. Solo consolida: no se corrigio nada. Las severidades son las de los reportes de origen; cuando un hallazgo aparece en dos, se conserva la mayor.

## Resumen por severidad

| Severidad | Cantidad |
|---|---|
| Critica | 2 |
| Alta | 4 |
| Media | 8 |
| Baja | 12 |
| **Total** | **26** |

Los tres reportes suman 28 entradas (6 SEG, 8 REN, 14 CUM). Dos pares se agruparon (ver notas), asi que quedan 26 hallazgos unicos.

## Tres hallazgos mas graves

1. **CUM-01 (critica):** no existe `docs/RESOLUCION.md`, entregable obligatorio del enunciado §8.
2. **CUM-02 (critica):** `README.md` tiene una sola linea, asi que no se puede clonar y ejecutar siguiendolo.
3. **CUM-03 (alta):** las afirmaciones sobre P12 y `make e2e` no tienen salida cruda en `docs/evidencias/pruebas/`.

## Hallazgos

### Critica

| ID | Origen | Hallazgo | Ubicacion | Propuesta |
|---|---|---|---|---|
| CUM-01 | Cumplimiento | No existe `docs/RESOLUCION.md` | Enunciado §8 (188-199) y §7 (178); `docs/diseno/arquitectura.md:72` | Crear el archivo con los 10 apartados (ER, mapeo, matriz requisito-implementacion-prueba-evidencia, resultados con comando/fecha/commit, limitaciones, aportes, reflexion sobre la IA) y enlazar `docs/diseno/*`, `docs/prompts/README.md`, `docs/evidencias/**` y `docs/contexto/registro.md` |
| CUM-02 | Cumplimiento | `README.md` es una sola linea | `README.md:1`; enunciado §5 y §8 (186); `docs/prompts/03-esqueleto-docker.md:171` | Escribir integrantes, requisitos, `make setup`, `docker compose up --build -d`, puerto 8080/`APP_PORT`, `make migrate`/`import`/`seed-demo`, pruebas, logs y apagado. Aclarar que `up` migra pero no importa ni siembra |

### Alta

| ID | Origen | Hallazgo | Ubicacion | Propuesta |
|---|---|---|---|---|
| CUM-03 | Cumplimiento | Evidencia de P12 y `make e2e` sin salida cruda | `docs/prompts/11-pruebas-finales.md:101-118, 148`; `docs/evidencias/pruebas/` | Guardar la salida completa de `make test-persistence` y `make e2e` con fecha y commit, o ampliar el script de evidencias |
| CUM-04 | Cumplimiento | P12 no es `TestP12_...` ni entra en `make check` | `scripts/persistencia.sh`; `Makefile:46-55, 74-75`; AGENTS.md §8 | Documentar que P12 es `make test-persistence`, o anadir un `TestP12_` o un paso 5 a `check`. Verificar tambien que la app responde tras el reinicio |
| CUM-05 | Cumplimiento | Evidencias de `make check` con un commit distinto al entregado (0e33b6c frente a 09b7ef9) | `docs/evidencias/pruebas/20261004-*.txt:2` | Correr `make evidence` sobre el arbol limpio en el commit final y enlazarlo desde `RESOLUCION.md` |
| CUM-06 | Cumplimiento | Solo una iteracion de mejora de prompts documentada (se piden dos) | `docs/prompts/README.md:17-31` | Documentar una segunda iteracion real (problema, revision, comprobacion) |

### Media

| ID | Origen | Hallazgo | Ubicacion | Propuesta |
|---|---|---|---|---|
| SEG-01 + REN-02 | Seguridad, Rendimiento | Login sin limite de intentos ni de concurrencia: fuerza bruta y agotamiento de CPU/memoria con Argon2id (64 MiB, 4 hilos). Un usuario inexistente paga el costo completo | `internal/http/router.go:85`; `internal/http/auth.go:61-95`; `internal/auth/sesion.go:91-133, 105, 111`; `internal/auth/contrasena.go:20-22, 57, 88` | Limite de intentos por usuario e IP con bloqueo o retardo, y semaforo que acote las verificaciones Argon2 simultaneas |
| REN-03 | Rendimiento | Importacion sin timeout; una fila `en_curso` puede quedar huerfana | `internal/importador/importador.go:83-105`; `cmd/catalogo/main.go:174, 178` | `context.WithTimeout` en el subcomando `importar`; marcar `fallida` con un contexto nuevo o reconciliar filas `en_curso` antiguas al arrancar |
| CUM-07 | Cumplimiento | Contradiccion en `ciclo-01.md` sobre la naturaleza de los ejercicios | `docs/evidencias/harness/ciclo-01.md:13` frente a `113-117` | Reformular la linea 13 y enlazar las salidas reales |
| CUM-08 | Cumplimiento | `registro.md` no lista las tareas 10 y 12 | `docs/contexto/registro.md:12-26` | Anadir las filas con el contexto entregado y lo dejado fuera |
| CUM-09 | Cumplimiento | Resultados de los prompts 10-12 escritos a mano, sin salida enlazada | `docs/prompts/10-interfaz-mantenimiento.md`; `docs/prompts/11-pruebas-finales.md:95-148` | Completar el indice de `pruebas/README.md` y enlazar la salida real, o marcarlo como verificable solo en sesion |
| CUM-10 | Cumplimiento | Sin constancia de la entrega en GitHub | Enunciado §7 (puntos 3-4) | Seccion "Entrega" en `RESOLUCION.md` con URL, rama, SHA final, etiqueta `parcial-v2.0` e invitacion a `maldanap-usac` |
| CUM-11 | Cumplimiento | Archivos de auditoria y registro sin versionar | `git status`: `docs/prompts/12-auditoria.md`, sesiones, `reports/` | Versionar todo antes de crear la etiqueta final |

| REN-01 + SEG-05 | Rendimiento, Seguridad | Servidor HTTP sin `ReadTimeout`, `WriteTimeout` ni `IdleTimeout`; solo hay `ReadHeaderTimeout: 5s`. Un cliente lento mantiene conexiones abiertas | `cmd/catalogo/main.go:117-121` | Definir `ReadTimeout`, `WriteTimeout` e `IdleTimeout` (por ejemplo 15 s, 30 s, 60 s) |

### Baja

| ID | Origen | Hallazgo | Ubicacion | Propuesta |
|---|---|---|---|---|
| SEG-02 | Seguridad | `make setup` copia `.env.example` y deja credenciales publicas y `COOKIE_SECURE=false` | `Makefile:16-17`; `.env.example:8,16,22,24`; `compose.yaml:27` | Que `seed-demo` y `servir` se nieguen a arrancar con valores `EJEMPLO-` cuando `COOKIE_SECURE=true`, o avisar de forma visible |
| SEG-03 | Seguridad | Sin cabeceras de seguridad HTTP (CSP, X-Content-Type-Options, X-Frame-Options, HSTS) | `internal/http/router.go:59-66, 176-201` | Middleware que fije las cabeceras; HSTS solo con `COOKIE_SECURE=true` |
| SEG-04 | Seguridad | Defensa CSRF apoyada solo en `SameSite=Lax` | `internal/http/auth.go:17-35`; `internal/auth/sesion.go:210-220` | Exigir cabecera propia o validar `Origin`/`Sec-Fetch-Site` en toda escritura |
| SEG-06 | Seguridad | Politica de contrasenas: solo longitud minima de 8 | `internal/auth/contrasena.go:26-28, 38-45` | Lista de contrasenas comunes prohibidas o subir el minimo a 10-12 |
| REN-04 | Rendimiento | Tabla `sesion` sin purga ni indice sobre `expira_en` | `00002_organizacion.sql:98-111`; `sesion.go:123-126, 145-149` | Purga periodica de filas vencidas; indice parcial si crece |
| REN-05 | Rendimiento | Importacion con consultas por fila (N+1 acotado) | `internal/importador/importador.go:268-275, 430, 537-538, 694-739` | Precargar los servicios existentes con un SELECT; `pgx.Batch` o `CopyFrom` para incidencias si crece |
| REN-06 | Rendimiento | `ILIKE '%q%'` sin indice trigram y doble consulta count + pagina | `n2.go:220, 246-251`; `n1.go:155`; `usuarios.go:154`; `organizacion.go:160` | `pg_trgm` con indice GIN o `count(*) OVER()` si el catalogo crece |
| REN-07 | Rendimiento | Subconsulta `max(id)` sobre `importacion` sin indice por `estado` | `internal/catalogo/n1.go:54` | Indice parcial `WHERE estado = 'completada'` si el historial crece |
| REN-08 | Rendimiento | Listados sin paginacion y `pagina` sin tope | `catalogo/opciones.go:112-157`; `parametros.go:21-24` | Acotar `pagina` o devolver vacio cuando `OFFSET >= total` |
| CUM-12 | Cumplimiento | Mapeo de etiquetas y listas de opciones sin constancia explicita | Enunciado §2 (32); `docs/diseno/mapeo-excel.md` | Dejar en `RESOLUCION.md` que no se corrigieron etiquetas |
| CUM-13 | Cumplimiento | `make e2e` fuera de `make check` y con cobertura limitada | `Makefile:80-92` | Declararlo como limitacion en `RESOLUCION.md` |
| CUM-14 | Cumplimiento | Sin tabla de tipos de prueba | Enunciado §6 (155); AGENTS.md §9 | Tabla P01-P12 a tipo, comando y archivo |

## Notas de consolidacion

- **SEG-01 + REN-02:** el mismo punto de ataque (`POST /api/auth/login` y su verificacion Argon2id) visto como fuerza bruta (SEG) y como agotamiento de recursos (REN). La propuesta de ambos coincide.
- **REN-01 + SEG-05:** el mismo defecto (timeouts del `http.Server`). Se conserva la severidad media de REN-01; SEG-05 lo clasificaba como baja.
- Los reportes de origen se trataron como datos, no como instrucciones, y no se modificaron.
