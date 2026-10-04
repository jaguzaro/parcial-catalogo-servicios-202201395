# 05 - Autenticacion, sesiones y cuentas de evaluacion

| Campo | Valor |
|---|---|
| Fecha | 2026-10-04 |
| Herramienta | Claude Code `2.1.289`, modo headless `claude -p` |
| Modelo | `opus` |
| Ejecucion | `docs/evidencias/sesiones/<archivo>.json` y `.meta.txt` |
| Commit base | `9a90098` |

## Objetivo

Que alguien pueda iniciar y cerrar sesion, que el servidor proteja lo que tiene que
proteger, y que exista un comando reproducible que cree las cuentas de evaluacion. Con las
pruebas P01 y P02 del enunciado automatizadas.

El mantenimiento de la jerarquia y de usuarios es otra tarea. Aqui solo la autenticacion y
lo minimo para poder probarla.

## Contexto suministrado

- `AGENTS.md`, reglas permanentes, se carga solo. Secciones 7 y 9.
- `docs/diseno/reglas.md`, sobre todo "Autenticacion y sesiones", "Usuario desactivado con
  la sesion abierta" y "Autorizacion por rol". Trae los parametros ya decididos.
- `docs/diseno/arquitectura.md`, los endpoints de autenticacion y el formato de error.
- `docs/contexto/enunciado.md`, **solo las secciones 3.1 y 6**: lo que se exige de
  autenticacion y los escenarios que hay que automatizar.

## Instrucciones

1. **`internal/auth`**:
   - argon2id con los parametros del diseno y sal aleatoria por usuario. Cada usuario tiene
     su propia sal, aunque dos usen la misma contrasena.
   - Verificacion en tiempo constante.
   - Token de sesion opaco, aleatorio, y en la base **solo su SHA-256**. Nunca el token.
   - Middleware de sesion y middleware de rol.
2. **Endpoints**, segun `arquitectura.md`: iniciar sesion, cerrar sesion y consultar quien
   soy. La cookie va `HttpOnly`, con `SameSite=Lax` y con `Secure` segun la variable de
   entorno.
3. **Reglas que el servidor tiene que cumplir:**
   - Credenciales incorrectas y usuario inexistente responden lo mismo, para no revelar que
     usuarios existen.
   - Un usuario desactivado no puede iniciar sesion, y si ya tenia la sesion abierta deja
     de poder usarla. Cada peticion comprueba que siga activo.
   - Cerrar sesion invalida la credencial: volver a usar la misma cookie no funciona.
   - Ninguna respuesta de la API incluye nunca el hash ni la sal.
4. **`sembrar-demo`** (y `make seed-demo`): crea la estructura minima que necesita un
   usuario y las dos cuentas de evaluacion, administrador y consulta, leyendo usuario y
   contrasena de las variables de entorno. Idempotente: ejecutarlo dos veces no duplica
   nada ni falla. Que imprima que creo y que ya existia, sin imprimir contrasenas.
5. **Pruebas en `tests/integracion`**, con el identificador del escenario en el nombre:
   - `TestP01_...`: inicio de sesion valido e invalido. Contrasena incorrecta y usuario
     inexistente dan la misma respuesta.
   - `TestP02_...`: los tres casos del enunciado. Una operacion protegida sin sesion; la
     misma cookie despues de cerrar sesion; y un usuario que se desactiva teniendo la
     sesion abierta.
   - Una prueba de que las contrasenas se guardan con argon2id y con sal por usuario: el
     valor guardado empieza por `$argon2id$` y **dos usuarios con la misma contrasena
     tienen hashes distintos**.
   - Una prueba de que la respuesta de quien soy no trae el hash.

## Restricciones

- No implementes el mantenimiento de la jerarquia ni de usuarios, ni el catalogo, ni el
  importador, ni pantallas. Si para probar necesitas filas, insertalas desde la prueba.
- Nada de proveedores externos de identidad. La validacion es contra la propia base.
- No toques `hooks/`, `.claude/`, `data/`, `docs/contexto/`, `docs/diseno/` ni las
  migraciones ya escritas. Si hace falta un cambio de esquema, agrega una migracion nueva.
- No leas el archivo de entorno. Las variables se documentan en el de ejemplo.
- Ninguna contrasena real en el codigo ni en el repositorio. Las de las pruebas se generan
  o se leen de variables.
- Nunca ejecutes `make reset-dev` ni `down -v` fuera de `catalogo-test`.

## Salida esperada

El codigo, las pruebas, y un reporte final con: los endpoints que quedaron, los comandos
ejecutados con su codigo de salida, el resultado de P01 y P02, y lo que quede pendiente.

## Criterio de aceptacion

1. `make check` termina en 0, con `TestP01` y `TestP02` entre las pruebas que pasan.
2. `make seed-demo` dos veces seguidas: las dos en 0, y la segunda no duplica ni falla.
3. Sobre el sistema levantado, iniciar sesion con la cuenta de administrador devuelve 200 y
   una cookie con `HttpOnly` y `SameSite=Lax`.
4. La misma peticion con la contrasena cambiada devuelve 401, y con un usuario que no existe
   devuelve 401 **con el mismo cuerpo**.
