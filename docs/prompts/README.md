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
