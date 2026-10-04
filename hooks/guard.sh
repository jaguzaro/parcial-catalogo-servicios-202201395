#!/usr/bin/env bash
# guard.sh — Hook PreToolUse de Claude Code.
# Recibe por stdin el JSON del evento. Exit 0 = permitir; exit 2 = bloquear
# (el mensaje de stderr se devuelve al agente). Cada bloqueo queda en logs/hooks.log.
#
# Variables opcionales:
#   GUARD_EXTRA_DIRS       rutas adicionales permitidas, separadas por ':' (relativas al proyecto)
#   GUARD_PROTECT_HARNESS  1 (por defecto) impide editar hooks/ y .claude/settings*.json
set -uo pipefail

INPUT="$(cat)"
PROJECT_DIR="${CLAUDE_PROJECT_DIR:-$(pwd)}"
PROTECT_HARNESS="${GUARD_PROTECT_HARNESS:-1}"
LOG_DIR="$PROJECT_DIR/logs"
mkdir -p "$LOG_DIR"
LOG_FILE="$LOG_DIR/hooks.log"

if ! command -v jq >/dev/null 2>&1; then
  echo "BLOQUEADO: jq no está instalado; el guardián no puede validar la acción." >&2
  exit 2
fi

TOOL="$(jq -r '.tool_name // ""' <<<"$INPUT")"
CMD="$(jq -r '.tool_input.command // ""' <<<"$INPUT")"
FILE="$(jq -r '.tool_input.file_path // .tool_input.notebook_path // .tool_input.path // ""' <<<"$INPUT")"

log_event() {
  jq -nc --arg ts "$(date -u +%Y-%m-%dT%H:%M:%SZ)" --arg tool "$TOOL" \
    --arg command "$CMD" --arg file "$FILE" --arg decision "$1" --arg reason "$2" \
    '{ts:$ts, tool:$tool, command:$command, file:$file, decision:$decision, reason:$reason}' >> "$LOG_FILE"
}
block() { log_event "blocked" "$1"; echo "BLOQUEADO por hooks/guard.sh: $1" >&2; exit 2; }

