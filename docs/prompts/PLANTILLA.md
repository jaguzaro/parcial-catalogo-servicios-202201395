# NN — Título del prompt

| Campo | Valor |
|---|---|
| Fecha | |
| Herramienta | Claude Code `<versión>` (modo headless `claude -p`) |
| Modelo | |
| Ejecución | `docs/evidencias/sesiones/<archivo>.json` y `.meta.txt` |
| Commit base | |

## Objetivo
Una o dos frases: qué debe existir al terminar.

## Contexto suministrado
Los archivos que el asistente debe leer, y solo esos, con el motivo de cada uno:
- `AGENTS.md` — reglas permanentes (se carga automáticamente).
- `ruta/archivo` — motivo.

## Instrucciones
Pasos concretos y verificables.

## Restricciones
Qué no hacer: alcance, archivos intocables, tecnologías fuera de lo ya decidido.

## Salida esperada
Archivos, comandos y formato del reporte final.

## Criterio de aceptación
Comprobaciones objetivas, con comando y resultado esperado (`make check` → exit 0).
Que el criterio se pueda ejecutar; "funciona bien" no es un criterio.

---

> Lo de arriba es el texto que se envía. Lo de abajo se escribe después y cambia el hash
> del archivo respecto al que quedó registrado al ejecutarlo.

## Resultado
Aceptado o rechazado, con la evidencia: comando, código de salida, archivo.

## Iteración (solo si hubo)
- **Problema observado:** qué falló y cómo se detectó (comando y salida).
- **Cambio en el prompt:** qué se precisó y por qué. Versión revisada en `NN-titulo.v2.md`.
- **Resultado comprobado:** comando y salida después del cambio.
