# Limitaciones conocidas de `hooks/guard.sh`

**2026-10-04**

El guardián decide mirando el texto completo del comando, no su estructura. Es lo que lo
hace simple y auditable, y tiene dos consecuencias que conviene dejar escritas porque
aparecen durante el trabajo normal.

Se documentan como limitaciones. No se debilitó ninguna regla para evitarlas.

## 1. Falsos positivos por contenido literal

Un comando que *menciona* un patrón prohibido queda bloqueado aunque no lo ejecute. Por
ejemplo, escribir un archivo cuyo contenido incluye la lista de acciones prohibidas:

```bash
cat > .claude/settings.json <<'JSON'
{ "deny": ["Bash(docker volume rm:*)"] }
JSON
```

→ `BLOQUEADO: borrado de volúmenes o limpieza global de Docker`

El guardián ve la cadena dentro del texto y no distingue que está en un literal. Pasa igual
con `.env`, `git commit`, `/etc` o `DROP TABLE` citados dentro de documentación.

Otros dos casos reales:

| Comando | Motivo del bloqueo | Por qué no corresponde |
|---|---|---|
| `grep … \| sed 's/^/  /'` | ruta absoluta fuera del proyecto: `/` | el `/` es el separador de `sed` |
| `jq -r '.result // .'` | ruta absoluta fuera del proyecto: `//` | `//` es el operador "alternativa" de `jq` |

Cómo se trabaja con esto, sin tocar el guardián:

- Para escribir archivos cuyo contenido menciona patrones prohibidos, usar la herramienta de
  escritura de archivos en lugar de un `heredoc`. El guardián valida la ruta del archivo,
  que es el control que corresponde ahí.
- Usar otros separadores en `sed` (`s|^|  |`) y evitar `//` en `jq`.
- Un comando por propósito, en vez de encadenar varios con `&&`: como se evalúa el texto
  completo, una sola mención bloquea el bloque entero.

## 2. Las reglas del Excel miran todo el comando

Las reglas que protegen `data/CatalogoServicios.xlsx` buscan un verbo peligroso (`cp`, `mv`,
`rm`, `tee`, `sed -i`…) en cualquier parte del comando y el nombre del archivo en cualquier
otra. Esto queda bloqueado aunque el `cp` no toque el Excel:

```bash
cp docs/a.md docs/b.md
ls -l data/CatalogoServicios.xlsx
```

## Qué no significan

Ninguna de las dos deja pasar algo prohibido: fallan bloqueando de más, nunca de menos. El
costo es incomodidad al escribir comandos, no un agujero. Las 33 comprobaciones de
`hooks/test-guard.sh` —12 permitidas y 21 bloqueadas— siguen pasando:
[`00-test-guard.txt`](00-test-guard.txt).