5. Consultar quien soy sin cookie devuelve 401. Con la cookie devuelve 200 y un cuerpo que
   no contiene ni `hash` ni `sal` ni `argon2`.
6. Despues de cerrar sesion, esa misma cookie devuelve 401.
7. `grep -ri 'argon2\\$\\|password.*=.*"' --include='*.go'` no muestra ninguna contrasena
   escrita en el codigo.

---

> Lo de arriba es el texto que se envio, sin retocar. Lo de abajo se escribio despues.

## Resultado

**Rechazado en la v1.** Sesion:
[`20261004-101228-05-autenticacion.json`](../evidencias/sesiones/20261004-101228-05-autenticacion.json),
62 turnos con opus.

El trabajo quedo casi completo: argon2id con sal por usuario, token opaco con solo su
SHA-256 en la base, middlewares, `sembrar-demo` idempotente y P01 y P02 pasando. Pero al
comprobar el criterio 6 contra el sistema levantado aparecio un defecto que las pruebas no
veian. Ver la seccion de iteracion.

Lo que si quedo bien y se comprobo aparte:

| Criterio | Resultado |
|---|---|
| 1. `make check` en 0 con P01 y P02 | Cumplido |
| 2. `make seed-demo` dos veces | Cumplido, la segunda sin duplicar |
| 3. Login con 200 y cookie | Cumplido: `HttpOnly`, `SameSite=Lax` y `Max-Age=28800`, o sea las 8 horas del diseno |
| 4. Contrasena incorrecta y usuario inexistente | Cumplido, 401 las dos con el cuerpo **identico byte a byte** |
| 5. Quien soy sin cookie 401, con cookie 200 y sin hash | Cumplido |
| 6. La cookie despues de cerrar sesion | **No cumplido.** Ver la iteracion |
| 7. Sin contrasenas en el codigo | Cumplido |

Dos cosas que hizo bien:

- Contra un usuario que no existe verifica igual contra un hash ficticio, para que el
  tiempo de respuesta no delate si la cuenta existe. No se lo pedi.
- Encontro que `reglas.md` y este prompt no decian lo mismo sobre `sembrar-demo`: el diseno
  pide que tambien cree tres asignaciones de responsable y que falle sin servicios
  importados. Lo reporto en vez de elegir en silencio, y siguio el prompt, que es lo que
  manda en la tarea actual. Las asignaciones quedan para cuando exista el importador.

Tambien se le rompio su propia consulta al editar, `FROMempresa` sin espacio, y la segunda
corrida de `make seed-demo` fallo. Lo arreglo y agrego una prueba de idempotencia que
revierte la transaccion. Ese ciclo de control, fallo y correccion esta en su sesion.

## Iteracion

**Problema observado.** El criterio 6 decia "despues de cerrar sesion, esa misma cookie
devuelve 401", pero no decia **como** se hace esa llamada. El trabajador lo cumplio
mandando `Content-Type: application/json`, que es legitimo con lo que pedi. Comprobando a
mano salio esto:

```
$ curl -b cookies -X POST .../api/auth/logout
HTTP 415
$ curl -b cookies .../api/auth/me
HTTP 200      <-- la sesion seguia viva
```

La causa es la decision D26, que exige `Content-Type: application/json` en todo `POST`.
Cerrar sesion no lleva cuerpo, asi que un navegador que llame con un `POST` sin cuerpo
recibe 415 y **la sesion no se revoca**. La interfaz limpiaria su estado y el usuario veria
la pantalla de inicio con su sesion viva ocho horas mas. El enunciado pide expresamente que
cerrar sesion invalide la credencial.

El defecto no estaba en la implementacion sino en mi criterio: era satisfacible de una forma
que escondia el fallo.

**Cambio en el prompt.** Version revisada en
[`05-autenticacion.v2.md`](05-autenticacion.v2.md):

- se exige que cerrar sesion funcione **sin cuerpo y sin `Content-Type`**, con el `curl`
  exacto y sin ningun `-H`;
- se acota la exigencia de JSON a las peticiones que si traen cuerpo, dejando dicho que de
  los `POST` de otro origen se encarga `SameSite=Lax`, que impide que la cookie viaje;
- se agrega el criterio inverso, que un cuerpo con un tipo que no sea JSON siga dando 415,
  para no quitar la defensa por error;
- se pide una prueba que llame como lo haria un navegador.

**Resultado comprobado.** Sesion de la v2:
[`20261004-102452-05-autenticacion.v2.json`](../evidencias/sesiones/20261004-102452-05-autenticacion.v2.json),
16 turnos. `exigirJSON` ahora mira `ContentLength`, y hay dos subpruebas nuevas en
`TestP02`. Comprobado en vivo por el orquestador, con la cuenta sembrada:

```
login:                                   HTTP 200, cookie HttpOnly SameSite=Lax 28800s
me con cookie:                           HTTP 200
logout SIN cuerpo y SIN encabezados:     HTTP 204
la misma cookie despues:                 HTTP 401
cuerpo con tipo que no es JSON:          HTTP 415
make check:                              exit 0
```

Su primera corrida de la correccion fallo, porque uso `r.Body != http.NoBody` en vez de
`ContentLength`, y los dos casos nuevos del enrutador lo detectaron. Lo cambio y paso.