# Normaliza una ruta (relativa al proyecto) sin exigir que exista.
norm() {
  local p="$1" out
  if [[ "$p" == "~" || "$p" == "~/"* ]]; then p="${HOME}${p:1}"; fi
  [[ "$p" != /* ]] && p="$PROJECT_DIR/$p"
  if out="$(realpath -m "$p" 2>/dev/null)"; then echo "$out"
  elif out="$(python3 -c 'import os,sys;print(os.path.normpath(sys.argv[1]))' "$p" 2>/dev/null)"; then echo "$out"
  else echo "$p"; fi
}

ROOTS=("$(norm "$PROJECT_DIR")")
if [[ -n "${GUARD_EXTRA_DIRS:-}" ]]; then
  IFS=':' read -ra EXTRA <<<"$GUARD_EXTRA_DIRS"
  for d in "${EXTRA[@]}"; do [[ -n "$d" ]] && ROOTS+=("$(norm "$d")"); done
fi

inside_roots() {
  local p r; p="$(norm "$1")"
  for r in "${ROOTS[@]}"; do
    if [[ "$p" == "$r" || "$p" == "$r"/* ]]; then return 0; fi
  done
  return 1
}

# ---------- Herramientas de archivos ----------
case "$TOOL" in
  Read|Edit|MultiEdit|Write|NotebookEdit|Glob|Grep)
    if [[ -n "$FILE" ]]; then
      inside_roots "$FILE" || block "ruta fuera del proyecto: $FILE"
      base="$(basename "$FILE")"
      if [[ "$base" =~ ^\.env(\..+)?$ && "$base" != ".env.example" ]]; then
        block "acceso a archivo de secretos: $base"
      fi
      if [[ "$TOOL" != "Read" && "$TOOL" != "Glob" && "$TOOL" != "Grep" ]]; then
        if [[ "$(norm "$FILE")" == */data/CatalogoServicios.xlsx ]]; then
          block "el Excel original es de solo lectura"
        fi
        if [[ "$PROTECT_HARNESS" == "1" ]]; then
          case "$(norm "$FILE")" in
            */hooks/*|*/.claude/settings.json|*/.claude/settings.local.json|*/.claude/hooks/*)
              block "el harness (hooks y permisos) no puede modificarse desde esta sesión" ;;
          esac
        fi
      fi
    fi
    exit 0 ;;
  Bash) ;;
  *) exit 0 ;;
esac

# ---------- Bash ----------
c="$CMD"
SEP='(^|[;&|(`[:space:]])'

re="${SEP}git([[:space:]]+-C[[:space:]]+[^[:space:]]+)?[[:space:]]+(commit|push|tag|reset|checkout|switch|merge|rebase|restore|clean|stash|cherry-pick|revert|remote|config|init|am|apply|rm|mv|branch[[:space:]]+-[dDmM])([[:space:]]|$)"
[[ "$c" =~ $re ]] && block "las operaciones git de escritura las realiza solo Joel"

re="${SEP}sudo([[:space:]]|$)"
[[ "$c" =~ $re ]] && block "sudo no está permitido"

re='(curl|wget)[^|]*\|[[:space:]]*(sudo[[:space:]]+)?(ba|z)?sh'
[[ "$c" =~ $re ]] && block "descargar y ejecutar scripts remotos no está permitido"

re='docker[[:space:]]+(volume[[:space:]]+(rm|prune)|system[[:space:]]+prune)'
[[ "$c" =~ $re ]] && block "borrado de volúmenes o limpieza global de Docker"

re='(docker[[:space:]]+compose|docker-compose).*[[:space:]]down([[:space:]].*)?[[:space:]](-v|--volumes)([[:space:]]|$)'
if [[ "$c" =~ $re && ! "$c" =~ catalogo-test ]]; then
  block "'down -v' solo se permite en el proyecto de pruebas (-p catalogo-test); el reinicio destructivo de datos de evaluación lo ejecuta Joel"
fi

shopt -s nocasematch
re='(drop[[:space:]]+(database|schema|table)|truncate[[:space:]])'
if [[ "$c" =~ $re && ! "$c" =~ catalogo[-_]test ]]; then
  shopt -u nocasematch
  block "SQL destructivo fuera de la base de pruebas"
fi
shopt -u nocasematch

re='(^|[^[:alnum:]_.-])\.env([^[:alnum:]_.-]|$)|\.env\.(local|dev|development|prod|production)'
[[ "$c" =~ $re ]] && block "acceso a archivo de secretos (.env); usar .env.example o los targets del Makefile"

if [[ "$c" == *CatalogoServicios.xlsx* ]]; then
  re="${SEP}(rm|mv|truncate|shred|chmod|chown|tee|dd|sed[[:space:]]+-i)([[:space:]]|$)"
  [[ "$c" =~ $re ]] && block "el Excel original es de solo lectura"
  re='>[[:space:]]*[^[:space:]]*CatalogoServicios\.xlsx'
  [[ "$c" =~ $re ]] && block "el Excel original es de solo lectura"
  re='(^|[[:space:]])cp[[:space:]].*[[:space:]][^[:space:]]*CatalogoServicios\.xlsx[[:space:]]*$'
  [[ "$c" =~ $re ]] && block "el Excel original es de solo lectura"
fi

re="${SEP}rm[[:space:]]+(-[[:alnum:]-]+[[:space:]]+)*(~|\.\.|\*([[:space:]]|$)|\.([[:space:]]|$))"
[[ "$c" =~ $re ]] && block "rm sobre ~, .., * o el directorio actual"

# Rutas absolutas fuera del proyecto.
re="${SEP}(docker|docker-compose)[[:space:]]"
if [[ "$c" =~ $re ]]; then
  # En comandos Docker las rutas absolutas suelen ser del contenedor:
  # solo se valida el origen en el host de los montajes -v/--volume.
  while read -r src; do
    src="${src#\"}"; src="${src#\'}"
    [[ -z "$src" ]] && continue
    if [[ "$src" == /* || "$src" == "~"* ]] && ! inside_roots "$src"; then
      block "montaje Docker de una ruta del host fuera del proyecto: $src"
    fi
  done < <(grep -oE -- '(-v|--volume)[[:space:]=]+[^:[:space:]]+' <<<"$c" | sed -E 's/^(-v|--volume)[[:space:]=]+//')
else
  ABS_RE="(^|[[:space:]=\"'])(/|~/)[^[:space:]\"';|&)<>]*"
  while read -r p; do
    p="${p#[[:space:]]}"; p="${p#=}"; p="${p#\"}"; p="${p#\'}"
    [[ -z "$p" ]] && continue
    case "$p" in /dev/null|/dev/stdout|/dev/stderr|/tmp|/tmp/*) continue ;; esac
    inside_roots "$p" || block "ruta absoluta fuera del proyecto: $p"
  done < <(grep -oE -- "$ABS_RE" <<<"$c")
fi

exit 0
