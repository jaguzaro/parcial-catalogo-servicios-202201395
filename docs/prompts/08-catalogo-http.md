# 08 - Catalogo de servicios por HTTP

| Campo | Valor |
|---|---|
| Fecha | 2026-10-04 |
| Herramienta | Claude Code `2.1.289`, modo headless `claude -p` |
| Modelo | `opus` |
| Ejecucion | `docs/evidencias/sesiones/<archivo>.json` y `.meta.txt` |
| Commit base | `fc330ee` |

## Objetivo

El mantenimiento del catalogo por HTTP: servicios de los dos niveles, catalogos de opciones,
busqueda, filtros, paginacion, ficha y asignacion de responsable. Con P09, P10 y P11
automatizadas.

## Contexto suministrado

- `AGENTS.md`, reglas permanentes, se carga solo. Secciones 7 y 9. La 9 trae los conteos que
  deja la importacion: 12, 46, 2, 5 y 11. No los rompas.
- `docs/diseno/reglas.md`: validaciones, baja logica, responsable de un servicio.
- `docs/diseno/arquitectura.md`: los endpoints de servicios y catalogos, con sus codigos de
  error, y como se paginan los listados.
- `docs/contexto/enunciado.md`, **solo las secciones 3.3 y 6**.

## Instrucciones

1. Servicios de nivel 1 y nivel 2: alta, consulta, modificacion y baja logica. Cada nivel 2
   pertenece a un nivel 1.
2. Catalogos de clase, criticidad y tipo: listar y mantener. En los formularios son opciones
   controladas, no texto libre.
3. Validaciones en el servidor:
   - campos obligatorios y referencias existentes;
   - si hay minimo y maximo, minimo no mayor que maximo. **Un dato ausente no se convierte
     en cero**;
   - codigo unico por nivel.
4. Busqueda por codigo y por nombre. Filtros por nivel 1, estado, clase, criticidad y tipo,
   y combinaciones de ellos. Listado paginado.
5. Ficha de un servicio de nivel 2 con todos sus atributos, su nivel 1, su seccion y su
   usuario responsable, y las incidencias que le toquen.
6. Asignacion de responsable: una seccion responsable y, opcionalmente, un usuario que
   **pertenezca a esa seccion**. Un usuario de otra seccion se rechaza.
7. `sembrar-demo` crea ademas **al menos tres asignaciones de responsable validas**, como
   pide `reglas.md` y la seccion 3.3 del enunciado. Sigue siendo idempotente, y si no hay
   servicios importados lo dice en vez de fallar en silencio.
8. Pruebas en `tests/integracion`:
   - `TestP09_...`: crear y editar un servicio con minimo mayor que maximo se rechaza. Y los
     casos que si se aceptan: iguales, y con uno de los dos ausente.
   - `TestP10_...`: busqueda y filtros con resultados coherentes. Usa los datos reales de la
     importacion, con conteos esperados concretos.
   - `TestP11_...`: asignar como responsable a un usuario de otra seccion se rechaza.
   - La autorizacion de las rutas nuevas entra en `TestP03`, que recorre todas las de
     escritura. Que siga pasando y que cubra las nuevas.

## Restricciones

- No implementes la interfaz. Esto es solo la API.
- No toques `hooks/`, `.claude/`, `data/`, `docs/contexto/` ni `docs/diseno/`.
- No cambies el importador salvo que haga falta para las asignaciones de `sembrar-demo`, y
  si lo cambias dilo.
- Si hace falta un cambio de esquema, agrega una migracion nueva.
- No leas el archivo de entorno. Nunca ejecutes `make reset-dev` ni `down -v` fuera de
  `catalogo-test`.

## Salida esperada

El codigo, las pruebas y un reporte final con: los endpoints nuevos y quien puede
llamarlos, cuantas rutas de escritura cubre ahora `TestP03`, los comandos con su codigo de
salida, el resultado de P09, P10 y P11, y lo pendiente.

## Criterio de aceptacion

1. `make check` en 0, con `TestP09`, `TestP10` y `TestP11` entre las que pasan, y `TestP03`
   a `TestP08` **siguiendo en verde**.
