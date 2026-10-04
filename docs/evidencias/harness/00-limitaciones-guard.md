# Limitaciones conocidas de `hooks/guard.sh`

2026-10-04

El guardian decide mirando el texto completo del comando, no su estructura. Es lo que lo
hace simple y facil de auditar, y tiene dos consecuencias que conviene dejar escritas,
porque aparecen durante el trabajo normal.

Se anotan como limitaciones. No se debilito ninguna regla para evitarlas.

## 1. Falsos positivos por contenido literal

Un comando que menciona un patron prohibido queda bloqueado aunque no lo ejecute. Por
ejemplo, escribir un archivo cuyo contenido incluye la lista de acciones prohibidas:

```bash
cat > .claude/settings.json <<'JSON'
{ "deny": ["Bash(docker volume rm:*)"] }
JSON
```

El resultado es `BLOQUEADO: borrado de volumenes o limpieza global de Docker`.

El guardian ve la cadena dentro del texto y no distingue que esta en un literal. Pasa lo
mismo con `.env`, `git commit`, `/etc` o `DROP TABLE` citados dentro de documentacion.

Otros dos casos reales:

| Comando | Motivo del bloqueo | Por que no corresponde |
|---|---|---|
| `grep ... \| sed 's/^/  /'` | ruta absoluta fuera del proyecto: `/` | el `/` es el separador de `sed` |
| `jq -r '.result // .'` | ruta absoluta fuera del proyecto: `//` | `//` es el operador "alternativa" de `jq` |

Como se trabaja con esto, sin tocar el guardian:

- Para escribir archivos cuyo contenido menciona patrones prohibidos, usar la herramienta
  de escritura de archivos en lugar de un `heredoc`. El guardian revisa la ruta del
  archivo, que es el control que corresponde ahi.
- Usar otro separador en `sed`, por ejemplo `s|^|  |`, y evitar `//` en `jq`.
- Un comando por proposito, en vez de encadenar varios con `&&`. Como se evalua el texto
  completo, una sola mencion bloquea el bloque entero.

## 2. Las reglas del Excel miran todo el comando

Las reglas que protegen `data/CatalogoServicios.xlsx` buscan un verbo peligroso (`cp`,
`mv`, `rm`, `tee`, `sed -i`) en cualquier parte del comando y el nombre del archivo en
cualquier otra. Esto queda bloqueado aunque el `cp` no toque el Excel:

```bash
cp docs/a.md docs/b.md
ls -l data/CatalogoServicios.xlsx
```

## Que no significan

Ninguna de las dos deja pasar algo prohibido. Fallan bloqueando de mas, nunca de menos. El
costo es incomodidad al escribir comandos, no un agujero. Las 33 comprobaciones de
`hooks/test-guard.sh`, 12 permitidas y 21 bloqueadas, siguen pasando:
[`00-test-guard.txt`](00-test-guard.txt).
