# Ciclo completo: tarea, cambio, control, fallo, correccion y nueva comprobacion

2026-10-04

El enunciado pide, en su seccion 4.3, la evidencia de **al menos un ciclo completo**: una
tarea, un cambio propuesto por la IA, la ejecucion de los controles, un fallo detectado, su
correccion y una nueva ejecucion satisfactoria.

Hubo varios. Aqui va el que mejor muestra para que sirve el harness, porque el fallo era de
seguridad, lo encontro un control y **no lo vio ninguna prueba automatica**. Al final se
listan los otros, que tambien son reales.

Ninguno se introdujo a proposito.

## El ciclo: cerrar sesion no cerraba la sesion

### 1. La tarea

[`05-autenticacion.md`](../../prompts/05-autenticacion.md): autenticacion local, sesiones y
cuentas de evaluacion, con las pruebas P01 y P02 del enunciado.

Entre otras cosas, el prompt pedia que «cerrar sesion invalida la credencial: volver a usar
la misma cookie no funciona», y su criterio 6 decia: «despues de cerrar sesion, esa misma
cookie devuelve 401».

### 2. El cambio que propuso la IA

El asistente implemento argon2id con sal por usuario, token opaco con solo su SHA-256 en la
base, los middlewares de sesion y de rol, y `sembrar-demo`. Escribio `TestP01` y `TestP02`,
y las dos pasaban.

Su reporte decia que los siete criterios estaban cumplidos.

### 3. La ejecucion de los controles

`make check` terminaba en 0. Las pruebas pasaban. Si el control hubiera sido solo eso, el
trabajo se habria aceptado.

El criterio 3 del prompt, en cambio, pedia comprobar contra el sistema levantado. Al
hacerlo:

```
$ curl -b cookies -X POST .../api/auth/logout
HTTP 415

$ curl -b cookies .../api/auth/me
HTTP 200      <-- la sesion seguia viva
```

### 4. El fallo

Cerrar sesion devolvia 415 y **no revocaba nada**.

La causa estaba en una decision de diseno que el propio orquestador habia aceptado (D26):
exigir `Content-Type: application/json` en todo `POST` como defensa contra peticiones
falsificadas. Cerrar sesion no lleva cuerpo, asi que una interfaz que llame con un `POST`
sin cuerpo recibe 415. La pantalla limpiaria su estado y el usuario veria el formulario de
inicio **con su sesion viva ocho horas mas**. En un equipo compartido, eso es el fallo
entero, y el enunciado pide expresamente que cerrar sesion invalide la credencial.

Lo importante: **la prueba no estaba mal escrita**. Su cliente mandaba el encabezado, asi
que el caso que fallaba nunca se ejecutaba. El defecto no estaba en el codigo del asistente
sino en el criterio del prompt, que era satisfacible de una forma que escondia el problema.

### 5. La correccion

Version revisada en [`05-autenticacion.v2.md`](../../prompts/05-autenticacion.v2.md), con el
problema observado escrito en su cabecera:

- se exige que cerrar sesion funcione **sin cuerpo y sin `Content-Type`**, con el comando
  exacto y sin ningun encabezado;
- se acota la exigencia de JSON a las peticiones que si traen cuerpo, porque de un `POST`
  de otro origen ya se encarga `SameSite=Lax`, que impide que la cookie viaje;
- se agrega el criterio inverso, que un cuerpo con un tipo que no sea JSON siga dando 415,
  para no quitar la defensa por error;
- se pide una prueba que llame como lo haria un navegador.

El asistente cambio la comprobacion para mirar la longitud del cuerpo y agrego dos
subpruebas a `TestP02`. Su **primera** correccion tambien fallo, porque uso una
comparacion que no distinguia los dos casos, y la detectaron los casos nuevos del
enrutador. La arreglo y volvio a pasar.

### 6. La nueva comprobacion

```
login:                                HTTP 200, cookie HttpOnly SameSite=Lax 28800s
me con cookie:                         HTTP 200
logout sin cuerpo y sin encabezados:   HTTP 204
la misma cookie despues:               HTTP 401
cuerpo con tipo que no es JSON:        HTTP 415
make check:                            exit 0
```

Comprobado por el orquestador contra el sistema levantado, no leyendo el reporte del
asistente.

### Que se aprende

- **Un criterio de aceptacion puede ser cierto y util y aun asi esconder el fallo.** El
  criterio decia que la cookie debia dar 401, pero no decia como se hace la llamada.
- **Las pruebas que escribe quien implementa tienden a probar el camino que implemento.** El
  control que encontro esto fue ejercer el sistema como lo hace un cliente real.
- Por eso los prompts siguientes piden un control positivo junto a los negativos, y que el
  asistente diga **que** rechazo cada accion y no solo que la rechazaron.

## Otros ciclos reales del mismo tipo

| Donde | Control | Fallo | Correccion |
|---|---|---|---|
| Arranque | Una sonda en modo headless, mirando el log del hook | `hooks/guard.sh` no estaba conectado: la accion se rechazaba sin registro. Al copiar el kit se perdieron las carpetas ocultas | Se restauro `.claude/`, con el hook declarado y los subagentes. [Detalle](00-hallazgo-hook-no-registrado.md) |
| Arranque | El mismo prompt, ejecutado una segunda vez | El `deny` se adelantaba al hook en las herramientas de archivo, asi que un intento quedaba impedido pero **sin registro** | Se movio la proteccion del Excel a la capa que si registra. [Detalle](00-hallazgo-capas-de-permisos.md) |
| Cuentas de evaluacion | Segunda corrida de `make seed-demo` | Una consulta quedo escrita sin un espacio y la segunda corrida fallaba. La primera no pasaba por ahi | El asistente lo corrigio y agrego una prueba de idempotencia que revierte su transaccion |
| Autorizacion por rol | Quitar a proposito el middleware de una ruta | `TestP03` fallo **solo** en esa ruta, mientras la subprueba generica seguia pasando | Confirmo que la prueba sirve. El middleware se restauro. [Salida](../pruebas/06-p03-sin-middleware.txt) |
| Esquema | Quitar a proposito una restriccion de la migracion | Su prueba fallo con «la base acepto la operacion» | Confirmo que la restriccion es la que rechaza. Se restauro |

Los dos ultimos no son fallos: son comprobaciones de que las pruebas detectan lo que dicen
detectar. Se incluyen porque una prueba que pasa igual con y sin el control que vigila no
prueba nada, y eso es facil de escribir sin darse cuenta.
