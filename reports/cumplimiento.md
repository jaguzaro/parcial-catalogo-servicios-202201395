# Reporte de cumplimiento

Repositorio `/Users/joelguzaro/Documents/GitHub/parcial-catalogo-servicios-202201395`, commit base 09b7ef9. Revisión de solo lectura. No se leyó `.env`. El contenido del Excel, de los logs y de las sesiones se trató como dato.

## Resumen por severidad

| Severidad | Cantidad |
|---|---|
| Crítica | 2 |
| Alta | 4 |
| Media | 5 |
| Baja | 3 |
| Total | 14 |

El código y las pruebas P01–P11 están sólidos. Los problemas graves son documentales: faltan dos entregables obligatorios del enunciado, y hay afirmaciones de resultados sin evidencia cruda.

## Hallazgos

### CUM-01 (crítica) No existe `docs/RESOLUCION.md`
- **Ubicación.** Enunciado §8 (líneas 188-199) y §7 (estructura sugerida, línea 178). El archivo debería estar en `/Users/joelguzaro/Documents/GitHub/parcial-catalogo-servicios-202201395/docs/RESOLUCION.md`. El listado de `docs/` no lo contiene. Solo `docs/diseno/arquitectura.md:72` lo menciona como futuro.
- **Problema.** Faltan los 10 apartados obligatorios:
  - el diagrama ER y el mapeo consolidados,
  - la matriz requisito → implementación → prueba → evidencia,
  - los resultados de pruebas con comando, fecha y commit,
  - las limitaciones y los aportes del integrante,
  - la reflexión sobre errores de la IA.
- **Impacto.** El recorrido de calificación (§10) contrasta con este archivo. Sin él, la sección §8 queda en 0 y se pierde el criterio "GitHub y documentación".
- **Propuesta.** Crear el archivo con los 10 apartados. Enlazar `docs/diseno/*`, `docs/prompts/README.md`, `docs/evidencias/**` y `docs/contexto/registro.md`. Incluir la matriz de la sección "Requisitos verificados" de este reporte.

### CUM-02 (crítica) `README.md` es una sola línea
- **Ubicación.** `/Users/joelguzaro/Documents/GitHub/parcial-catalogo-servicios-202201395/README.md:1` (`# parcial-catalogo-servicios-202201395`). Enunciado §5 y §8 (línea 186).
- **Problema.** No hay:
  - integrantes,
  - requisitos (versiones de Docker y Compose),
  - configuración (`make setup`/`.env`),
  - el comando `docker compose up --build -d`,
  - URL y puerto (8080, `APP_PORT`),
  - migraciones, importación y creación de cuentas (`make migrate`, `make import`, `make seed-demo`),
  - pruebas (`make check`, `make e2e`, `make test-persistence`),
  - logs, apagado normal (`make down`) frente al reinicio destructivo (`make reset-test`, `reset-dev`).
- **Impacto.** "Clonar y seguir el README" es imposible. `docs/prompts/03-esqueleto-docker.md:171` ya prometía documentar el cambio de `APP_PORT` en el README, y no está.
- **Propuesta.** Escribir el README con esos puntos. Aclarar que `docker compose up` migra solo (`cmd/catalogo/main.go:100`) pero no importa ni siembra, y que eso lo hacen `make import` y `make seed-demo`. Indicar que `make` requiere `make` en el anfitrión, o dar los equivalentes con `docker compose run --rm app importar`.

### CUM-03 (alta) Evidencia de P12 y de `make e2e` sin salida cruda en el repositorio
- **Ubicación.** `/Users/joelguzaro/Documents/GitHub/parcial-catalogo-servicios-202201395/docs/prompts/11-pruebas-finales.md:101-118, 148`.
- **Problema.**
  - Afirma `make test-persistence exit 0 (ejecutado dos veces)`, `make e2e exit 0, 2 pruebas pasadas` y "P01 a P12 estan completas".
  - La única "salida" de P12 es un extracto retocado a mano (`P12-...-61997`).
  - En `docs/evidencias/pruebas/` no hay ningún archivo con la salida de `make test-persistence` ni de `make e2e`. Una búsqueda de "P12 OK" en `docs/evidencias` sin sesiones no da resultados.
  - `docs/evidencias/README.md:9` promete "Aqui solo hay salidas reales".
  - El enunciado (§8) prohíbe inventar resultados. Esto es verificable solo dentro del JSON de sesión, no como evidencia entregada.
