# Registro de contexto

Qué contexto recibió el asistente en cada tarea y por qué. Las actualizaciones de
`AGENTS.md` enlazan al hallazgo o decisión que las motivó.

El criterio es el mismo siempre: `AGENTS.md` se carga solo, y además se entregan únicamente
los archivos que esa tarea necesita leer. Lo que se deja fuera también se anota, porque dar
de más invita a trabajar fuera de lo pedido.

## Contexto entregado

| Tarea | Prompt | Archivos entregados | Por qué | Excluido a propósito |
|---|---|---|---|---|
| Verificar los límites del harness | [`00-verificacion-harness.md`](../prompts/00-verificacion-harness.md) | `AGENTS.md` (automático); `hooks/guard.sh`; `docs/contexto/excel-sha256.txt` | La tarea es probar los límites, no construir. Basta la sección 4 de `AGENTS.md`, el script que los aplica y el hash esperado para el control positivo. | El enunciado completo: el asistente no necesita conocer el alcance funcional para esto, y conocerlo invita a adelantar implementación durante una verificación. |

## Actualizaciones de `AGENTS.md`

| Versión | Fecha | Qué la motiva | Secciones | Commit |
|---|---|---|---|---|
| v1 | 2026-10-03 | Contexto inicial | Todas | `7c38301` |
