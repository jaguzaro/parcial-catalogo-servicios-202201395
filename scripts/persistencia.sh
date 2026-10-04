#!/usr/bin/env bash
# P12: los datos sobreviven a un reinicio de contenedores sin borrar el volumen.
# Opera SOLO sobre el proyecto de pruebas catalogo-test (base catalogo_test).
# Reinicia con "stop" + "up -d --wait": no usa "down" ni borra volumenes.
set -euo pipefail

COMPOSE_TEST=(docker compose -p catalogo-test -f compose.yaml -f compose.test.yaml)
CODIGO="P12-$(date +%Y%m%d%H%M%S)-$$"
NOMBRE="Marcador de persistencia P12"

# POSTGRES_USER y POSTGRES_DB ya existen dentro del contenedor: no se lee el .env.
psql_db() {
  "${COMPOSE_TEST[@]}" exec -T db sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -v ON_ERROR_STOP=1 -tA -c "$1"' _ "$1"
}
volumen() { docker volume ls -q --filter name='^catalogo-test_catalogo_pgdata$'; }

echo "== P12 [1/5] levantar la base y aplicar migraciones (proyecto catalogo-test) =="
"${COMPOSE_TEST[@]}" up -d --wait db
"${COMPOSE_TEST[@]}" run --rm --build app migrar
VOL_ANTES="$(volumen)"
[ -n "$VOL_ANTES" ] || { echo "P12 FALLO: no existe el volumen catalogo-test_catalogo_pgdata"; exit 1; }
echo "volumen antes del reinicio: $VOL_ANTES"

echo "== P12 [2/5] escribir un dato reconocible =="
psql_db "INSERT INTO empresa (codigo, nombre) VALUES ('$CODIGO', '$NOMBRE')"
ESCRITO="$(psql_db "SELECT codigo || ' | ' || nombre FROM empresa WHERE codigo = '$CODIGO'")"
echo "escrito antes del reinicio: $ESCRITO"
[ -n "$ESCRITO" ] || { echo "P12 FALLO: el dato no quedo escrito"; exit 1; }

echo "== P12 [3/5] reiniciar los contenedores SIN borrar el volumen =="
echo "comando: ${COMPOSE_TEST[*]} stop   y luego   ${COMPOSE_TEST[*]} up -d --wait db"
"${COMPOSE_TEST[@]}" stop
"${COMPOSE_TEST[@]}" up -d --wait db

echo "== P12 [4/5] comprobar que el dato sigue ahi =="
VOL_DESPUES="$(volumen)"
echo "volumen despues del reinicio: $VOL_DESPUES"
LEIDO="$(psql_db "SELECT codigo || ' | ' || nombre FROM empresa WHERE codigo = '$CODIGO'")"
echo "leido despues del reinicio: ${LEIDO:-<nada>}"

echo "== P12 [5/5] resultado =="
psql_db "DELETE FROM empresa WHERE codigo = '$CODIGO'" >/dev/null
if [ "$VOL_ANTES" = "$VOL_DESPUES" ] && [ "$LEIDO" = "$ESCRITO" ]; then
  echo "P12 OK: el dato persistio tras el reinicio (mismo volumen, mismo contenido)"
else
  echo "P12 FALLO: el dato no persistio"
  exit 1
fi
