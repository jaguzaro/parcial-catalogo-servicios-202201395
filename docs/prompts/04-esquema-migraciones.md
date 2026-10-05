# 04 - Esquema completo como migraciones

| Campo | Valor |
|---|---|
| Fecha | 2026-10-04 |
| Herramienta | Claude Code `2.1.289`, modo headless `claude -p` |
| Modelo | `sonnet` |
| Ejecucion | `docs/evidencias/sesiones/<archivo>.json` y `.meta.txt` |
| Commit base | `2cc8639` |

## Objetivo

Llevar las 14 entidades del diseno a migraciones de `goose`, con sus llaves, restricciones
y comprobaciones puestas **en la base**. Y pruebas que demuestren que la base rechaza de
verdad lo que tiene que rechazar.

## Contexto suministrado

- `AGENTS.md`, reglas permanentes, se carga solo. Seccion 7: entidades y reglas. Seccion 9:
  `make check`.
- `docs/diseno/modelo.md`, el diccionario de datos. Es la fuente de esta tarea: columnas,
  tipos, nulos, llaves primarias, llaves foraneas y unicidad.
- `docs/diseno/reglas.md`, las reglas de negocio. De aqui sale que restricciones conviene
  que vivan en la base y no solo en el codigo.

El enunciado no hace falta: lo que pide ya esta recogido en el diseno.

## Instrucciones

1. Escribe las migraciones en `internal/db/migraciones/`, con el formato de `goose` y su
   seccion de bajada. Puedes repartirlas en varios archivos si queda mas claro, numerados
   en orden.
2. Crea las 14 tablas del diccionario: `empresa`, `area`, `departamento`, `seccion`,
   `puesto`, `usuario`, `sesion`, `servicio_n1`, `servicio_n2`, `clase_servicio`,
   `criticidad`, `tipo_servicio`, `importacion` e `incidencia`.
3. Que la base haga cumplir, no solo el codigo:
   - llaves primarias y llaves foraneas de toda la jerarquia;
   - codigo unico global en los servicios, y unico dentro del padre en las unidades de la
     organizacion;
   - `usuario`: usuario o correo unico;
   - `servicio_n2.activo` solo admite `S`, `N` o `DESCONOCIDO`;
   - si hay minimo y maximo, minimo no puede ser mayor que maximo. Que uno falte no es
     motivo de rechazo, y no se convierte en cero;
   - lo que el diccionario marque como no nulo.
4. Indices para las busquedas y los filtros del catalogo: codigo, nombre, nivel 1, estado,
   clase, criticidad y tipo.
5. **Sin datos.** Ninguna migracion inserta filas. Los catalogos de clase, criticidad y
   tipo se cargan desde el Excel cuando corra el importador (D12), y las cuentas de
   evaluacion se siembran por su propio comando.
6. Pruebas nuevas en Go, contra la base de pruebas, que comprueben que la base rechaza:
   - dos servicios de nivel 1 con el mismo codigo;
   - dos areas con el mismo codigo dentro de la misma empresa, y que si lo permite en
     empresas distintas;
   - un hijo que apunte a un padre que no existe;
   - un `activo` fuera de los tres valores validos;
   - un minimo mayor que el maximo, aceptando el caso en que uno de los dos falta.

   Cada prueba comprueba que la operacion falla, no solo que devuelve error: que lo haga
   por la restriccion que toca.
7. Si `make test` no puede ejecutar pruebas que necesitan base, ajusta lo necesario para
   que el servicio de pruebas espere a la base y aplique las migraciones antes. Puedes
   tocar el `Makefile` y `compose.test.yaml`.

## Restricciones

- No toques `hooks/`, `.claude/`, `data/`, `docs/contexto/` ni `docs/diseno/`.
- Nada de logica de negocio: ni handlers, ni autenticacion, ni importador, ni interfaz.
  Solo esquema y las pruebas que lo comprueban.
- No inventes columnas ni tablas que no esten en el diccionario. Si algo falta o se
  contradice, implementa lo que si este y dilo en el reporte.
