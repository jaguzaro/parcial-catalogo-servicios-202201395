# 13 - Correcciones de la auditoria

| Campo | Valor |
|---|---|
| Fecha | 2026-10-04 |
| Herramienta | Claude Code `2.1.289`, modo headless `claude -p` |
| Modelo | `sonnet` |
| Ejecucion | `docs/evidencias/sesiones/<archivo>.json` y `.meta.txt` |
| Commit base | `09b7ef9` |

## Objetivo

Corregir tres hallazgos de la auditoria. Solo esos tres, y nada mas.

## Contexto suministrado

- `AGENTS.md`, reglas permanentes, se carga solo.
- `reports/consolidado.md`, la lista priorizada de la auditoria.
- `reports/seguridad.md` y `reports/rendimiento.md`, para el detalle de estos tres.

## Instrucciones

Corrige exactamente estos, en este orden:

1. **REN-01 y SEG-05: el servidor HTTP no tiene tiempos limite.** Ponle `ReadTimeout`,
   `WriteTimeout`, `IdleTimeout` y `ReadHeaderTimeout` con valores razonables y explicados
   en un comentario. Ten en cuenta que iniciar sesion tarda, porque argon2id esta
   configurado con 64 MiB.
2. **REN-04: la tabla de sesiones no tiene indice por fecha de expiracion.** Agrega una
   **migracion nueva** con ese indice. No edites ninguna migracion existente.
3. **SEG-03: faltan cabeceras de seguridad.** Agrega las habituales para una aplicacion que
   se sirve a si misma: `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy` y
   una `Content-Security-Policy` que **no rompa la interfaz compilada**. Comprueba que la
   interfaz sigue cargando: una politica que rompe la aplicacion es peor que no tenerla.

## Restricciones

- **Solo esos tres hallazgos.** Los demas quedan documentados como limitaciones conocidas,
  que es una decision tomada, no un olvido.
- No cambies el comportamiento de ningun endpoint ni el formato de las respuestas.
- No toques `hooks/`, `.claude/`, `data/`, `docs/contexto/`, `docs/diseno/` ni `reports/`.
- No leas el archivo de entorno. Nunca `make reset-dev` ni `down -v` fuera de
  `catalogo-test`.

## Salida esperada

Los cambios y un reporte final con: que valor le pusiste a cada tiempo limite y por que, la
migracion nueva, las cabeceras elegidas, y la comprobacion de que la interfaz sigue
funcionando.

## Criterio de aceptacion

1. `make check` termina en 0 y P01 a P11 siguen pasando.
2. `make e2e` termina en 0: **es la prueba de que la politica de contenidos no rompio la
   interfaz**. Si falla, la politica esta mal y hay que ajustarla, no quitar la prueba.
3. `make migrate` dos veces: las dos en 0 y la segunda sin aplicar nada.
4. Con el sistema levantado, la respuesta de la raiz trae las cuatro cabeceras nuevas.
5. Iniciar sesion sigue funcionando: los tiempos limite no lo cortan.

---

> Lo de arriba es el texto que se envio, sin retocar. Lo de abajo se escribio despues.

## Resultado

Pendiente de ejecucion.
