# NN - Titulo del prompt

| Campo | Valor |
|---|---|
| Fecha | |
| Herramienta | Claude Code `<version>`, modo headless `claude -p` |
| Modelo | |
| Ejecucion | `docs/evidencias/sesiones/<archivo>.json` y `.meta.txt` |
| Commit base | |

## Objetivo
Una o dos frases: que debe existir al terminar.

## Contexto suministrado
Los archivos que el asistente debe leer, y solo esos, con el motivo de cada uno:
- `AGENTS.md`, reglas permanentes, se carga solo.
- `ruta/archivo`, motivo.

## Instrucciones
Pasos concretos y verificables.

## Restricciones
Que no hacer: alcance, archivos intocables, tecnologias fuera de lo ya decidido.

## Salida esperada
Archivos, comandos y formato del reporte final.

## Criterio de aceptacion
Comprobaciones objetivas, con comando y resultado esperado, por ejemplo `make check` y
codigo 0. Que se pueda ejecutar. "Funciona bien" no es un criterio.

---

> Lo de arriba es el texto que se envia. Lo de abajo se escribe despues y cambia el hash
> del archivo respecto al que quedo registrado al ejecutarlo.

## Resultado
Aceptado o rechazado, con la evidencia: comando, codigo de salida, archivo.

## Iteracion (solo si hubo)
- Problema observado: que fallo y como se detecto, con comando y salida.
- Cambio en el prompt: que se preciso y por que. La version revisada va en `NN-titulo.v2.md`.
- Resultado comprobado: comando y salida despues del cambio.
