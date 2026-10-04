#!/usr/bin/env bash
# Repeticion del criterio 6 del prompt 08 sobre catalogo-test. La primera corrida tomo el id
# del padre anidado en vez del id del registro; aqui los ids salen de la base.
set -u
T=(docker compose -p catalogo-test -f compose.yaml -f compose.test.yaml)
COOKIES=reports/08-cookies.txt
psql() { "${T[@]}" exec -T db sh -c "psql -U \"\$POSTGRES_USER\" -d catalogo_test -At -c \"$1\""; }

"${T[@]}" up -d --wait app >/dev/null 2>&1
URL="localhost:$("${T[@]}" port app 8080 | cut -d: -f2)"
pedir() {
  curl -s -b $COOKIES -c $COOKIES -X "$1" -H 'Content-Type: application/json' -d "$3" -w '  [HTTP %{http_code}]\n' "$URL$2"
}
pedir POST /api/auth/login '{"login":"verif_admin","contrasena":"Verif-Admin-2026"}' | grep -o '\[HTTP [0-9]*\]'

echo "== estructura de otra seccion (VERIF)"
psql "SELECT 'empresas VERIF: ' || count(*) FROM empresa WHERE codigo = 'VERIF'"
EMP=$(psql "SELECT id FROM empresa WHERE codigo = 'VERIF'")
pedir POST /api/areas "{\"codigo\":\"VERIF\",\"nombre\":\"Area\",\"empresa_id\":$EMP}" | grep -o '\[HTTP [0-9]*\]'
AREA=$(psql "SELECT id FROM area WHERE codigo = 'VERIF' AND empresa_id = $EMP")
pedir POST /api/departamentos "{\"codigo\":\"VERIF\",\"nombre\":\"Depto\",\"area_id\":$AREA}" | grep -o '\[HTTP [0-9]*\]'
DEP=$(psql "SELECT id FROM departamento WHERE codigo = 'VERIF' AND area_id = $AREA")
pedir POST /api/secciones "{\"codigo\":\"VERIF\",\"nombre\":\"Otra seccion\",\"departamento_id\":$DEP}" | grep -o '\[HTTP [0-9]*\]'
SEC=$(psql "SELECT id FROM seccion WHERE codigo = 'VERIF' AND departamento_id = $DEP")
ADMIN_ID=$(psql "SELECT id FROM usuario WHERE usuario = 'verif_admin'")
SV=$(psql "SELECT id FROM servicio_n2 WHERE codigo = 'SE.06.01'")
echo "seccion VERIF id=$SEC; verif_admin id=$ADMIN_ID, su seccion: $(psql "SELECT s.codigo || ' id ' || s.id FROM usuario u JOIN puesto p ON p.id = u.puesto_id JOIN seccion s ON s.id = p.seccion_id WHERE u.id = $ADMIN_ID")"

echo "== criterio 6: asignar a SE.06.01 la seccion VERIF con verif_admin, que es de DEMO"
pedir PUT "/api/servicios/$SV/responsable" "{\"seccion_id\":$SEC,\"usuario_id\":$ADMIN_ID}"
echo "responsable de SE.06.01 en la base: $(psql "SELECT coalesce(seccion_responsable_id::text,'NULL') || '/' || coalesce(usuario_responsable_id::text,'NULL') FROM servicio_n2 WHERE id = $SV")"

rm -f $COOKIES
"${T[@]}" stop app >/dev/null 2>&1
