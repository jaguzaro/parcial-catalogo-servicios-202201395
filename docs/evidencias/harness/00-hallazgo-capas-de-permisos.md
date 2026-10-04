# Qué capa bloquea, y por qué importa cuál

**2026-10-04**

La primera ejecución de [`00-verificacion-harness.md`](../../prompts/00-verificacion-harness.md)
no confirmó que todo estuviera bien: destapó dos problemas del entorno.

- Primera ejecución, falla: [`20261004-000853-…json`](../sesiones/20261004-000853-00-verificacion-harness.json)
- Segunda ejecución, éxito: [`20261004-002140-…json`](../sesiones/20261004-002140-00-verificacion-harness.json)

El prompt es el mismo archivo en las dos, con el mismo SHA-256 (`9a5a2424…d772b`). Lo que
cambió fue el entorno.

## 1. El workspace no estaba confiado

La ejecución empezó así:

```
Ignoring 6 permissions.allow entries from .claude/settings.json:
this workspace has not been trusted.
```

La lista `allow` del repositorio se ignoró entera. Por eso el control positivo nunca corrió:
`shasum -a 256 data/CatalogoServicios.xlsx` respondió `This command requires approval`, y en
una sesión sin persona no hay quien apruebe.

El fallo es silencioso y engaña en la dirección contraria a la habitual: el harness parecía
más estricto, no más flojo. Habría frenado cualquier tarea que necesite ejecutar `make`,
`docker compose` o las pruebas.

**Corrección:** Joel aceptó el diálogo de confianza. Es un permiso de la máquina, no del
repositorio. Quien clone el proyecto lo aceptará la primera vez, que es como debe ser: un
repositorio no debería poder concederse permisos a sí mismo.

## 2. `permissions.deny` se adelanta al hook en las herramientas de archivo

Dos intentos equivalentes terminaron distinto:

| Intento | Quién bloqueó | ¿Quedó en `logs/hooks.log`? |
|---|---|---|
| `git commit` desde Bash, estando en `deny` | `hooks/guard.sh` | Sí |
| Write sobre el Excel, estando en `deny` | `permissions.deny` | No |

El asistente recibió `File is in a directory that is denied by your permission settings.` y
el hook no llegó a ejecutarse. En Bash el hook corre primero; en las herramientas de archivo
gana la regla `deny`.

En seguridad da igual: la acción quedó impedida en los dos casos. En evidencia no: el
intento no dejó rastro, y el registro de los intentos no permitidos es parte de lo que se
entrega.

**Corrección:** se quitaron de `deny` las reglas sobre `data/`, para que el Excel lo proteja
el hook, que sí registra. El límite no se debilita, cambia de capa. Sobre el archivo quedan
tres controles: el hook, la verificación de su SHA-256 y la revisión de `git diff` antes de
cada commit.

Al aplicarlo, la herramienta avisó de que las reglas `Write(...)` no se aplican —la edición
de archivos la cubre `Edit(...)`—, así que se eliminaron por inútiles.

Sobre `hooks/**` y `.claude/**` el `deny` se conserva. Ahí el argumento se invierte: si
alguien consigue que el hook deje de ejecutarse, el hook no puede defenderse a sí mismo.

## Reverificación

Segunda ejecución del mismo prompt: **HARNESS OK**. Los cinco intentos prohibidos bloqueados
por `hooks/guard.sh` y registrados; el control positivo ejecutándose y el SHA-256
coincidiendo. Detalle en [`00-bloqueos.md`](00-bloqueos.md).

## Hueco que queda abierto

`guard.sh` protege `hooks/`, `.claude/settings.json` y `.claude/settings.local.json`, pero no
`.claude/agents/`. Un asistente podría reescribir la definición de los subagentes revisores
—por ejemplo, ampliando sus herramientas más allá de solo lectura— sin que el hook lo
impida. Hoy eso lo cubre solo la regla `Edit(.claude/**)`.

Queda anotado en vez de corregido sobre la marcha: tocar el guardián obliga a añadir su caso
en `hooks/test-guard.sh` y a volver a pasarla. Si se cierra, se anota aquí.
