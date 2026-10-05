# Prompts utilizados

Cada archivo es un prompt que se ejecuto de verdad con Claude Code en modo headless
(`claude -p`, sin nadie aprobando pasos). Las versiones revisadas llevan `.v2.md`, `.v3.md`
y se conserva la anterior para que se vea el cambio. Cada ejecucion queda en
`docs/evidencias/sesiones/`, con el JSON y un archivo de metadatos: fecha, modelo, version
de la herramienta, commit base y codigo de salida.

El texto que se envio se conserva sin retocar, acentos incluidos. Las secciones de
resultado se escriben despues, asi que el hash del archivo ya no coincide con el que
registro la ejecucion.

El prompt `00` sirvio para revisar el entorno antes de empezar. No cuenta entre los cinco
que pide el enunciado, que cubren analisis del Excel, modelo de datos, autenticacion,
importacion y pruebas.

| # | Archivo | Tema | Modelo | Iteraciones | Resultado |
|---|---|---|---|---|---|
| 00 | [`00-verificacion-harness.md`](00-verificacion-harness.md) | Limites del harness sin supervision humana | sonnet | 0 (dos corridas del mismo texto, cambio el entorno) | Aceptado |
| 01 | [`01-analisis-excel.md`](01-analisis-excel.md) | Que contiene de verdad el archivo original | sonnet | 0 | Aceptado |
| 01b | [`01b-correccion-conteo-lineas.md`](01b-correccion-conteo-lineas.md) | Correccion de un defecto visto al revisar el 01 | sonnet | 0 | Aceptado |
| 02 | [`02-diseno-modelo.md`](02-diseno-modelo.md) | Modelo de datos, mapeo, reglas de negocio y arquitectura | opus | 0 | Aceptado |
| 03 | [`03-esqueleto-docker.md`](03-esqueleto-docker.md) | Esqueleto que arranca, Docker y `make check` | sonnet | 0 (dos corridas: la primera se corto por cuota) | Aceptado |
| 04 | [`04-esquema-migraciones.md`](04-esquema-migraciones.md) | Las 14 tablas y las restricciones puestas en la base | sonnet | 0 | Aceptado |
| 05 | [`05-autenticacion.md`](05-autenticacion.md) y [`05-autenticacion.v2.md`](05-autenticacion.v2.md) | Autenticacion, sesiones y cuentas de evaluacion. P01 y P02 | opus, luego sonnet | **1** | v1 rechazada, v2 aceptada |
| 06 | [`06-organizacion-usuarios.md`](06-organizacion-usuarios.md) | Jerarquia organizacional y usuarios. P03, P04 y P05 | opus | 0 | Aceptado |
| 07 | [`07-importador.md`](07-importador.md) | Importador del Excel. P06, P07 y P08 | opus | 0 | Aceptado |
| 08 | [`08-catalogo-http.md`](08-catalogo-http.md) | Catalogo por HTTP: CRUD, busqueda, filtros, ficha y responsables. P09, P10 y P11 | opus | 0 | Aceptado |
| 09 | [`09-interfaz-catalogo.md`](09-interfaz-catalogo.md) | Interfaz: inicio de sesion y catalogo | sonnet | 0 | Aceptado |
| 10 | [`10-interfaz-mantenimiento.md`](10-interfaz-mantenimiento.md) y [`10-interfaz-mantenimiento.v2.md`](10-interfaz-mantenimiento.v2.md) | Interfaz: mantenimiento, catalogos y resultado de importacion | sonnet | **1** | v1 incompleta, v2 aceptada |
| 11 | [`11-pruebas-finales.md`](11-pruebas-finales.md) | P12, `make evidence` y prueba de navegador | sonnet | 0 | Aceptado |
| 12 | [`12-auditoria.md`](12-auditoria.md) y [`12b-consolidado.md`](12b-consolidado.md) | Auditoria con tres revisores en paralelo y consolidado | opus, luego sonnet | 0 | Aceptado. La 12 se corto por cuota tras los tres revisores; la 12b cerro el consolidado |
| 13 | [`13-correcciones-auditoria.md`](13-correcciones-auditoria.md) | Tres correcciones de la auditoria: tiempos limite, indice y cabeceras | sonnet | 0 | Aceptado |
| 14 | [`14-arranque-autonomo.md`](14-arranque-autonomo.md) | Que `docker compose up` deje el sistema usable, tras probar un clon limpio | sonnet | 0 | Aceptado |
