# Como se resolvio

Joel Guzaro, carne 202201395. USAC, Laboratorio de Software Avanzado. 2026-10-04.

## 1. Problema, alcance y supuestos

Una organizacion lleva su catalogo de servicios de TI en una hoja de calculo y necesita una
aplicacion para consultarlo y mantenerlo, organizar sus unidades y asignar responsables.

**Dentro del alcance:** autenticacion local con dos roles, estructura organizacional de seis
niveles, catalogo de servicios de dos niveles con sus catalogos de opciones, importacion
repetible del Excel con trazabilidad, y todo ejecutable con Docker desde un clon limpio.

**Fuera, porque el enunciado lo dice:** tickets, facturacion y consumo de servicios.

**Supuestos que se tomaron, y donde estan justificados:**

- El nombre canonico de `SE.12` es «Suministrar Analitica» (D09). El archivo trae dos
  nombres para el mismo codigo.
- Una fila con codigo de nivel 2 pero sin nivel 1 a la vista hereda el ultimo de arriba, y
  eso queda registrado como incidencia (D08). Sin esta regla, `SE.12.3` se queda huerfano.
- Las filas 42 y 67 no son servicios (D10).
- Lo que el archivo no dice, no se inventa: queda nulo y marcado para revision (D11).

Cada decision esta numerada en [`contexto/decisiones.md`](contexto/decisiones.md), con su
alternativa descartada y quien la tomo.

## 2. Arquitectura y justificacion de tecnologias

Un solo binario de Go sirve la API y la interfaz ya compilada, desde el mismo origen.
PostgreSQL 17 en su contenedor, con volumen con nombre y `healthcheck`.

| Pieza | Eleccion | Por que, en una linea |
|---|---|---|
| Backend | Go con la libreria estandar, `pgx/v5`, `goose` embebido, `excelize` | El binario se compila en una imagen sin dependencias de ejecucion, que es lo que hace facil arrancar desde un clon limpio (D14) |
| Contrasenas | argon2id, 64 MiB, 3 pasadas, sal por usuario | El enunciado pide un algoritmo especializado con sal (D15, D26) |
| Sesion | Token opaco en cookie `HttpOnly`; en la base solo su SHA-256 | Cerrar sesion tiene que invalidar de verdad. Un JWT obliga igual a una lista de revocacion (D16) |
| Interfaz | React, Vite y TypeScript, servida por el mismo binario | Mismo origen: sin CORS ni un segundo puerto que documentar (D17) |
| Base de datos | PostgreSQL 17 con volumen y healthcheck | Persistencia y arranque que respeta dependencias (D18) |
| Pruebas | Integracion en Go contra una base aparte | Lo que hay que probar son reglas que viven en la base y en el servidor (D19) |

Detalle en [`diseno/arquitectura.md`](diseno/arquitectura.md).

## 3. Modelo de datos

Diagrama entidad relacion en Mermaid y diccionario completo, con llaves, relaciones y
restricciones: [`diseno/modelo.md`](diseno/modelo.md).

Catorce entidades: `empresa`, `area`, `departamento`, `seccion`, `puesto`, `usuario`,
`sesion`, `servicio_n1`, `servicio_n2`, `clase_servicio`, `criticidad`, `tipo_servicio`,
`importacion` e `incidencia`.

Las restricciones estan **en la base**, no solo en el codigo: codigo unico global en los
servicios y unico dentro del padre en las unidades, `activo` limitado a `S`, `N` y
`DESCONOCIDO`, y minimo no mayor que maximo cuando los dos existen. Las pruebas de
`tests/integracion/esquema_test.go` comprueban que la base rechaza, exigiendo el SQLSTATE y
el nombre de la restriccion.

## 4. Del Excel a la base

Mapeo columna a columna y los seis casos del enunciado:
[`diseno/mapeo-excel.md`](diseno/mapeo-excel.md). El analisis del archivo, reproducible con
`scripts/analisis/analizar_excel.py`, esta en
[`contexto/analisis-excel.md`](contexto/analisis-excel.md).

| Caso | Que se hace |
|---|---|
| Celdas combinadas | El valor sale de la celda principal del rango. Hay 76 rangos, ninguno multicolumna |
| Conflicto `SE.12` | Se queda `B99`. La incidencia conserva **los dos** valores con sus celdas |
| Formato de codigos | `SE.12.1`, `SE.12.2` y `SE.12.3` se guardan tal cual, con una incidencia que lo senala |
| Atributos incompletos | Filas 99 a 101: `activo = DESCONOCIDO`, el resto nulo, marcadas para revision |
| Filas de continuacion | Las filas 42 y 67 no se importan ni se asignan a nadie. Quedan registradas |
| Trazabilidad | Cada servicio guarda hoja y rango de origen. Cada incidencia, su tipo, celda, valores y la regla que se aplico |