- Nunca ejecutes `make reset-dev`, ni `down -v` fuera del proyecto `catalogo-test`.
- No leas el archivo de entorno.

## Salida esperada

Las migraciones, las pruebas, y un reporte final con: las tablas creadas, las restricciones
que puso la base, los comandos ejecutados con su codigo de salida y lo que quede pendiente.

## Criterio de aceptacion

1. `make reset-test` y despues `make test`: los dos en 0. Demuestra que las migraciones se
   aplican desde cero sobre una base vacia.
2. `make migrate` dos veces seguidas: las dos en 0, y la segunda no aplica nada.
3. `make check` termina en 0.
4. Las pruebas del punto 6 estan y pasan. Si se quita a proposito una restriccion de la
   migracion, su prueba falla; comprobalo al menos con una y deja la restriccion como
   estaba.
5. Una consulta al catalogo del sistema devuelve las 14 tablas.
6. Ninguna migracion contiene `INSERT`.

---

> Lo de arriba es el texto que se envio, sin retocar. Lo de abajo se escribio despues.

## Resultado

**Aceptado.** Sesion:
[`20261004-095550-04-esquema-migraciones.json`](../evidencias/sesiones/20261004-095550-04-esquema-migraciones.json),
21 turnos.

Tres migraciones nuevas, repartidas por area: `00002_organizacion.sql` (empresa, area,
departamento, seccion, puesto, usuario y sesion), `00003_importacion.sql` y
`00004_catalogo.sql` (los tres catalogos, los dos niveles de servicio e incidencia). Mas
`tests/integracion/esquema_test.go` con siete pruebas.

### Criterios, comprobados de forma independiente

| # | Criterio | Resultado |
|---|---|---|
| 1 | `make reset-test` y luego `make test` | Los dos en 0: las migraciones se aplican sobre una base vacia |
| 2 | `make migrate` dos veces | Las dos en 0. La segunda: "la base ya estaba en la version 4, nada que aplicar" |
| 3 | `make check` | **0** |
| 4 | Las pruebas del punto 6 | Las siete pasan, `catalogo/tests/integracion ok` |
| 5 | Las 14 tablas | La prueba `TestEsquema_CatorceTablas` exige que existan las 14 y que no haya ninguna mas |
| 6 | Ninguna migracion con `INSERT` | `grep -i insert` sin coincidencias |

### Alcance de las pruebas

Lo facil al escribir pruebas de esquema es comprobar que la operacion devuelve error, lo
que pasaria igual si fallara por cualquier otro motivo. Estas exigen el SQLSTATE y el
nombre de la restriccion, asi que solo pasan si rechazo la restriccion que toca.

La de minimo y maximo es la mas completa: acepta los casos ordenado, igual y con cualquiera
de los dos ausente, comprueba que el ausente queda `NULL` y no cero, que es lo que el
enunciado pide expresamente, y rechaza 24 mayor que 12 exigiendo
`servicio_n2_minmax_ck`.

El trabajador quito a proposito esa restriccion y la prueba fallo con "la base acepto la
operacion". Despues la restauro; el orquestador comprobo que aparece una sola vez en
`00004_catalogo.sql`.

### Bloqueo del guardian al escribir una migracion

`goose` exige una seccion de bajada, y la de estas tablas dice `DROP TABLE`. El guardian
bloqueo el `heredoc` por ese texto. El trabajador escribio los mismos archivos con la
herramienta de escritura, sin ofuscar nada, lo reporto y pregunto si eso era aceptable.

Lo era, y la pregunta estaba bien hecha. El limite del enunciado es no **ejecutar** acciones
destructivas fuera del entorno de pruebas, y ese no se toco: ningun `DROP` se ejecuto fuera
de `catalogo-test`. Como la duda es razonable y volveria a aparecer, se resolvio en el
contexto en vez de dejarla a criterio de cada sesion: `AGENTS.md` v5 y la decision D27.

### Pendiente

- Que el usuario responsable pertenezca a la seccion responsable se valida en el servidor,
  no con un disparador. Queda para cuando haya handlers.
- `internal/db` sigue sin pruebas propias. Las de esquema viven en `tests/integracion`.
