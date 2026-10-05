# 12 - Auditoria con subagentes en paralelo

| Campo | Valor |
|---|---|
| Fecha | 2026-10-04 |
| Herramienta | Claude Code `2.1.289`, modo headless `claude -p` |
| Modelo | `opus` (coordinador); los subagentes usan el modelo de su definicion |
| Ejecucion | `docs/evidencias/sesiones/<archivo>.json` y `.meta.txt` |
| Commit base | `09b7ef9` |

## Objetivo

Revisar el proyecto terminado con tres revisores especializados trabajando **en paralelo** y
un cuarto que consolida sus reportes en una sola lista priorizada. No se corrige nada en
esta tarea: se encuentra y se ordena.

## Contexto suministrado

- `AGENTS.md`, reglas permanentes, se carga solo.
- Las definiciones de los subagentes, ya en `.claude/agents/`: `revisor-seguridad`,
  `revisor-rendimiento`, `revisor-cumplimiento` y `consolidador`. Cada una trae su alcance
  y sus herramientas, que son de solo lectura en los tres revisores.
- Todo el repositorio, que es lo que se audita.

## Instrucciones

1. **Lanza los tres revisores en paralelo**, en una sola tanda, no uno despues de otro:
   `revisor-seguridad`, `revisor-rendimiento` y `revisor-cumplimiento`.
2. Cada uno guarda su reporte en `reports/`: `seguridad.md`, `rendimiento.md` y
   `cumplimiento.md`.
3. Cuando los tres terminen, invoca al `consolidador`, que produce
   `reports/consolidado.md` con los hallazgos priorizados.
4. En tu reporte final anota **cuanto tardo la tanda en paralelo** y tu estimacion de lo que
   habria tardado en secuencial, con los tiempos de cada revisor. Es parte de lo que se
   evalua.
5. Cada hallazgo debe decir el archivo y la linea, por que es un problema y que se
   propone. **Un hallazgo sin ubicacion no sirve.**
6. Si un revisor no encuentra nada en su area, que lo diga. Un reporte vacio es un
   resultado valido; inventar hallazgos para llenarlo, no.

## Restricciones

- **No corrijas nada.** Esta tarea solo audita. Las correcciones van en otro prompt, con los
  hallazgos ya priorizados.
- No modifiques codigo, migraciones, pruebas ni documentacion fuera de `reports/`.
- Los tres revisores trabajan en solo lectura. No les cambies las herramientas.
- No toques `hooks/`, `.claude/`, `data/`, `docs/contexto/` ni `docs/diseno/`.
- No leas el archivo de entorno. Nunca ejecutes `make reset-dev` ni `down -v` fuera de
  `catalogo-test`.

## Salida esperada

Los cuatro archivos en `reports/` y un reporte final con: cuantos hallazgos encontro cada
revisor por severidad, los tiempos en paralelo y la comparacion con secuencial, y los tres
hallazgos que consideras mas graves.

## Criterio de aceptacion

1. Existen `reports/seguridad.md`, `reports/rendimiento.md`, `reports/cumplimiento.md` y
   `reports/consolidado.md`.
2. Los tres revisores se lanzaron en una sola tanda. Dilo explicitamente en el reporte.
3. Cada hallazgo del consolidado tiene archivo, linea o ubicacion equivalente, severidad y
   propuesta.
4. El consolidado no repite el mismo hallazgo tres veces por venir de tres reportes: los
   agrupa.
5. `git status` no muestra cambios fuera de `reports/`.
6. `make check` sigue terminando en 0, porque nada se toco.

---

> Lo de arriba es el texto que se envio, sin retocar. Lo de abajo se escribio despues.

## Resultado

Pendiente de ejecucion.