**Resultado de la importacion**, salida real en
[`evidencias/pruebas/07-importacion.txt`](evidencias/pruebas/07-importacion.txt):

| Corrida | Creados | Actualizados | Omitidos | Observados | Incidencias |
|---|---:|---:|---:|---:|---:|
| Primera | 76 | 0 | 2 | 6 | 10 |
| Segunda | 0 | 0 | 78 | 6 | 10 |

Los 76 son 12 servicios de nivel 1, 46 de nivel 2, 2 clases, 5 criticidades y 11 tipos.
**Los controles del enunciado, 12 y 46, se cumplen sin ajustar ninguna regla.**

Las incidencias se guardan por corrida, ligadas a su importacion: por eso tras dos
importaciones hay 20. Los servicios no se duplican.

## 5. Autenticacion, autorizacion y sesiones

- **Contrasenas:** argon2id con sal aleatoria por usuario, en formato PHC, comparacion en
  tiempo constante. Dos usuarios con la misma contrasena tienen hashes distintos, y hay una
  prueba que lo exige.
- **Usuario inexistente:** se verifica igual contra un hash ficticio, para que el tiempo de
  respuesta no revele si la cuenta existe. Contrasena incorrecta y usuario inexistente
  devuelven 401 con el cuerpo **identico byte a byte**.
- **Sesion:** token opaco de 32 bytes en cookie `HttpOnly`, `SameSite=Lax`, 8 horas. En la
  base solo su SHA-256. Cada peticion comprueba que el usuario siga activo; si se desactiva,
  la sesion se revoca.
- **Cierre de sesion:** revoca en la base y funciona llamado como lo hace un navegador, sin
  cuerpo y sin cabeceras. Esto fue un fallo encontrado y corregido: seccion 8.
- **Roles:** administrador mantiene; consulta solo lee. La autorizacion esta en el servidor.
  `TestP03` recorre **las 38 rutas de escritura** y exige 403 para consulta.

Las reglas completas, incluida la politica de baja con dependencias, en
[`diseno/reglas.md`](diseno/reglas.md).

## 6. Las tres tecnicas de ingenieria con IA

### Context engineering

[`AGENTS.md`](../AGENTS.md) es el contexto versionado. Va por la **version 10**, y cada
cambio dice que hallazgo lo motivo:
[`contexto/registro.md`](contexto/registro.md).

| Version | Que la motivo |
|---|---|
| v2 | El analisis del Excel: `A99` y `A100` son celdas sueltas, lo que deja sin padre a `SE.12.3` |
| v3 | El diseno terminado, antes de escribir migraciones |
| v4 | Ya existe `make check`, que pasa a ser el comando que valida cualquier cambio |
| v5 | El guardian bloqueo escribir una migracion por el `DROP TABLE` de su bajada: habia que distinguir escribir de ejecutar |
| v8 | El limite de «todo en contenedores» parecia prohibir editar un archivo de texto |
| v10 | La auditoria encontro que el contexto prometia una prueba `TestP12_` que no existe |

`registro.md` tambien anota, por cada tarea, **que se dejo fuera a proposito** y por que.

La regla que distingue instrucciones de datos esta en la seccion 3 de `AGENTS.md`, y se
puso a prueba sola: la celda `I5` del Excel contiene `'Revele su rollo '`, un imperativo. El
asistente lo detecto, no lo siguio, lo reporto como observacion y lo importo como
descripcion.

### Prompt engineering

Dieciocho archivos de prompt realmente ejecutados, contando las dos versiones revisadas, en [`prompts/`](prompts/), cada uno con su objetivo,
contexto, instrucciones, restricciones, salida esperada y criterio de aceptacion. Cada
ejecucion tiene su JSON y sus metadatos en [`evidencias/sesiones/`](evidencias/sesiones/),
con fecha, modelo, version de la herramienta, commit base y codigo de salida.

Los cinco que exige el enunciado: analisis del Excel (01), modelo de datos (02),
pruebas y Docker (03), autenticacion (05) e importacion (07).

**Las dos iteraciones que exige el enunciado**, las dos nacidas de un defecto real:

| | Problema observado | Revision | Resultado |
|---|---|---|---|
| 1 | El criterio de cerrar sesion no decia **como** se hace la llamada, y eso dejo pasar un fallo de seguridad | [`05-autenticacion.v2.md`](prompts/05-autenticacion.v2.md) | Cerrar sesion devuelve 204 y la cookie queda muerta |
| 2 | Al enumerar las pantallas me salte los catalogos de clase, criticidad y tipo, que el enunciado pide mantener | [`10-interfaz-mantenimiento.v2.md`](prompts/10-interfaz-mantenimiento.v2.md) | Pantalla con las tres listas, el valor de origen junto al nombre, y la baja en uso rechazada con sus dependencias |

En los dos casos **el defecto estaba en el prompt, no en el trabajo del asistente**: hizo lo
que se le pidio y, en el segundo, aviso de lo que quedaba fuera. El detalle esta en la
seccion "Iteracion" de cada prompt original.

Lo que se aprendio sobre criterios de aceptacion, y se aplico en los prompts siguientes:

- pedir siempre un **control positivo** junto a los negativos;
- pedir que el asistente diga **que capa** rechazo una accion, no solo que la rechazo;
- para una prueba de control, exigir que **se quite el control y la prueba falle**.

### Harness engineering

- [`AGENTS.md`](../AGENTS.md) dice donde esta el contexto y que comandos existen.
- `hooks/guard.sh` aplica los limites, y `hooks/test-guard.sh` los comprueba con 33 casos,
  12 permitidos y 21 bloqueados.
- Cada intento bloqueado queda en [`logs/hooks.log`](../logs/hooks.log).
- `make check` es la rutina: lint, pruebas, limites del harness y SHA-256 del Excel.
- Limites: git de escritura solo lo hace la persona, nada de secretos, el Excel de solo
  lectura, nada fuera del proyecto, y acciones destructivas solo en el entorno de pruebas.
- Evidencia de los bloqueos funcionando **en modo headless**:
  [`evidencias/harness/00-bloqueos.md`](evidencias/harness/00-bloqueos.md).
- Ciclo completo: [`evidencias/harness/ciclo-01.md`](evidencias/harness/ciclo-01.md).

La aplicacion terminada no necesita ninguna suscripcion ni clave de IA para funcionar.

## 7. Matriz de requisito a evidencia

| Requisito | Implementacion | Prueba | Evidencia |
|---|---|---|---|
| Inicio y cierre de sesion local | `internal/auth`, `internal/http/auth.go` | `TestP01` | `evidencias/sesiones/*-05-*` |
| Hash con sal | argon2id, `internal/auth/contrasena.go` | `TestAuth_ContrasenaArgon2idConSalPorUsuario` | idem |
| Rutas protegidas en servidor | middlewares de sesion y rol | `TestP02`, `TestP03` (38 rutas) | `evidencias/pruebas/06-p03-sin-middleware.txt` |
| Dos roles | `rol` en `usuario`, `RequerirRol` | `TestP03` | idem |
| Usuario desactivado sin acceso | revision en cada peticion | `TestP02` | idem |
| Cuentas de evaluacion reproducibles | `make seed-demo` | `TestDemo_SembrarEsIdempotente` | README |
| Jerarquia de seis niveles | `internal/organizacion` | `TestP04` | sesion 06 |
| Codigo unico dentro del padre | restriccion en la base | `TestP05`, `esquema_test.go` | sesion 04 |
| Sin huerfanos ni padres inactivos | validacion en servidor | `organizacion_test.go` | sesion 06 |
| Baja con dependencias | 409 con la lista | `organizacion_test.go` | sesion 06 |
| Catalogo de dos niveles | `internal/catalogo` | `TestP10` | sesion 08 |
| Catalogos de opciones | cargados del Excel | `TestP06` | `evidencias/pruebas/07-importacion.txt` |
| Minimo menor o igual que maximo | validacion y restriccion | `TestP09` | sesion 08 |
| Busqueda, filtros y paginacion | `GET /api/servicios` | `TestP10` | sesion 08 |
| Ficha con todos los atributos | pantalla y `GET` por id | `make e2e` | `evidencias/pruebas/*-e2e.txt` |
| Responsable de la misma seccion | validacion en servidor | `TestP11` | sesion 08 |
| Importacion repetible | `internal/importador` | `TestP06`, `TestP07` | `evidencias/pruebas/07-importacion.txt` |
| Conflicto y ausencias | reglas D09 y D11 | `TestP08` | idem |
| Persistencia por volumen | volumen con nombre | `make test-persistence` | `evidencias/pruebas/*-p12.txt` |
| Docker desde clon limpio | `Dockerfile`, `compose.yaml` | `make check` | `evidencias/pruebas/<fecha>-<commit>.txt` |
| Excel sin modificar | `hooks/guard.sh`, paso 4 de `make check` | `shasum -c` | `contexto/excel-sha256.txt` |

