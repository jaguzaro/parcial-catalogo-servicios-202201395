# 12b - Consolidado de la auditoria

| Campo | Valor |
|---|---|
| Fecha | 2026-10-04 |
| Herramienta | Claude Code `2.1.289`, modo headless `claude -p` |
| Modelo | `sonnet` (el `consolidador` usa haiku, el de su definicion) |
| Ejecucion | `docs/evidencias/sesiones/<archivo>.json` y `.meta.txt` |
| Commit base | `09b7ef9` |

## Objetivo

La tarea 12 lanzo los tres revisores en paralelo y los tres terminaron, pero la sesion se
corto por limite de uso antes de invocar al consolidador. Falta solo eso.

## Contexto suministrado

- `AGENTS.md`, reglas permanentes, se carga solo.
- Los tres reportes ya escritos: `reports/seguridad.md`, `reports/rendimiento.md` y
  `reports/cumplimiento.md`.
- La definicion de `consolidador` en `.claude/agents/`.

## Instrucciones

1. Invoca al subagente `consolidador`.
2. Produce `reports/consolidado.md` con los hallazgos de los tres reportes en una sola
   lista priorizada.
3. Agrupa lo que aparezca en mas de un reporte en un solo hallazgo, citando de cuales
   viene. No lo repitas tres veces.
4. Cada entrada conserva su ubicacion, su severidad y su propuesta.

## Restricciones

- **No corrijas nada.** Solo consolidar.
- No modifiques los tres reportes de entrada.
- No toques nada fuera de `reports/consolidado.md`.

## Salida esperada

`reports/consolidado.md` y un reporte final con el conteo por severidad y los tres
hallazgos mas graves.

## Criterio de aceptacion

1. Existe `reports/consolidado.md`.
2. Los hallazgos que aparecen en dos reportes estan agrupados en uno solo.
3. `git status` no muestra cambios fuera de `reports/consolidado.md`.

---

> Lo de arriba es el texto que se envio, sin retocar. Lo de abajo se escribio despues.

## Resultado

Pendiente de ejecucion.
