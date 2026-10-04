# 01b — Corrección: el script anuncia mal cuántas líneas escribió

| Campo | Valor |
|---|---|
| Fecha | 2026-10-04 |
| Herramienta | Claude Code `2.1.289` (modo headless `claude -p`) |
| Modelo | `sonnet` |
| Ejecución | `docs/evidencias/sesiones/<archivo>.json` y `.meta.txt` |
| Commit base | `28abb78` |

## Objetivo

`scripts/analisis/analizar_excel.py` termina imprimiendo que escribió 222 líneas, pero el
archivo generado tiene 512. Que el número sea correcto.

## Contexto suministrado

- `AGENTS.md` — reglas permanentes (se carga automáticamente).
- `scripts/analisis/analizar_excel.py` — el script a corregir.

## Instrucciones

1. En la línea final que imprime `Escrito … (N líneas)`, `N` sale de contar los elementos
   de una lista de bloques, no las líneas del archivo. Cuenta las líneas reales del
   contenido escrito.
2. No cambies nada más: ni el informe que genera, ni su formato, ni el resto de los
   mensajes.

## Restricciones

- El contenido de `docs/contexto/analisis-excel.md` debe quedar **byte a byte igual**. Su
  SHA-256 actual es `e5278327dfcd451122bb82ed37d282582f71f6e2314749be757d59c3eeda8a17`.
- No toques ningún otro archivo.

## Salida esperada

El script corregido y, en el reporte final, las dos cifras: lo que imprime el script y lo
que devuelve `wc -l` sobre el informe.

## Criterio de aceptación

1. Tras ejecutar el script en contenedor:

   ```bash
   docker run --rm -v "$PWD":/work -v "$PWD/data":/work/data:ro -w /work python:3.12-slim \
     sh -c "pip install --quiet openpyxl && python scripts/analisis/analizar_excel.py"
   ```

   el número que imprime coincide con `wc -l < docs/contexto/analisis-excel.md`.
2. `shasum -a 256 docs/contexto/analisis-excel.md` sigue dando `e5278327…8a17`.
3. `git status --short` no muestra más archivos modificados que el script.

---

> Lo de arriba es el texto que se envio, sin retocar y con sus acentos. El SHA-256 que
> quedo registrado en las evidencias corresponde a esa version. Lo de abajo se escribio
> despues, asi que el hash del archivo de hoy ya no coincide.

## Resultado

Aceptado, 4 turnos. El mensaje contaba los elementos de una lista de bloques. Ahora cuenta
los saltos de linea del texto que escribe.

Comprobado aparte: el script imprime 512 lineas, `wc -l` devuelve 512 y el informe sigue en
`e5278327`, sin un byte de diferencia.

El asistente agrego una precision que no se le pidio: el informe no termina en salto de
linea, asi que `wc -l` cuenta 512 y las lineas reales son 513. Dejo la cifra alineada con
`wc -l`, que era el criterio.
