# 06 - Jerarquia organizacional y usuarios

| Campo | Valor |
|---|---|
| Fecha | 2026-10-04 |
| Herramienta | Claude Code `2.1.289`, modo headless `claude -p` |
| Modelo | `opus` |
| Ejecucion | `docs/evidencias/sesiones/<archivo>.json` y `.meta.txt` |
| Commit base | `e72aa49` |

## Objetivo

El mantenimiento completo de la estructura organizacional y de los usuarios, con la
autorizacion aplicada en el servidor. Con P03, P04 y P05 del enunciado automatizadas.

## Contexto suministrado

- `AGENTS.md`, reglas permanentes, se carga solo. Secciones 7 y 9.
- `docs/diseno/reglas.md`: unicidad, huerfanos y padres inactivos, baja logica con
  dependencias, autorizacion por rol. Las reglas ya estan decididas ahi.
- `docs/diseno/arquitectura.md`: los endpoints de estructura organizacional y de usuarios,
  con sus codigos de error.
- `docs/contexto/enunciado.md`, **solo las secciones 3.2 y 6**.

## Instrucciones

1. Alta, consulta, modificacion y baja logica para `empresa`, `area`, `departamento`,
   `seccion` y `puesto`, y para `usuario`.
2. Reglas, tal como estan en `reglas.md`:
   - codigo unico dentro del padre;
   - no se crea nada colgando de un padre inactivo;
   - la baja se rechaza mientras existan dependientes activos, con 409
     `DEPENDENCIAS_ACTIVAS` y la lista de cuales son. Nunca en cascada;
   - reactivar exige que el padre este activo;
   - no se puede desactivar ni pasar a consulta al ultimo administrador activo (409
     `ULTIMO_ADMINISTRADOR`).
3. Usuarios: nombre, usuario o correo unico, rol, estado y puesto. Al crear o cambiar la
   contrasena se guarda con argon2id. **Ninguna respuesta incluye el hash ni la sal**, en
   ninguna ruta, ni en los listados.
4. **Autorizacion en el servidor.** Todo lo que escribe exige rol administrador. El rol
   consulta lee los datos funcionales y nada mas. Que no dependa de que la interfaz
   esconda botones.
5. Mensajes de error comprensibles: que digan que paso y con que dato, sin filtrar nada
   interno. Usa el formato de error del diseno.
6. Pruebas en `tests/integracion`:
   - `TestP03_...`: un usuario con rol consulta intenta escribir. **Recorre todos los
     endpoints de escritura de estructura y usuarios, uno por uno**, y comprueba que todos
     responden 403. Y que las lecturas si le funcionan. Construye la lista desde las rutas
     reales: si manana se agrega un endpoint de escritura y se olvida el middleware, esta
     prueba tiene que fallar.
   - `TestP04_...`: crear la jerarquia completa, de empresa a puesto, asignar un usuario y
     recuperarlo con su jerarquia.
   - `TestP05_...`: codigo duplicado y referencia inexistente. Comprueba el codigo de
     estado y que el mensaje nombre el dato que fallo.
   - Baja con dependientes activos: 409 y la lista.
   - Crear colgando de un padre inactivo: rechazado.
   - El ultimo administrador: 409.
   - Que ningun listado ni ficha de usuario traiga el hash.

## Restricciones

- No implementes el catalogo de servicios, ni el importador, ni la interfaz.
- No toques `hooks/`, `.claude/`, `data/`, `docs/contexto/` ni `docs/diseno/`.
- No cambies lo que ya funciona de autenticacion salvo que haga falta para esto, y si lo
  cambias dilo.
- Si hace falta un cambio de esquema, agrega una migracion nueva. No edites las existentes.
- No leas el archivo de entorno.
- Nunca ejecutes `make reset-dev` ni `down -v` fuera de `catalogo-test`.

## Salida esperada

El codigo, las pruebas y un reporte final con: los endpoints que quedaron y quien puede
llamarlos, los comandos con su codigo de salida, el resultado de P03, P04 y P05, y lo
pendiente.

## Criterio de aceptacion

1. `make check` termina en 0, con `TestP03`, `TestP04` y `TestP05` entre las que pasan.
2. `TestP03` cubre **todos** los endpoints de escritura de estructura y usuarios. Digo
   cuantos son en el reporte y que ninguno quedo fuera.
3. Para comprobar que `TestP03` sirve: quita a proposito el middleware de rol de un
   endpoint de escritura y confirma que la prueba falla. Despues dejalo como estaba y dilo
   en el reporte.