- **Propuesta.**
  - Guardar la salida completa de `make test-persistence` y de `make e2e` en `docs/evidencias/pruebas/`, con fecha y commit.
  - O ampliar `scripts/evidencia.sh` para que las incluya.
  - Retirar de la documentación los extractos con elisiones, o marcarlos como tales.

### CUM-04 (alta) P12 no es una prueba `TestP12_...` ni está en `make check`
- **Ubicación.** `/Users/joelguzaro/Documents/GitHub/parcial-catalogo-servicios-202201395/scripts/persistencia.sh` y `/Users/joelguzaro/Documents/GitHub/parcial-catalogo-servicios-202201395/Makefile:46-55, 74-75`. `AGENTS.md` §8: "`TestP01_...` hasta `TestP12_...`". `grep ^func Test` no encuentra `TestP12`.
- **Problema.**
  - P12 es un script bash. Funcionalmente cubre el escenario: escribe, hace `stop` y `up` sin borrar volumen, y relee.
  - Contradice la convención documentada de AGENTS.md y no entra en el único comando de aceptación (`make check`).
  - Los cuatro pasos de `check` pueden dar 0 con P12 roto.
  - Además solo comprueba la tabla `empresa`, no el flujo de la aplicación.
- **Propuesta.** Alinear la documentación (decir que P12 es `make test-persistence`) o añadir un `TestP12_` o un paso 5 en `check`. Verificar también que la app responde tras el reinicio.

### CUM-05 (alta) Evidencias de `make check` marcadas con un commit que no es el entregado
- **Ubicación.** `/Users/joelguzaro/Documents/GitHub/parcial-catalogo-servicios-202201395/docs/evidencias/pruebas/20261004-163555-0e33b6c.txt:2` y `20261004-163901-0e33b6c.txt:2`: "Commit: 0e33b6c (con cambios sin versionar)". HEAD es 09b7ef9.
- **Problema.** Ninguna corrida de `make check` corresponde al commit 09b7ef9 ni a un árbol limpio. El enunciado pide "comando, fecha, commit y resultado". Queda en el aire si el estado final pasa.
- **Propuesta.** Tras confirmar el commit final, ejecutar `make evidence` sobre el árbol limpio y enlazar ese archivo desde `RESOLUCION.md`.

### CUM-06 (alta) Prompt engineering: solo una iteración de mejora documentada (se piden dos)
- **Ubicación.** Enunciado §4.2 (línea 104). `/Users/joelguzaro/Documents/GitHub/parcial-catalogo-servicios-202201395/docs/prompts/README.md:17-31`.
- **Problema.**
  - La columna "Iteraciones" muestra **1** solo en 05 (v1 → v2).
  - 01 → 01b es una corrección de un defecto de script, con una columna que dice "0". 03 son "dos corridas" por cuota, no una mejora.
  - Falta un segundo ciclo "prompt inicial → problema → prompt revisado → resultado comprobado". Esto afecta el criterio "Prompt engineering" (10 puntos).
  - Cumple la parte de los cinco prompts, con objetivo, contexto, instrucciones, restricciones, salida esperada y criterio de aceptación.
- **Propuesta.** Documentar honestamente 01b (o `00`, con su segunda corrida) como segunda iteración con su problema y resultado. Si no lo fue, generar una iteración real. No presentar como mejora algo que no lo es.

### CUM-07 (media) Contradicción en `ciclo-01.md` sobre la naturaleza de los ejercicios
- **Ubicación.** `/Users/joelguzaro/Documents/GitHub/parcial-catalogo-servicios-202201395/docs/evidencias/harness/ciclo-01.md:13` ("Ninguno se introdujo a proposito") frente a `ciclo-01.md:113-114` (se quitó a propósito un middleware y una restricción).
- **Problema.**
  - La frase de la línea 13 refiere al ciclo principal, pero el documento se presenta como "los otros, que tambien son reales".
  - Enunciado §4.3: "Si el fallo se introduce deliberadamente… declararlo". Está declarado en 113-117, pero la afirmación general es ambigua.
  - La evidencia del fallo del ciclo principal (curl 415/200 y 204/401) no está como salida cruda. Solo se cita la sesión `05-autenticacion` y el reporte del orquestador. La "restricción de esquema quitada" (línea 114) no tiene archivo de salida.
