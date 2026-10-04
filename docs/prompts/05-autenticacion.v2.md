# 05 - Autenticacion, sesiones y cuentas de evaluacion (v2)

| Campo | Valor |
|---|---|
| Fecha | 2026-10-04 |
| Herramienta | Claude Code `2.1.289`, modo headless `claude -p` |
| Modelo | `opus` |
| Ejecucion | `docs/evidencias/sesiones/<archivo>.json` y `.meta.txt` |
| Commit base | `9a90098` |

> **Version revisada.** La v1 se ejecuto y dejo la autenticacion funcionando, pero su
> criterio 6 no decia **como** se llama a cerrar sesion, y eso dejo pasar un defecto: la
> regla de exigir `Content-Type: application/json` en todo `POST` se aplica tambien a
> cerrar sesion, que no lleva cuerpo. Un navegador que llama sin encabezado recibe 415 y
> **la sesion no se revoca**. Partes de aqui del codigo que ya existe; lo que hay que
> corregir es ese caso y precisar su prueba.

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
   - **Cerrar sesion funciona sin cuerpo y sin `Content-Type`**, que es como lo llama un
     navegador con `fetch(url, {method:'POST'})`. La exigencia de JSON se aplica solo a las
     peticiones que **si** traen cuerpo. No se pierde defensa contra peticiones falsificadas
     desde otro sitio: de eso se encarga `SameSite=Lax`, que impide que la cookie viaje en
     un `POST` de origen distinto.
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
6. Cerrar sesion llamado **sin cuerpo y sin `Content-Type`** devuelve 204, y despues esa
   misma cookie devuelve 401. Comprobado con
   `curl -b cookies -X POST .../api/auth/logout` sin ningun `-H`.
8. Una peticion que **si** trae cuerpo con un `Content-Type` que no es JSON sigue
   devolviendo 415: la defensa no se quita, se acota.
9. Una prueba de `TestP02_...` cubre el cierre de sesion llamado como lo haria un
   navegador, sin encabezados.
7. `grep -ri 'argon2\\$\\|password.*=.*"' --include='*.go'` no muestra ninguna contrasena
   escrita en el codigo.

---

> Lo de arriba es el texto que se envio, sin retocar. Lo de abajo se escribio despues.

## Resultado

**Aceptado.** Sesion:
[`20261004-102452-05-autenticacion.v2.json`](../evidencias/sesiones/20261004-102452-05-autenticacion.v2.json),
16 turnos.

El detalle de esta iteracion, con el problema que la motivo y la comprobacion en vivo, esta
en la seccion "Iteracion" de [`05-autenticacion.md`](05-autenticacion.md), para no contar la
misma historia dos veces.

En resumen: cerrar sesion sin cuerpo ni encabezados devuelve 204 y deja la cookie muerta,
401 despues, y una peticion con cuerpo y un tipo que no sea JSON sigue devolviendo 415.
`make check` en 0.
