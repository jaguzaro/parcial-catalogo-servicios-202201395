# Registro de decisiones

Cada decisión anota quién la tomó: propuesta de IA aceptada tal cual, propuesta modificada
o decisión humana.

| ID | Fecha | Decisión | Alternativa considerada | Justificación | Origen |
|---|---|---|---|---|---|
| D01 | 2026-10-03 | Declarar `hooks/guard.sh` como hook `PreToolUse` en un `.claude/settings.json` versionado en el repositorio. | Confiar en el sistema de permisos genérico de la herramienta. | Sin ese archivo el script no se ejecuta: la acción se rechaza, pero sin registro y sin distinguir casos ([detalle](../evidencias/harness/00-hallazgo-hook-no-registrado.md)). Versionarlo es lo que hace el control reproducible al clonar. | IA; causa raíz y restauración, Joel. |
| D02 | 2026-10-03 | Dar acceso amplio a Bash y poner el control fino en `hooks/guard.sh`, con una lista `deny` corta como segunda barrera. | Lista blanca de comandos permitidos. | La lista blanca se queda corta cada vez que aparece una herramienta nueva y obliga a ampliarla con prisa, que es cuando se cometen errores. Además rechaza sin dejar rastro. | IA, corregida por Joel. |
| D03 | 2026-10-03 | Versionar `logs/hooks.log`. | Tratarlo como archivo temporal. | Es la prueba de que los límites se aplicaron. Sin él, el harness solo estaría descrito. | IA. |
| D04 | 2026-10-03 | Fijar el SHA-256 del Excel en `docs/contexto/excel-sha256.txt` antes de empezar. | Comprobar la integridad del archivo al final. | Fijarlo antes de tocar nada permite demostrar que siguió intacto durante todo el desarrollo, no solo al cierre. | IA. |
| D05 | 2026-10-03 | No modificar `hooks/guard.sh` pese a sus falsos positivos. | Relajar las reglas que los provocan. | Los falsos positivos bloquean de más, nunca de menos. Relajar un límite por comodidad es justo lo que no se debe hacer. [Limitaciones documentadas](../evidencias/harness/00-limitaciones-guard.md). | IA. |
| D06 | 2026-10-04 | Quitar del `deny` las reglas sobre `data/` y dejar que el Excel lo bloquee el hook. Conservar el `deny` sobre `hooks/**` y `.claude/**`. | Mantener el `deny` sobre el Excel y aceptar que ese intento no quede registrado. | En las herramientas de archivo el `deny` se evalúa antes que el hook: impide la acción pero no la registra. El límite no se debilita, cambia a la capa que deja evidencia. En `hooks/**` y `.claude/**` el argumento se invierte: el hook no puede defenderse a sí mismo. [Detalle](../evidencias/harness/00-hallazgo-capas-de-permisos.md). | IA, aplicada por Joel. |
| D06.1 | 2026-10-04 | Eliminar también las reglas `Write(...)` de la lista `deny`. | Dejarlas por si acaso. | La herramienta avisó de que no se aplican: la edición la cubre `Edit(...)`. Una regla que no hace nada da falsa sensación de control al leer la configuración. | Joel. |
| D06.2 | 2026-10-04 | Quitar `Read(.env.*)` del `deny`. | Mantenerla y añadir una excepción para `.env.example`. | Ese patrón alcanzaba a `.env.example`, que hay que poder leer. `.env` sigue cubierto por `Read(.env)` y por `guard.sh`. | Joel. |
| D07 | 2026-10-04 | Aceptar el diálogo de confianza del workspace. | Conceder los permisos por línea de comandos al lanzar el asistente. | Sin confianza, la lista `allow` del repositorio se ignora y el asistente no puede ejecutar `make`, `docker` ni las pruebas. La alternativa deja los permisos fuera del repositorio y el harness no se reproduce al clonar. | IA, aplicada por Joel. |

> El stack técnico —lenguaje, base de datos, hash de contraseñas, mecanismo de sesión— se
> anota aquí junto con el diseño que lo justifique.