- **Propuesta.** Reformular la línea 13 ("el ciclo principal no fue deliberado; las dos últimas filas lo fueron"). Enlazar las salidas reales de curl o de las pruebas (`docs/evidencias/pruebas/06-p03-sin-middleware.txt` existe para P03) y marcar lo que no tenga archivo.

### CUM-08 (media) `registro.md` no lista las tareas 10 y 12
- **Ubicación.** `/Users/joelguzaro/Documents/GitHub/parcial-catalogo-servicios-202201395/docs/contexto/registro.md:12-26`. Falta la fila de `10-interfaz-mantenimiento.md`. `12-auditoria.md` aún no consta.
- **Problema.** Enunciado §4.1: "indicar qué documentos se proporcionaron al asistente en cada fase y por qué". Hay un hueco para el prompt 10, que sí existe en `docs/prompts/README.md:30`.
- **Propuesta.** Añadir las filas faltantes con su contexto entregado y lo dejado fuera.

### CUM-09 (media) Resultados de prompts 10-12 y secciones "Resultado" escritas a mano sin salida enlazada
- **Ubicación.** `/Users/joelguzaro/Documents/GitHub/parcial-catalogo-servicios-202201395/docs/prompts/10-interfaz-mantenimiento.md` y `11-pruebas-finales.md:95-148`. `docs/evidencias/pruebas/README.md` indexa 5 archivos y omite `20261004-163555-0e33b6c.txt` y `20261004-163901-0e33b6c.txt` de la tabla (solo cita 163442 y 163555).
- **Problema.**
  - "Comprobado por el orquestador… contra el sistema levantado" no es verificable por un tercero salvo por el JSON de sesión.
  - `pruebas/README.md` omite 163901 en su índice.
- **Propuesta.** Completar el índice. Enlazar la salida real de cada comprobación, o marcar "verificable solo en la sesión X".

### CUM-10 (media) No hay constancia de la entrega en GitHub
- **Ubicación.** Enunciado §7 (puntos 3-4). Búsqueda de `maldanap`, `colaborador` y `parcial-v2` en `docs/` fuera del enunciado: solo una coincidencia incidental en `docs/evidencias/harness/00-test-guard.txt`.
- **Problema.**
  - Ningún documento registra que `maldanap-usac` fue invitado, el estado de la invitación, la rama de entrega, el SHA final ni la etiqueta `parcial-v2.0`.
  - Esto se confirma solo fuera del repo, y no se puede verificar desde aquí.
- **Propuesta.** Añadir una sección "Entrega" en `RESOLUCION.md` o en el README con URL, rama, SHA, etiqueta y estado de la invitación. Hacer la comprobación manualmente antes de entregar.

### CUM-11 (media) Archivos de auditoría y registro sin versionar, y `reports/` vacío
- **Ubicación.** `git status`: `docs/contexto/registro.md` modificado. Sin seguimiento: `docs/prompts/12-auditoria.md` y `docs/evidencias/sesiones/20261004-164524-12-auditoria.*`.
- **Problema.** Si el commit final se toma antes de versionarlos, las referencias de `registro.md` apuntarán a archivos ausentes. `reports/.gitkeep` sigue vacío porque el reporte aún no se guardó.
- **Propuesta.** Versionar todo antes de crear la etiqueta.

### CUM-12 (baja) Mapeo de etiquetas y listas de opciones
- **Ubicación.** Enunciado §2 (línea 32). `docs/diseno/mapeo-excel.md` y `tests/integracion/importador_test.go:108-129`.
- **Problema.** Las etiquetas se conservan idénticas, y `D12` y el test lo prueban con `nombre = valor_origen`. No hay un hallazgo de incumplimiento. Queda anotado como nota: el enunciado exige un mapeo si se corrige alguna etiqueta, y no se corrigió ninguna.
- **Propuesta.** Dejar una línea explícita en `RESOLUCION.md`: "no se corrigieron etiquetas, mapeo vacío".