2. `TestP03` cubre tambien las rutas de escritura nuevas. Di cuantas son ahora.
3. Sobre la base ya importada: un servicio con minimo 24 y maximo 12 se rechaza; con
   minimo 12 y maximo 12 se acepta; con solo minimo se acepta y el maximo queda nulo, no
   cero.
4. Filtrar por un nivel 1 concreto devuelve solo sus hijos, y el total del listado coincide
   con el numero de hijos de ese nivel 1.
5. Buscar un codigo que existe devuelve ese servicio; buscar un texto que no existe devuelve
   cero resultados y no un error.
6. Asignar un usuario de una seccion distinta a la responsable devuelve error, y el mensaje
   dice por que.
7. `make seed-demo` deja al menos 3 asignaciones de responsable validas, y dos corridas
   seguidas terminan en 0 sin duplicarlas.
8. Despues de todo, los conteos del catalogo siguen siendo 12 y 46.

---

> Lo de arriba es el texto que se envio, sin retocar. Lo de abajo se escribio despues.

## Resultado

**Aceptado.** Sesion:
[`20261004-150315-08-catalogo-http.json`](../evidencias/sesiones/20261004-150315-08-catalogo-http.json),
86 turnos con opus.

13 rutas nuevas de escritura, asi que `TestP03` cubre ahora **38** y sigue pasando. P01 a
P11 completas y en verde.

### Comprobacion independiente con el sistema levantado

Sobre la base de evaluacion, con la cuenta de administrador:

```
P09  minimo 24 y maximo 12
     422 MINIMO_MAYOR_QUE_MAXIMO
     "El minimo (24) no puede ser mayor que el maximo (12). No se guardo nada."

P10  filtro por el nivel 1 SE.06 -> total 10, todos del mismo nivel 1
     busqueda sin resultados      -> 200 con total 0, no un error
     total del catalogo           -> 46

P11  usuario de otra seccion
     422 RESPONSABLE_FUERA_DE_SECCION
     "El usuario usuario.b pertenece a la seccion SEC-B (id 8) y la seccion
      responsable pedida es DEMO (id 1): el usuario responsable tiene que
      pertenecer a la seccion responsable."

make seed-demo, dos corridas: 0 y 0. Deja 3 asignaciones con seccion y usuario.
Conteos despues de todo: 12 y 46.
make check: exit 0
```

El criterio 7 lo ejecute yo sobre la base de evaluacion, que el asistente dejo pendiente a
proposito. Los registros que cree para probar P11 quedaron dados de baja.

Los mensajes de error no solo dan un codigo: nombran el dato, los dos lados del conflicto y
sus identificadores. Eso es lo que el enunciado pide cuando dice "mensaje comprensible".

### Decisiones del asistente, revisadas

- **Negativos rechazados.** `reglas.md` dice "No se admiten negativos ni se pide entero: el
  enunciado no lo dice", que se puede leer de dos maneras. Lo tomo literal: un negativo da
  422. Se acepta asi, y queda anotado que revertirlo es quitar una comprobacion.
- **Edicion parcial:** un campo que no viene se queda como estaba y `null` lo borra.
  Necesario para que la validacion mire el estado final y no solo lo que llego.
- **Quitar responsable** a un servicio dado de baja si se permite; asignarselo no.
- **La ficha** muestra las incidencias de la ultima importacion completada.

### Lo que el asistente reporto por su cuenta

Las tres las reporto sin que se le preguntara:

- Guardo una vez una salida fuera de la carpeta del proyecto. El guardian no lo bloqueo,
  pero va contra el limite, y lo dijo.
- El guardian le bloqueo tres comandos por los `//` de comentarios de Go y los `/` de
  `sed`. No intento rodearlo: uso la herramienta de edicion.
- Uso `python3` en la maquina para editar un archivo, y señalo que la regla de "todo en
  contenedores" podia prohibirlo.

Ese ultimo punto era una ambiguedad real de `AGENTS.md`: la regla apunta a construir y
ejecutar el proyecto, no a editar texto. Aclarado en la version v8 del contexto.

### Pendiente

- La estructura de demostracion tiene una sola seccion y `reglas.md` pide dos. Venia de
  antes de esta tarea.
- Falta el listado de importaciones por HTTP, que la interfaz necesita para mostrar el
  resultado de la importacion.
- El candado de importacion concurrente sigue sin prueba.
