#!/usr/bin/env bash
# Verificacion manual de los criterios 3 a 8 del prompt 08, sobre el proyecto catalogo-test.
# No toca el proyecto de evaluacion ni lee el archivo de entorno: las cuentas DEMO de esta
# corrida se pasan con -e y solo existen en catalogo_test, que se resetea al terminar.
set -u
T=(docker compose -p catalogo-test -f compose.yaml -f compose.test.yaml)
ADMIN=verif_admin
CLAVE_ADMIN=Verif-Admin-2026
CONSULTA=verif_consulta
CLAVE_CONSULTA=Verif-Consulta-2026
COOKIES=reports/08-cookies.txt

psql() { "${T[@]}" exec -T db sh -c "psql -U \"\$POSTGRES_USER\" -d catalogo_test -At -c \"$1\""; }

echo "== importar (catalogo-test)"
"${T[@]}" run --rm --build app importar | grep -E "^importacion|servicio_n|total"
echo "exit=${PIPESTATUS[0]}"

for i in 1 2; do
  echo "== sembrar-demo, corrida $i"
  "${T[@]}" run --rm -e DEMO_ADMIN_USUARIO=$ADMIN -e DEMO_ADMIN_CONTRASENA=$CLAVE_ADMIN \
    -e DEMO_CONSULTA_USUARIO=$CONSULTA -e DEMO_CONSULTA_CONTRASENA=$CLAVE_CONSULTA app sembrar-demo
  echo "exit=$?"
done

echo "== criterio 7: asignaciones validas de la seccion DEMO tras dos corridas"
psql "SELECT sv.codigo || ' -> ' || s.codigo || ' / ' || u.usuario || ' (puesto en la misma seccion: ' || (p.seccion_id = s.id) || ')'
        FROM servicio_n2 sv JOIN seccion s ON s.id = sv.seccion_responsable_id
        LEFT JOIN usuario u ON u.id = sv.usuario_responsable_id LEFT JOIN puesto p ON p.id = u.puesto_id
       ORDER BY sv.codigo"
echo "servicios con responsable: $(psql 'SELECT count(*) FROM servicio_n2 WHERE seccion_responsable_id IS NOT NULL')"

echo "== levantar app de catalogo-test"
"${T[@]}" up -d --wait app >/dev/null 2>&1
URL="localhost:$("${T[@]}" port app 8080 | cut -d: -f2)"
echo "app en $URL"