4. Un usuario con rol consulta recibe 403 al intentar crear una empresa, y 200 al listarlas.
5. Crear dos areas con el mismo codigo en la misma empresa devuelve 409, y el mensaje
   nombra el codigo repetido.
6. Crear un departamento con un area que no existe devuelve error, y el mensaje nombra la
   referencia que falta.
7. Desactivar una empresa con areas activas devuelve 409 `DEPENDENCIAS_ACTIVAS` y el cuerpo
   lista las areas.
8. `grep -ri 'hash' --include='*.go' internal/http` no muestra ningun campo de hash en una
   respuesta.

---

> Lo de arriba es el texto que se envio, sin retocar. Lo de abajo se escribio despues.

## Resultado

**Aceptado.** Sesion:
[`20261004-103722-06-organizacion-usuarios.json`](../evidencias/sesiones/20261004-103722-06-organizacion-usuarios.json),
66 turnos con opus, 16 minutos.

25 endpoints de escritura, solo para administrador, y 12 de lectura para cualquier sesion.
Las rutas se registran recorriendo los cinco niveles, y `RutasAPI()` devuelve el mismo
registro que usa el enrutador.

### Criterios, comprobados por el orquestador

| # | Criterio | Resultado |
|---|---|---|
| 1 | `make check` en 0 con P03, P04 y P05 | Cumplido, exit 0 |
| 2 | P03 cubre los 25 endpoints de escritura | Cumplido, la lista sale de `RutasAPI()` |
| 3 | Quitar el middleware hace fallar P03 | **Cumplido.** Ver abajo |
| 4 | Consulta: 403 al escribir, 200 al leer | Cumplido en vivo |
| 5 | Codigo duplicado da 409 y el mensaje nombra el codigo | Cumplido en vivo |
| 6 | Referencia inexistente, mensaje que la nombra | Cumplido en vivo |
| 7 | Baja con dependientes activos, 409 con la lista | Cumplido en vivo |
| 8 | Ningun campo de hash en las respuestas | Cumplido, `grep` sin coincidencias |

### Por que P03 vale

El error clasico de autorizacion no es olvidar el middleware: es olvidarlo en **una sola
ruta** de veinticinco. Una prueba que comprueba "el usuario de consulta no puede crear una
empresa" pasa igual aunque desactivar un puesto haya quedado abierto.

Por eso el prompt pedia recorrer todas las rutas y, sobre todo, demostrar que la prueba
sirve. El trabajador quito el middleware de la ruta que fija la contrasena de un usuario y
corrio P03. Fallo **solo en esa subprueba**:

```
=== RUN   TestP03_ConsultaNoPuedeEscribir/escritura_PUT_...contrasena
    organizacion_test.go:246: estado = 422, se esperaba 403
--- PASS: TestP03_ConsultaNoPuedeEscribir/crear_empresa_da_403_y_listarlas_da_200
FAIL
```

La subprueba generica siguio pasando. Ahi esta la diferencia entre enumerar y poner un
ejemplo. Evidencia completa en
[`06-p03-sin-middleware.txt`](../evidencias/pruebas/06-p03-sin-middleware.txt). El
middleware quedo restaurado y `make check` volvio a 0.

### Comprobacion en vivo con las cuentas de evaluacion

Con la cuenta de consulta: crear empresa 403, desactivar un puesto 403, cambiar la
contrasena de otro usuario 403. Listar empresas y usuarios 200, y el listado sin ningun
hash.

Con la cuenta de administrador, los mensajes que el enunciado pide "comprensibles":

```
Ya existe un area con codigo A1 en la empresa VERIF.
Los codigos dados de baja tampoco se reutilizan.                       409

No existe un area con id 999999 (campo area_id).                       422

No se puede desactivar la empresa VERIF: tiene 1 area activa.
Desactivelos primero, de abajo hacia arriba.
  dependencias: [{entidad: area, id: 6, codigo: A1, nombre: Area uno}]  409
```

Nombran el dato y el padre, no solo el codigo de error. Los registros creados para esta
comprobacion quedaron dados de baja.

### Decisiones suyas, revisadas y aceptadas

D28 (limites de longitud), D29 (correo vacio se rechaza en vez de convertirse en nulo) y
D30 (forma del aviso al dar de baja a un responsable).

Cambio en lo que ya funcionaba: la firma de `http.Nuevo` pasa a recibir una estructura de
servicios. Lo reporto. Autenticacion se comporta igual y sus pruebas siguen pasando.

### Pendiente

- Detalle de redaccion: el mensaje de dependencias dice "tiene 1 area activa. Desactivelos
  primero", sin concordar en singular. Anotado para la revision.
- `reglas.md` no recoge los limites de D28.
