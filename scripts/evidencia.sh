#!/usr/bin/env bash
# Ejecuta "make check" y guarda la salida con cabecera en docs/evidencias/pruebas/.
# La evidencia se guarda aunque check falle, y el script termina con el codigo de check.
set -uo pipefail

FECHA="$(date +%Y%m%d-%H%M%S)"
COMMIT="$(git rev-parse --short HEAD)"
SUCIO=""
[ -n "$(git status --porcelain)" ] && SUCIO=" (con cambios sin versionar)"
DIR="docs/evidencias/pruebas"
ARCHIVO="$DIR/$FECHA-$COMMIT.txt"
mkdir -p "$DIR"

{
  echo "Fecha:       $(date '+%Y-%m-%d %H:%M:%S %z')"
  echo "Commit:      $COMMIT$SUCIO"
  echo "Herramienta: $(claude --version 2>/dev/null || echo 'Claude Code no disponible en esta maquina') ; $(docker --version)"
  echo "Comando:     make check"
  echo "=========================================================================="
} > "$ARCHIVO"

make check >> "$ARCHIVO" 2>&1
CODIGO=$?

{
  echo "=========================================================================="
  echo "Codigo de salida de make check: $CODIGO"
} >> "$ARCHIVO"

echo "Evidencia guardada en $ARCHIVO (codigo de salida: $CODIGO)"
exit "$CODIGO"