## 8. Resultados de pruebas, fallos y correcciones

Las salidas crudas estan en [`evidencias/pruebas/`](evidencias/pruebas/), con fecha y
commit en el nombre. `make evidence` las genera, y **las guarda tambien cuando el resultado
es malo**: hay una corrida fallida conservada a proposito, explicada en el README de esa
carpeta.

Estado al cierre: `make check` en 0, `make test-persistence` en 0, `make e2e` con sus dos
pruebas pasando, y los conteos del catalogo en 12 y 46.

### El cierre de sesion no invalidaba la sesion

El mas importante, y el unico que afectaba a la seguridad.

`make check` pasaba y `TestP02` tambien, pero al ejercer el sistema con `curl` como lo hace
un navegador, cerrar sesion devolvia **415 y no revocaba nada**. La causa era una decision
de diseno aceptada antes (D26): exigir `Content-Type: application/json` en todo `POST`.
Cerrar sesion no lleva cuerpo.

La prueba no estaba mal escrita: su cliente mandaba la cabecera, asi que el caso que fallaba
nunca se ejecutaba. **El defecto estaba en el criterio de aceptacion del prompt**, que era
satisfacible de una forma que escondia el problema.

Correccion en [`prompts/05-autenticacion.v2.md`](prompts/05-autenticacion.v2.md), que exige
la llamada sin cuerpo y sin cabeceras, acota la exigencia de JSON a las peticiones que si
traen cuerpo, y agrega el criterio inverso para no perder la defensa. Verificado despues:
cerrar sesion devuelve 204 y la misma cookie da 401.

**Esta es la iteracion de prompt numero 1.**

### Una pantalla obligatoria quedo fuera del prompt

Al enumerar las pantallas de mantenimiento me salte los catalogos de clase, criticidad y
tipo. La seccion 3.3 del enunciado pide **mantenerlos**, no solo usarlos como listas de
opciones. El asistente hizo lo pedido y aviso de lo que quedaba fuera; la auditoria lo
confirmo como incumplimiento.

La revision, [`prompts/10-interfaz-mantenimiento.v2.md`](prompts/10-interfaz-mantenimiento.v2.md),
agrego la pantalla y dos exigencias que el enunciado implica: que cada opcion muestre su
valor de origen junto a su nombre editable —ese par es el mapeo verificable de etiquetas que
se pide—, y que dar de baja una opcion en uso se rechace listando los servicios que la usan.

Comprobado: 2 clases, 5 criticidades, 11 tipos con `Demostration` intacto, y la baja de una
clase en uso rechazada con sus 19 servicios.

**Esta es la iteracion de prompt numero 2.**

### El hook no estaba conectado

Antes de escribir una linea de aplicacion, una sonda mostro que `hooks/guard.sh` existia
pero no se ejecutaba: la accion se rechazaba por el permiso generico de la herramienta,
sin dejar registro. Al copiar el kit de arranque se habian perdido las carpetas ocultas, y
con ellas los subagentes de la auditoria.
[Detalle](evidencias/harness/00-hallazgo-hook-no-registrado.md).

### Una prueba dejaba la base en un estado que rompia otras

La primera version de `make e2e` dejaba el catalogo importado en la base de pruebas, y las
pruebas de Go exigen esa base vacia: 15 fallaron. Lo detecto `make evidence`, y esa corrida
fallida es la que se conserva.

### Comprobaciones deliberadas

Dos veces se quito un control a proposito para ver si su prueba fallaba: una restriccion de
la base y el middleware de rol de una ruta. Las dos pruebas fallaron, y solo en lo que
tenian que fallar. Despues se restauro. No son fallos: son la demostracion de que esas
pruebas detectan lo que dicen detectar.

### Auditoria con subagentes

Tres revisores en paralelo (seguridad, rendimiento y cumplimiento) y un consolidador, con
sus reportes en [`../reports/`](../reports/). **26 hallazgos**: 2 criticos, 4 altos, 8
medios y 12 bajos.