### CUM-13 (baja) `e2e` y `make e2e` fuera de `make check`
- **Ubicación.** `/Users/joelguzaro/Documents/GitHub/parcial-catalogo-servicios-202201395/Makefile:80-92` y `e2e/tests/catalogo.spec.ts`.
- **Problema.** La interfaz será revisada (§6). La única cobertura de interfaz son dos pruebas e2e: login por consulta y un filtro. No hay prueba de la interfaz de administrador. Es una limitación válida, pero debe constar.
- **Propuesta.** Declararla en el apartado 10 de `RESOLUCION.md`.

### CUM-14 (baja) Explicación de tipos de prueba
- **Ubicación.** Enunciado §6 (línea 155). No hay un documento que clasifique los escenarios en unitarias, integración y extremo a extremo y diga cómo se ejecutan.
- **Problema.** Los comandos existen en AGENTS.md §9, pero la explicación pedida no está en un entregable.
- **Propuesta.** Añadir una tabla P01–P12 → tipo → comando → archivo.

## Requisitos verificados sin hallazgo

| Requisito | Estado | Evidencia |
|---|---|---|
| P01 login válido e inválido | Cumple | `/Users/joelguzaro/Documents/GitHub/parcial-catalogo-servicios-202201395/tests/integracion/auth_test.go:234-300` (por usuario, por correo, inválido) |
| P02 sin sesión, cierre, usuario inactivo | Cumple | `auth_test.go:304-435` (sin sesión, misma cookie tras logout, logout sin cuerpo, 415 con tipo no JSON, usuario desactivado, sesión vencida) |
| P03 consulta no modifica | Cumple | `tests/integracion/organizacion_test.go:204-302`. Recorre las 38 rutas de escritura registradas, exige 403 y 401 sin sesión, compara una foto de datos y lee con 200 |
| P04 jerarquía y usuario | Cumple | `organizacion_test.go:355-435` |
| P05 duplicados y referencia inexistente | Cumple | `organizacion_test.go:437-525`. Incluye código de padre distinto, referencia inexistente y mensajes |
| P06 importación 12/46 e incidencias | Cumple | `tests/integracion/importador_test.go:90-187`. Cuenta en la base, catálogos 2/5/11, incidencias por tipo, filas 42 y 67, trazabilidad, resumen y SHA del Excel |
| P07 repetir importación | Cumple | `importador_test.go:189-259`. Segunda corrida: 0 creados y 78 omitidos, mismos ids y huellas, tercera corrida sin pisar ediciones |
| P08 SE.12 y ausentes | Cumple | `importador_test.go:261-337`. B99 canónico, B100 conservado, incidencia, `DESCONOCIDO`, nulos, mínimo y máximo solo en las filas 5 y 25 |
| P09 mínimo mayor que máximo | Cumple | `tests/integracion/catalogo_test.go:120-230` (alta y edición, nulo distinto de cero) |
| P10 búsqueda y filtros | Cumple | `catalogo_test.go:243-447`. Verificación parcial por lectura, sin hallazgos |
| P11 responsable de otra sección | Cumple | `catalogo_test.go:451-529` (422 `RESPONSABLE_FUERA_DE_SECCION`, sin cambio en la base) |
| P12 persistencia | Parcial | Ver CUM-03 y CUM-04. El script `/Users/joelguzaro/Documents/GitHub/parcial-catalogo-servicios-202201395/scripts/persistencia.sh` es correcto (`stop` y `up`, mismo volumen, mismo contenido) |
| Controles 12 códigos N1 y 46 N2 | Cumple | `importador_test.go:101-106, 175-178` (`n1_distintos` 12, `celdas_n1` 13, `n2_distintos` 46, `celdas_n2` 46) |
| Importación caso 1 (celdas combinadas) | Cumple | `importador_test.go:152-161` (SE.01.01 en `A5:L7`, SE.06.01 en `A26:L29`) |
| Importación caso 2 (SE.12) | Cumple | `importador_test.go:265-284` |
| Importación caso 3 (formato de códigos) | Cumple | `importador_test.go:286-297`. Texto sin normalizar, con incidencia `CODIGO_FORMATO_NO_ESTANDAR` |
| Importación caso 4 (atributos incompletos) | Cumple | `importador_test.go:303-316` |
| Importación caso 5 (filas sin código) | Cumple | `importador_test.go:136-144`. Filas 42 y 67 sin servicio |
| Importación caso 6 (trazabilidad) | Cumple | `importador_test.go:146-150, 342-366`. Hoja, rango, huella y fusión por campo |
| Excel original intacto | Cumple | `importador_test.go:18, 92-95, 184-186` y `make check` paso 4 contra `docs/contexto/excel-sha256.txt` |
| Docker (compose, Dockerfile, `.dockerignore`, `.env.example`) | Cumple | `compose.yaml` (db con volumen nombrado, healthcheck, `depends_on: service_healthy`, `./data` en `:ro`). `Dockerfile` multi-etapa, `.env.example` solo con valores de ejemplo, `.gitignore` excluye `.env` |
| `make check` verde, 33 comprobaciones de hooks | Cumple | `docs/evidencias/pruebas/20261004-163901-0e33b6c.txt:231-234` ("33 OK, 0 fallas", "CHECK OK"). Sujeto a CUM-05 |
| Afirmación de AGENTS.md "38 rutas" y 12/46 | Cumple | Coincide con el código de P03 (20 + 5 + 13 = 38) |
| Evidencia de fallo conservado (15 pruebas rojas) | Cumple | `docs/evidencias/pruebas/20261004-163442-0e33b6c.txt:230-267`. Cuenta 15 `FAIL` y `CHECK FALLO en el paso 2/4`, y coincide con lo escrito en `pruebas/README.md:16-27` |
| Context engineering | Cumple | `AGENTS.md` con regla de datos no confiables (§3). `registro.md:30-40` con 9 versiones, 8 de ellas motivadas por hallazgo o decisión. Más de dos actualizaciones justificadas (v2, v5, v8). Sujeto a CUM-08 |
| Cinco prompts utilizados | Cumple | `docs/prompts/00-11` cubren análisis del Excel (01), modelo (02), autenticación (05), importación (07) y pruebas/Docker (03, 11). Sujeto a CUM-06 |
| Harness: límites, scripts y comandos | Cumple | `hooks/guard.sh`, `hooks/test-guard.sh`, `Makefile`, bitácora de bloqueos en `docs/evidencias/harness/` |
| Ciclo completo de corrección | Cumple, con matiz | `docs/evidencias/harness/ciclo-01.md`. Ver CUM-07 |
| Pruebas fuera de lo que no existe en repositorio | Sin hallazgos | No se detectaron secretos en `.env.example` ni credenciales fijas en `e2e/tests/catalogo.spec.ts` (se leen por variables de entorno) |

