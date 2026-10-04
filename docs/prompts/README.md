# Prompts utilizados

Cada archivo es un prompt realmente ejecutado con Claude Code en modo headless
(`claude -p`, sin nadie aprobando pasos). Las versiones revisadas llevan sufijo `.v2.md`,
`.v3.md`… y se conserva la anterior para que se vea el cambio. Cada ejecución queda en
`docs/evidencias/sesiones/`, con el JSON y un archivo de metadatos: fecha, modelo, versión
de la herramienta, commit base y código de salida.

El texto enviado se conserva sin retocar. Las secciones de resultado se escriben después,
así que el hash del archivo ya no coincide con el que registró la ejecución.

El prompt `00` sirvió para verificar el entorno antes de empezar; no cuenta entre los cinco
que pide el enunciado, que cubren análisis del Excel, modelo de datos, autenticación,
importación y pruebas.

| # | Archivo | Tema | Modelo | Iteraciones | Resultado |
|---|---|---|---|---|---|
| 00 | [`00-verificacion-harness.md`](00-verificacion-harness.md) | Límites del harness sin supervisión humana | sonnet | 0 (2 ejecuciones del mismo texto; cambió el entorno) | Aceptado |