pedir() { # metodo ruta [cuerpo]
  if [ $# -ge 3 ]; then
    curl -s -b $COOKIES -c $COOKIES -X "$1" -H 'Content-Type: application/json' -d "$3" -w '  [HTTP %{http_code}]\n' "$URL$2"
  else
    curl -s -b $COOKIES -c $COOKIES -X "$1" -w '  [HTTP %{http_code}]\n' "$URL$2"
  fi
}
campo() { grep -o "\"$1\":[^,}]*" | head -1; }

pedir POST /api/auth/login "{\"login\":\"$ADMIN\",\"contrasena\":\"$CLAVE_ADMIN\"}" | grep -o '\[HTTP [0-9]*\]'

N1_05=$(pedir GET '/api/servicios-n1?q=SE.05' | campo id | cut -d: -f2)
N1_06=$(pedir GET '/api/servicios-n1?q=SE.06' | campo id | cut -d: -f2)

echo "== criterio 8 (antes): servicio_n1=$(psql 'SELECT count(*) FROM servicio_n1') servicio_n2=$(psql 'SELECT count(*) FROM servicio_n2')"

echo "== criterio 3: minimo 24 y maximo 12"
pedir POST /api/servicios "{\"servicio_n1_id\":$N1_05,\"codigo\":\"VERIF.01\",\"nombre\":\"Verificacion 24-12\",\"minimo\":24,\"maximo\":12}"
echo "== criterio 3: minimo 12 y maximo 12"
pedir POST /api/servicios "{\"servicio_n1_id\":$N1_05,\"codigo\":\"VERIF.02\",\"nombre\":\"Verificacion 12-12\",\"minimo\":12,\"maximo\":12}" | grep -o '"minimo":[^,]*,"maximo":[^,]*\|\[HTTP [0-9]*\]'
echo "== criterio 3: solo minimo"
pedir POST /api/servicios "{\"servicio_n1_id\":$N1_05,\"codigo\":\"VERIF.03\",\"nombre\":\"Verificacion solo minimo\",\"minimo\":12}" | grep -o '"minimo":[^,]*,"maximo":[^,]*\|\[HTTP [0-9]*\]'
echo "en la base: $(psql "SELECT codigo || ' minimo=' || coalesce(minimo::text,'NULL') || ' maximo=' || coalesce(maximo::text,'NULL') FROM servicio_n2 WHERE codigo LIKE 'VERIF.%' ORDER BY codigo" | tr '\n' ';')"

echo "== criterio 4: filtro por SE.06"
pedir GET "/api/servicios-n1?q=SE.06" | campo servicios_n2
pedir GET "/api/servicios?n1_id=$N1_06&tamano=100" | grep -o '"total":[0-9]*\|\[HTTP [0-9]*\]'
echo "codigos de nivel 1 en el resultado: $(pedir GET "/api/servicios?n1_id=$N1_06&tamano=100" | grep -o '"servicio_n1":{"id":[0-9]*,"codigo":"[^"]*"' | grep -o 'SE\.[0-9]*"' | sort | uniq -c | tr '\n' ' ')"

echo "== criterio 5: buscar SE.06.01 y un texto que no existe"
pedir GET "/api/servicios?q=SE.06.01" | grep -o '"total":[0-9]*\|"codigo":"SE.06.01"\|\[HTTP [0-9]*\]' | sort -u
pedir GET "/api/servicios?q=no-existe-zzz"

echo "== criterio 6: usuario de otra seccion"
EMP=$(pedir POST /api/empresas '{"codigo":"VERIF","nombre":"Empresa de verificacion"}' | campo id | cut -d: -f2)
AREA=$(pedir POST /api/areas "{\"codigo\":\"VERIF\",\"nombre\":\"Area\",\"empresa_id\":$EMP}" | campo id | cut -d: -f2)
DEP=$(pedir POST /api/departamentos "{\"codigo\":\"VERIF\",\"nombre\":\"Depto\",\"area_id\":$AREA}" | campo id | cut -d: -f2)
SEC=$(pedir POST /api/secciones "{\"codigo\":\"VERIF\",\"nombre\":\"Otra seccion\",\"departamento_id\":$DEP}" | campo id | cut -d: -f2)
ADMIN_ID=$(psql "SELECT id FROM usuario WHERE usuario = '$ADMIN'")
SV=$(psql "SELECT id FROM servicio_n2 WHERE codigo = 'SE.06.01'")
pedir PUT "/api/servicios/$SV/responsable" "{\"seccion_id\":$SEC,\"usuario_id\":$ADMIN_ID}"
echo "responsable de SE.06.01 en la base: $(psql "SELECT coalesce(seccion_responsable_id::text,'NULL') || '/' || coalesce(usuario_responsable_id::text,'NULL') FROM servicio_n2 WHERE id = $SV")"

echo "== criterio 8 (despues): importados servicio_n1=$(psql 'SELECT count(*) FROM servicio_n1 WHERE origen_hoja IS NOT NULL') servicio_n2=$(psql 'SELECT count(*) FROM servicio_n2 WHERE origen_hoja IS NOT NULL'); creados por esta verificacion: $(psql "SELECT count(*) FROM servicio_n2 WHERE codigo LIKE 'VERIF.%'")"
echo "== reimportar: no crea nada y los conteos siguen"
"${T[@]}" run --rm app importar | grep -E "servicio_n"
echo "exit=${PIPESTATUS[0]}"
echo "servicio_n1=$(psql 'SELECT count(*) FROM servicio_n1') servicio_n2 importados=$(psql 'SELECT count(*) FROM servicio_n2 WHERE origen_hoja IS NOT NULL') asignaciones=$(psql 'SELECT count(*) FROM servicio_n2 WHERE seccion_responsable_id IS NOT NULL')"

rm -f $COOKIES
"${T[@]}" stop app >/dev/null 2>&1
