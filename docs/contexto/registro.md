# Registro de contexto

Que contexto recibio el asistente en cada tarea y por que. Las actualizaciones de
`AGENTS.md` apuntan al hallazgo o la decision que las motivo.

El criterio es siempre el mismo: `AGENTS.md` se carga solo, y ademas se entregan unicamente
los archivos que esa tarea necesita leer. Lo que se deja fuera tambien se anota, porque dar
de mas invita a trabajar fuera de lo que se pidio.

## Contexto entregado

| Tarea | Prompt | Archivos entregados | Por que | Dejado fuera a proposito |
|---|---|---|---|---|
| Revisar los limites del harness | [`00-verificacion-harness.md`](../prompts/00-verificacion-harness.md) | `AGENTS.md` (automatico), `hooks/guard.sh`, `docs/contexto/excel-sha256.txt` | La tarea es probar los limites, no construir. Basta la seccion 4 de `AGENTS.md`, el script que los aplica y el hash esperado para el control positivo. | El enunciado completo. El asistente no necesita el alcance funcional para esto, y conocerlo invita a adelantar trabajo durante una verificacion. |
| Analizar el archivo original | [`01-analisis-excel.md`](../prompts/01-analisis-excel.md) | `AGENTS.md` (automatico), `docs/contexto/enunciado.md` solo secciones 2 y 3.4, `data/CatalogoServicios.xlsx` | La seccion 2 describe las columnas y las listas de opciones, y la 3.4 los casos de calidad de datos que hay que localizar. Sin ellas el informe no sabria que buscar ni contra que comparar. | El resto del enunciado. Autenticacion, Docker o pruebas no ayudan a leer una hoja de calculo y si invitan a empezar a construir antes de tiempo. |
| Corregir el conteo de lineas del script | [`01b-correccion-conteo-lineas.md`](../prompts/01b-correccion-conteo-lineas.md) | `AGENTS.md` (automatico), `scripts/analisis/analizar_excel.py` | Un defecto de una linea en un mensaje de consola. Dar el informe o el enunciado solo habria invitado a reescribir cosas que ya funcionaban. | Todo lo demas, incluido el informe que genera el script. |
| Disenar el modelo y la arquitectura | [`02-diseno-modelo.md`](../prompts/02-diseno-modelo.md) | `AGENTS.md` (automatico), `docs/contexto/enunciado.md` completo, `docs/contexto/analisis-excel.md`, `docs/contexto/decisiones.md` | Es la primera tarea que necesita el enunciado entero, porque el diseno tiene que cubrir requisitos, pruebas y Docker a la vez. El informe del Excel evita que invente la forma de los datos, y el registro de decisiones evita que vuelva a elegir stack o reglas de importacion ya resueltas. | El Excel. Ya esta leido y resumido, y abrirlo otra vez solo gastaria turnos. |
| Esqueleto, Docker y rutina de validacion | [`03-esqueleto-docker.md`](../prompts/03-esqueleto-docker.md) | `AGENTS.md` (automatico), `docs/diseno/arquitectura.md`, `docs/contexto/enunciado.md` solo secciones 5 y 6 | Arquitectura trae lo que hay que construir, y las secciones 5 y 6 del enunciado fijan los requisitos de Docker y de pruebas que el resultado tiene que cumplir. | El modelo de datos y el mapeo del Excel. El esquema es otra tarea, y tenerlo delante invita a adelantarlo y quedarse sin turnos antes de que el sistema arranque. |
| Esquema completo como migraciones | [`04-esquema-migraciones.md`](../prompts/04-esquema-migraciones.md) | `AGENTS.md` (automatico), `docs/diseno/modelo.md`, `docs/diseno/reglas.md` | El diccionario es la fuente literal del esquema, y las reglas de negocio dicen cuales conviene que haga cumplir la base y no solo el codigo. | El enunciado y el resto del diseno. Lo que el enunciado pide ya esta recogido en el diccionario, y darlo otra vez solo invita a adelantar handlers o importador. |
| Autenticacion, sesiones y cuentas de evaluacion | [`05-autenticacion.md`](../prompts/05-autenticacion.md) | `AGENTS.md` (automatico), `docs/diseno/reglas.md`, `docs/diseno/arquitectura.md`, `docs/contexto/enunciado.md` solo secciones 3.1 y 6 | Reglas trae los parametros de hash y sesion ya decididos, arquitectura los endpoints y el formato de error, y del enunciado hacen falta lo que se exige de autenticacion y los escenarios a automatizar. | El modelo y el mapeo del Excel, y el resto del enunciado. El catalogo y el importador no entran aqui, y tenerlos delante invita a adelantarlos. |
| Corregir el cierre de sesion | [`05-autenticacion.v2.md`](../prompts/05-autenticacion.v2.md) | El mismo contexto de la v1, con el problema observado escrito en la cabecera del prompt | Era una correccion de un caso concreto, no un trabajo nuevo. Decirle que fallaba y por que evita que vuelva a cumplir el criterio de la misma forma que escondia el defecto. | Nada mas: quitar contexto a mitad de una tarea hace que reescriba lo que ya funcionaba. |

## Actualizaciones de `AGENTS.md`

| Version | Fecha | Que la motiva | Secciones | Commit |
|---|---|---|---|---|
| v1 | 2026-10-03 | Contexto inicial | Todas | `28abb78` |
| v2 | 2026-10-04 | El analisis del archivo. `A99` y `A100` resultaron ser celdas sueltas, lo que deja sin padre a `SE.12.3`, asi que habia que fijar las reglas de lectura antes de escribir el importador | 6 | `e4b96a6` |
| v3 | 2026-10-04 | El diseno ya resuelto. Las reglas que se aplican en el servidor y la arquitectura tenian que estar en el contexto antes de escribir migraciones y codigo | 7 y 8 |`b099f6c` |
| v4 | 2026-10-04 | Ya existe `make check`. Pasa a ser el comando con el que se comprueba cualquier cambio, asi que tenia que estar en el contexto | 9 | `2cc8639` |
| v5 | 2026-10-04 | El guardian bloqueo escribir una migracion por el `DROP TABLE` de su seccion de bajada. Habia que distinguir escribir de ejecutar en el contexto | 4 | `9a90098` |
| v6 | 2026-10-04 | Correccion de un dato desactualizado: la seccion 9 decia que `make seed-demo` estaba pendiente y ya existe | 9 | |
