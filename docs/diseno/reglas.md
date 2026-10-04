# Reglas de negocio

Reglas que aplica el servidor. Las que tambien estan en la base (`UNIQUE`, `CHECK`, FK) se
citan con su restriccion; las demas se validan en Go dentro de la misma transaccion que
escribe. Los codigos de error y su formato estan en
[arquitectura.md](arquitectura.md#formato-de-error).

## Invariante general

> Todo registro activo tiene a todos sus ancestros activos.

Ancestros: la cadena Empresa -> Area -> Departamento -> Seccion -> Puesto -> Usuario, y
Servicio N1 -> Servicio N2. Las reglas de las secciones siguientes son las que la
mantienen: no se crea ni se reactiva nada bajo un padre inactivo, y no se desactiva un
padre con hijos activos.

## Unicidad de codigos

| Entidad | Alcance | Restriccion |
|---|---|---|
| `empresa` | Global | `UNIQUE (codigo)` |
| `area` | Dentro de su empresa | `UNIQUE (empresa_id, codigo)` |
| `departamento` | Dentro de su area | `UNIQUE (area_id, codigo)` |
| `seccion` | Dentro de su departamento | `UNIQUE (departamento_id, codigo)` |
| `puesto` | Dentro de su seccion | `UNIQUE (seccion_id, codigo)` |
| `usuario` | Global, sin distinguir mayusculas | `UNIQUE (lower(usuario))`, `UNIQUE (lower(correo))` |
| `servicio_n1` | Global | `UNIQUE (codigo)` |
| `servicio_n2` | Global | `UNIQUE (codigo)` |
| Catalogos | Global | `UNIQUE (nombre)`, `UNIQUE (valor_origen)` |

- La unicidad cuenta tambien con los registros inactivos: un codigo dado de baja no se
  puede reutilizar. Si se reutilizara, la baja logica dejaria de servir como historial.
- Se compara el codigo exacto, distinguiendo mayusculas. Desde la API se rechaza un codigo
  con espacios al inicio o al final (422 `VALIDACION`). El servidor no lo recorta en
  silencio.
- El servidor no consulta antes si el codigo existe: inserta, y si PostgreSQL responde con
  violacion de unicidad (`23505`) la traduce a 409 `CODIGO_DUPLICADO`, nombrando el campo
  y el alcance ("ya existe un area con codigo `TI` en la empresa `EMP01`"). Asi no hay
  carrera entre dos altas simultaneas.
- Cambiar el padre de un registro (mover un area a otra empresa) revisa la unicidad en el
  padre nuevo con la misma restriccion.

## Huerfanos y padres inactivos

- **Huerfanos.** La FK al padre es `NOT NULL` en `area`, `departamento`, `seccion`,
  `puesto`, `usuario` y `servicio_n2`. No hay borrado fisico y las FK son `RESTRICT`, asi
  que un padre no desaparece.
- **Referencia inexistente.** Si el `*_id` que llega no existe, responde 422
  `REFERENCIA_INEXISTENTE` con el campo (`{"campo": "area_id"}`). El servidor lo comprueba
  antes de insertar, para dar un mensaje claro, y la FK lo garantiza igual (`23503` se
  traduce al mismo error).
- **Padre inactivo.** Crear, mover o reactivar un registro cuyo padre esta inactivo
  responde 409 `PADRE_INACTIVO`, con el padre (`{"entidad": "seccion", "id": 7, "codigo": "S1"}`).
  Lo mismo vale para las referencias que no son de jerarquia y que se fijan o cambian en
  la operacion:
  - catalogos (clase, criticidad, tipo) inactivos al crear o editar un servicio. Si el
    servicio ya tenia ese valor y no se cambia, se conserva;
  - seccion responsable inactiva y usuario responsable inactivo, al asignar responsable.
- **Concurrencia.** La validacion lee el padre con `SELECT ... FOR SHARE` y la baja lee el
  registro con `SELECT ... FOR UPDATE`. Asi un alta y la baja de su padre no se cruzan.

## Baja logica con dependencias

Politica: **se rechaza la baja mientras existan dependientes activos, y se dice cuales
son.** Nunca se desactiva en cascada ni en silencio. Para dar de baja una rama hay que
hacerlo de abajo hacia arriba.

Operacion: `POST /api/{recurso}/{id}/desactivar`. Respuestas:

- **200** con el registro ya inactivo, si no habia dependientes que lo impidan. Si el
  registro ya estaba inactivo, tambien 200 y no cambia nada: la operacion es idempotente.
- **409 `DEPENDENCIAS_ACTIVAS`** si los hay. No se modifica nada. El cuerpo lista los
  dependientes:

```json
{
  "error": {
    "codigo": "DEPENDENCIAS_ACTIVAS",
    "mensaje": "No se puede desactivar la seccion S1: tiene 2 puestos activos y 1 servicio activo a su cargo.",
    "dependencias": [
      {"entidad": "puesto", "id": 4, "codigo": "P01", "nombre": "Analista"},
      {"entidad": "puesto", "id": 5, "codigo": "P02", "nombre": "Jefe"},
      {"entidad": "servicio_n2", "id": 18, "codigo": "SE.06.01", "nombre": "Administrar Aplicaciones - Area 1"}
    ],
    "total_dependencias": 3
  }
}
```

  `dependencias` trae como mucho 50 elementos, y `total_dependencias` dice cuantos hay en
  total.

Que bloquea la baja, entidad por entidad:

| Entidad | Bloquean la baja | Que pasa con lo demas |
|---|---|---|
| Empresa | Areas activas. | |
| Area | Departamentos activos. | |
| Departamento | Secciones activas. | |
| Seccion | Puestos activos, y servicios N2 con `activo <> 'N'` que la tengan como `seccion_responsable_id`. | Los servicios dados de baja conservan la seccion como dato historico. |
| Puesto | Usuarios activos. | |
| Usuario | Nada. Ver [la nota](#por-que-un-usuario-responsable-no-bloquea-su-baja). | Se revocan todas sus sesiones en la misma transaccion. La respuesta 200 trae `advertencias` con los servicios de los que sigue siendo responsable. La asignacion no se borra. |
| Servicio N1 | Servicios N2 con `activo <> 'N'`, es decir `S` o `DESCONOCIDO`. | |
| Servicio N2 | Nada. Desactivar pone `activo = 'N'`. | Conserva seccion y usuario responsable. |
| Clase, criticidad, tipo | Servicios N2 con `activo <> 'N'` que la usen. | Una opcion inactiva desaparece de los formularios, pero se sigue mostrando en los servicios que la tienen. |

`DESCONOCIDO` bloquea igual que `S`: no se sabe si el servicio esta activo, y suponer que
no lo esta seria decidir por el usuario.

**Reactivar** (`POST /api/{recurso}/{id}/activar`): exige que el padre este activo (409
`PADRE_INACTIVO`). No reactiva hijos. En un servicio N2 pone `activo = 'S'`, que es la
forma de resolver un `DESCONOCIDO` despues de revisarlo; tambien se puede fijar `activo`
directamente con `PUT`.

**Cambios que romperian una asignacion.** No son bajas, pero se tratan igual para que una
asignacion de responsable no quede invalida sin que nadie lo note:

- cambiar el `puesto_id` de un usuario a un puesto de otra seccion;
- mover un puesto a otra seccion si alguno de sus usuarios es responsable de servicios.

Los dos se rechazan con 409 `ASIGNACIONES_AFECTADAS`, listando los servicios. Primero hay
que reasignar o quitar el responsable.

### Por que un usuario responsable no bloquea su baja

Desactivar a un usuario es tambien una medida de seguridad, por ejemplo cuando alguien deja
la organizacion. Si la asignacion de un servicio la impidiera, habria que tocar el catalogo
para poder quitarle el acceso a una persona. Por eso la baja se hace y la dependencia se
informa en `advertencias`. Ademas la ficha del servicio muestra al responsable como
inactivo. No se pierde nada y nada queda oculto. Es una decision de este diseno.

### Proteccion del ultimo administrador

No se puede desactivar ni cambiar a `consulta` el ultimo usuario activo con rol
`administrador`: 409 `ULTIMO_ADMINISTRADOR`. Si no, nadie podria mantener el sistema
salvo por la linea de comandos. Es una decision de este diseno; el enunciado no la pide.

## Minimo y maximo

- `minimo` y `maximo` son `numeric` y admiten nulo. Un campo vacio en el formulario o
  ausente en el JSON se guarda `NULL`. El servidor no lo convierte en `0`.
- En el JSON se distingue `null` (sin valor) de `0` (cero). Una cadena vacia en un campo
  numerico es 422 `VALIDACION`, no cero.
- Si llegan los dos, tiene que cumplirse `minimo <= maximo`. Si no, 422
  `MINIMO_MAYOR_QUE_MAXIMO` con `campos: ["minimo", "maximo"]`, y no se guarda nada. La
  base lo garantiza igual con `CHECK (minimo IS NULL OR maximo IS NULL OR minimo <= maximo)`.
- Con uno solo informado no hay nada que comparar y se acepta.
- Al editar se valida el estado final: si solo cambia `minimo`, se compara contra el
  `maximo` que ya estaba guardado.
- No se admiten negativos ni se pide entero: el enunciado no lo dice.

## Validacion de servicios

| Campo | N1 | N2 |
|---|---|---|
| `codigo` | Obligatorio, unico global | Obligatorio, unico global |
| `nombre` | Obligatorio | Obligatorio |
| `servicio_n1_id` | — | Obligatorio, existente y activo |
| `activo` | — (`boolean` del registro) | `S`, `N` o `DESCONOCIDO`; por defecto `S` |
| `clase_id`, `criticidad_id`, `tipo_id` | — | Opcionales. Si vienen: existentes y activos |
| `descripcion`, `metrica` | — | Opcionales |
| `minimo`, `maximo` | — | Opcionales, [ver arriba](#minimo-y-maximo) |
| `requiere_revision` | — | Opcional, `boolean`. Lo puede limpiar el administrador |

En los servicios creados desde la aplicacion, clase, criticidad y tipo tambien son
opcionales. Si se exigieran, no se podrian editar los servicios importados que les faltan
sin inventar un valor. El codigo de un servicio N2 no tiene que empezar por el de su N1:
el enunciado no lo pide y el Excel no siempre lo sigue (`SE.12.1`).

## Responsable de un servicio

Se asigna con `PUT /api/servicios/{id}/responsable` y el cuerpo
`{"seccion_id": <id|null>, "usuario_id": <id|null>}`.

1. `usuario_id` sin `seccion_id`: 422 `VALIDACION`. La base tambien lo impide con un
   `CHECK`.
2. `seccion_id` tiene que existir (422 `REFERENCIA_INEXISTENTE`) y estar activa (409
   `PADRE_INACTIVO`).
3. `usuario_id` tiene que existir y estar activo, y su puesto tiene que ser de esa misma
   seccion (`usuario.puesto_id -> puesto.seccion_id = seccion_id`). Si es de otra: 422
   `RESPONSABLE_FUERA_DE_SECCION`, con la seccion del usuario y la pedida. Esto es P11.
4. `{"seccion_id": null, "usuario_id": null}` quita la asignacion.
5. No se puede asignar responsable a un servicio con `activo = 'N'` (409
   `SERVICIO_INACTIVO`).
6. El importador nunca toca estos dos campos (D13).

El resto de [cambios que romperian una asignacion](#baja-logica-con-dependencias) estan
cubiertos para que la regla 3 se siga cumpliendo despues.

## Autenticacion y sesiones

Segun D15 y D16.

**Contrasenas.** argon2id con `golang.org/x/crypto/argon2`, sal aleatoria de 16 bytes
por usuario, salida de 32 bytes y parametros `m = 64 MiB`, `t = 3`, `p = 4`, la segunda
opcion recomendada del RFC 9106. Se guarda en formato PHC
(`$argon2id$v=19$m=65536,t=3,p=4$<sal>$<hash>`), asi que se pueden cambiar los parametros
sin invalidar los hashes anteriores. La comparacion es de tiempo constante. Longitud
minima: 8 caracteres; maxima: 256, para acotar el costo. Las dos las decide este diseno.

**Inicio de sesion** (`POST /api/auth/login`, `{"login": ..., "contrasena": ...}`):

1. Si `login` lleva `@` se busca por `lower(correo)`; si no, por `lower(usuario)`.
2. Si no existe, esta inactivo o la contrasena no coincide: 401 `CREDENCIALES_INVALIDAS`,
   siempre con el mismo mensaje. Si el usuario no existe se verifica igual contra un hash
   ficticio, para que el tiempo de respuesta no revele si la cuenta existe.
3. Si todo esta bien: se generan 32 bytes con `crypto/rand`, se codifican en base64url y
   ese es el token. Se guarda `sha256(token)` en `sesion` con
   `expira_en = now() + SESSION_TTL` (por defecto 8 h, decision de este diseno).
4. Cookie `catalogo_sesion=<token>`; `HttpOnly`; `SameSite=Lax`; `Path=/`;
   `Max-Age = SESSION_TTL`; `Secure` si `COOKIE_SECURE=true`. Va en `false` por defecto,
   porque la evaluacion local es por `http`.

**Cada peticion a `/api/*`**, salvo `login` y `/healthz`:

1. Se lee la cookie y se busca `sha256(token)` con
   `revocada_en IS NULL AND expira_en > now()`, unida a `usuario`.
2. Sin cookie, o si no hay sesion valida: 401 `NO_AUTENTICADO` y se borra la cookie.
3. Si el usuario esta inactivo: se revoca esa sesion y 401 `NO_AUTENTICADO`.
4. El rol se lee de la base en cada peticion, nunca de la cookie. Un cambio de rol vale
   desde la peticion siguiente.

**Cierre de sesion** (`POST /api/auth/logout`): `revocada_en = now()` y se borra la
cookie. Reusar el mismo token despues da 401.

**Falsificacion de peticiones (CSRF).** `SameSite=Lax` ya impide que otro sitio mande la
cookie en un `POST`. Ademas, todo `POST` y `PUT` exige `Content-Type: application/json`
(415 `TIPO_NO_SOPORTADO` si no): un formulario HTML de otro origen no puede mandarlo sin
una verificacion previa de CORS, y el servidor no habilita CORS. Este segundo control lo
agrega este diseno.

No se implementan: limite de intentos, recuperacion de contrasena, cambio de contrasena
por el propio usuario ni limpieza de sesiones vencidas. El enunciado no los pide.

## Usuario desactivado con la sesion abierta

1. `POST /api/usuarios/{id}/desactivar` pone `activo = false` y, en la misma transaccion,
   `revocada_en = now()` en todas sus sesiones vigentes.
2. Aunque una sesion se escapara de ese paso, cada peticion revisa `usuario.activo` y la
   rechaza con 401.
3. Su siguiente peticion recibe 401 `NO_AUTENTICADO` y la interfaz vuelve al inicio de
   sesion. No puede volver a entrar mientras siga inactivo.
4. Al reactivarlo, las sesiones anteriores siguen revocadas: tiene que iniciar sesion de
   nuevo.

## Autorizacion por rol

La autorizacion se aplica en el servidor, con un middleware por ruta. Que la interfaz
oculte botones es solo comodidad (enunciado 3.1).

| Operacion | administrador | consulta |
|---|---|---|
| Iniciar y cerrar sesion, `GET /api/auth/me` | Si | Si |
| Leer empresas, areas, departamentos, secciones, puestos | Si | Si |
| Leer usuarios (nombre, usuario, correo, rol, estado, puesto, jerarquia) | Si | Si |
| Leer hashes de contrasena o sesiones | No, nadie | No, nadie |
| Leer servicios N1 y N2, fichas, catalogos | Si | Si |
| Leer importaciones y sus incidencias | Si | Si |
| Crear, editar, activar o desactivar unidades organizacionales | Si | No (403) |
| Crear, editar, activar o desactivar usuarios; fijar contrasena | Si | No (403) |
| Crear, editar, activar o desactivar servicios y catalogos | Si | No (403) |
| Asignar responsable | Si | No (403) |
| Lanzar una importacion | Si | No (403) |

Un usuario de consulta que intente escribir recibe 403 `PROHIBIDO` y no se ejecuta nada.
Es lo que prueba P03. Primero se revisa la autenticacion y despues el rol: sin sesion
responde 401, aunque la ruta sea solo de administrador.

## Cuentas de evaluacion

El comando `catalogo sembrar-demo` crea, si no existen, una estructura organizacional de
demostracion (empresa `DEMO` con su area, departamento, dos secciones y puestos), un
usuario `administrador` y uno `consulta`, y tres asignaciones de responsable validas sobre
servicios importados. Los usuarios y contrasenas salen de variables de entorno
(`DEMO_ADMIN_USUARIO`, `DEMO_ADMIN_CONTRASENA`, `DEMO_CONSULTA_USUARIO`,
`DEMO_CONSULTA_CONTRASENA`). `.env.example` las trae con valores de ejemplo, no reales. Si
faltan las contrasenas, el comando falla, no inventa una. Correrlo dos veces no duplica
nada. Las asignaciones se hacen solo si ya hay servicios importados; si no, avisa y
termina con codigo distinto de 0. Todo lo que crea lleva el codigo `DEMO`, para que no se
confunda con datos del Excel (enunciado 3.3).