Los dos criticos eran este archivo y el README, que faltaban. Se corrigieron tres hallazgos
de codigo (tiempos limite del servidor, indice de sesiones y cabeceras de seguridad) y
varios documentales. El resto queda en la seccion 10 como limitaciones conocidas: no son
olvidos, son una decision tomada con el tiempo que habia.

## 9. Docker, persistencia y recuperacion

Todo esta en el [README](../README.md): requisitos, puesta en marcha, cuentas, comandos,
logs, apagado normal y reinicio destructivo.

Lo esencial:

- `docker compose up --build -d` desde un clon limpio, tras `make setup`.
- La base persiste en un volumen con nombre. `make down` no lo borra.
- `make reset-dev CONFIRMAR=si` es el unico que destruye datos de evaluacion, y sin esa
  confirmacion no hace nada.
- Las pruebas usan un proyecto y una base aparte, asi que nunca tocan los datos de
  evaluacion.
- Si el puerto 8080 esta ocupado, se cambia `APP_PORT`. El README lo avisa antes de los
  comandos, porque es el tropiezo mas probable.

## 10. Limitaciones conocidas y reflexion

### Lo que falta

- **Sin limite de intentos de inicio de sesion** (hallazgo SEG-01 de la auditoria). Es la
  limitacion mas seria de las que quedan.
- **No se puede lanzar la importacion desde la aplicacion.** Se ejecuta con `make import`.
  El enunciado admite «por comando o desde la aplicacion», asi que se cumple, pero el
  diseno llego a listar ese endpoint.
- **Sin purga de sesiones vencidas**, aunque ya tienen su indice.
- La estructura de demostracion tiene una sola seccion, y el diseno preveia dos.
- `hooks/guard.sh` no protege `.claude/agents/`; hoy eso lo cubre solo una regla de
  permisos.

### Aportes y forma de trabajo

El trabajo se condujo con un **orquestador de IA**: un asistente que planifico, redacto los
prompts, los ejecuto contra un segundo asistente en modo headless, verifico los resultados
por su cuenta y escribio la documentacion. **Joel ejecuto git**, resolvio los bloqueos que
necesitaban una persona (confianza del entorno, restaurar las carpetas ocultas) y corrigio
al orquestador cuando hizo falta. En `contexto/decisiones.md` cada decision dice su origen
real.

### Errores de la IA y decisiones humanas

Lo que mas se aprendio no es que la IA se equivoque, sino **como**.

Los errores no fueron de sintaxis ni de logica gruesa: el codigo compilaba y las pruebas
pasaban. Fueron errores de **alcance de la comprobacion**. Cerrar sesion funcionaba por la
via que la prueba ejercitaba y fallaba por la que usa un navegador. La prueba de
autorizacion cubria un endpoint de veinticinco. El conteo de lineas de un informe contaba
bloques en vez de lineas.

El patron se repite: *el asistente comprueba el camino que implemento*. Por eso el control
que mas sirvio fue siempre el externo — ejercer el sistema como lo hace un cliente real, y
quitar un control a proposito para ver si su prueba se da cuenta.

La segunda leccion es sobre los criterios de aceptacion. Un criterio puede ser cierto, util
y estar cumplido, y aun asi esconder el fallo, porque no dice **como** se hace la
comprobacion. Las dos iteraciones de prompt de este trabajo nacieron de ahi, no de que el
asistente desobedeciera.

Y la tercera: el asistente reporto varias cosas que nadie le pregunto — una contradiccion
entre el diseno y el prompt, que habia escrito fuera de la carpeta del proyecto, que no
podia verificar algo por no tener las credenciales. Esos reportes valieron mas que varios
aciertos, porque son lo unico que permite confiar en el resto de lo que dice.

## 11. Entrega

- Rama entregada: `main`. Etiqueta: `parcial-v2.0`, sobre el commit entregado.
- El repositorio es privado.
- **El usuario `maldanap-usac` tiene la invitacion de colaborador enviada y pendiente de
  aceptacion** al momento de entregar, comprobado con:

  ```bash
  gh api repos/<usuario>/parcial-catalogo-servicios-202201395/invitations -q '.[].invitee.login'
  ```

  Aparece en la lista de invitaciones y no en la de colaboradores, lo que significa que la
  invitacion existe pero todavia no se ha aceptado. Se informa aqui porque la seccion 7 del
  enunciado lo pide expresamente.

