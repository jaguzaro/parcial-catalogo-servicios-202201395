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

## Actualizaciones de `AGENTS.md`

| Version | Fecha | Que la motiva | Secciones | Commit |
|---|---|---|---|---|
| v1 | 2026-10-03 | Contexto inicial | Todas | `28abb78` |
| v2 | 2026-10-04 | El analisis del archivo. `A99` y `A100` resultaron ser celdas sueltas, lo que deja sin padre a `SE.12.3`, asi que habia que fijar las reglas de lectura antes de escribir el importador | 6 | |
