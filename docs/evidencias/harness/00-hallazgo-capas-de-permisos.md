# Capas de permisos y registro de los bloqueos

2026-10-04

La primera ejecucion de [`00-verificacion-harness.md`](../../prompts/00-verificacion-harness.md)
no confirmo que todo estuviera bien. Destapo dos problemas del entorno.

- Primera ejecucion, con fallos: [`20261004-000853-...json`](../sesiones/20261004-000853-00-verificacion-harness.json)
- Segunda ejecucion, correcta: [`20261004-002140-...json`](../sesiones/20261004-002140-00-verificacion-harness.json)

El prompt es el mismo archivo en las dos, con el mismo SHA-256 (`9a5a2424`). Lo que cambio
fue el entorno.

## 1. El workspace no estaba confiado

La ejecucion empezo asi:

```
Ignoring 6 permissions.allow entries from .claude/settings.json:
this workspace has not been trusted.
```

La lista `allow` del repositorio se ignoro entera. Por eso el control positivo nunca corrio:
`shasum -a 256 data/CatalogoServicios.xlsx` respondio `This command requires approval`, y
en una sesion sin persona no hay quien apruebe.

Es un fallo silencioso y enganoso, porque el harness parecia mas estricto y no mas flojo.
Habria frenado cualquier tarea que necesite correr `make`, `docker compose` o las pruebas.

Correccion: Joel acepto el dialogo de confianza. Es un permiso de la maquina, no del
repositorio. Quien clone el proyecto lo aceptara la primera vez, que es como debe ser: un
repositorio no deberia poder darse permisos a si mismo.

## 2. El `deny` se adelanta al hook en las herramientas de archivo

Dos intentos equivalentes terminaron distinto:

| Intento | Quien bloqueo | Quedo en `logs/hooks.log` |
|---|---|---|
| `git commit` desde Bash, estando en `deny` | `hooks/guard.sh` | Si |
| Write sobre el Excel, estando en `deny` | `permissions.deny` | No |

El asistente recibio `File is in a directory that is denied by your permission settings.` y
el hook nunca llego a correr. En Bash el hook va primero. En las herramientas de archivo
gana la regla `deny`.

En seguridad da igual, porque la accion quedo impedida en los dos casos. En evidencia no:
el intento no dejo rastro, y el registro de los intentos no permitidos es parte de lo que
se entrega.

Correccion: se quitaron del `deny` las reglas sobre `data/`, para que el Excel lo proteja
el hook, que si registra. El limite no se debilita, cambia de capa. Sobre el archivo quedan
tres controles: el hook, la revision de su SHA-256 y el `git diff` antes de cada commit.

Al aplicarlo, la herramienta aviso de que las reglas `Write(...)` no se aplican, porque la
edicion de archivos la cubre `Edit(...)`. Se quitaron por inutiles.

Sobre `hooks/**` y `.claude/**` el `deny` se queda. Ahi el razonamiento es al reves: si
alguien logra que el hook deje de correr, el hook no puede defenderse a si mismo.

## Nueva comprobacion

Segunda ejecucion del mismo prompt: HARNESS OK. Los cinco intentos prohibidos bloqueados
por `hooks/guard.sh` y registrados, y el control positivo corriendo con el SHA-256
coincidiendo. El detalle esta en [`00-bloqueos.md`](00-bloqueos.md).

## Lo que queda abierto

`guard.sh` protege `hooks/`, `.claude/settings.json` y `.claude/settings.local.json`, pero
no `.claude/agents/`. Un asistente podria reescribir la definicion de los subagentes
revisores, por ejemplo ampliando sus herramientas mas alla de solo lectura, sin que el hook
lo impida. Hoy eso lo cubre solo la regla `Edit(.claude/**)`.

Queda anotado en vez de corregido sobre la marcha, porque tocar el guardian obliga a
agregar su caso en `hooks/test-guard.sh` y volver a pasarla. Si se cierra, se anota aqui.