Temas sin hallazgos adicionales:
- **Autenticación local, hash y roles.** Argon2id con sal por usuario, sesión opaca con solo su SHA-256, cookie `HttpOnly`, y el hash que no sale en las respuestas. Todo está cubierto por `auth_test.go:436-490` y `organizacion_test.go:781`. No encontré incumplimientos.
- **Jerarquía, baja lógica y padres inactivos.** Cubiertos por `TestOrganizacion_BajaConDependientesActivos`, `TestOrganizacion_PadreInactivo` y `TestUsuarios_UltimoAdministrador`. Sin hallazgos.
- **Datos de demostración.** Cubiertos por `tests/integracion/demo_test.go` (idempotencia). No verifiqué por lectura que el sembrado incluya al menos tres asignaciones. AGENTS.md lo afirma y queda pendiente de comprobar en `internal/demo`.

## Afirmaciones sin respaldo directo (riesgo alto, enunciado §8)

1. "make test-persistence exit 0 (ejecutado dos veces)", "make e2e exit 0, 2 pruebas pasadas" y "P01 a P12 estan completas" (`docs/prompts/11-pruebas-finales.md:104-106, 148`). Ver CUM-03.
2. "Hubo varios" ciclos del mismo tipo (`ciclo-01.md:9, 106-114`): la fila de esquema (línea 114) y la del arranque no tienen archivo de salida enlazado. Ver CUM-07.
3. "Aqui solo hay salidas reales de comandos" (`docs/evidencias/README.md:9`) es cierto para `make check`, pero no hay salida alguna de P12 ni de e2e. Ver CUM-03.
