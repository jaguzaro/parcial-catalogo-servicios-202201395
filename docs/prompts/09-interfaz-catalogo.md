# 09 - Interfaz: inicio de sesion y catalogo

| Campo | Valor |
|---|---|
| Fecha | 2026-10-04 |
| Herramienta | Claude Code `2.1.289`, modo headless `claude -p` |
| Modelo | `sonnet` |
| Ejecucion | `docs/evidencias/sesiones/<archivo>.json` y `.meta.txt` |
| Commit base | `2f8db1f` |

## Objetivo

Lo que el catedratico va a mirar primero: entrar con una cuenta y recorrer el catalogo con
busqueda, filtros, paginacion y ficha. Las pantallas de mantenimiento van en la tarea
siguiente.

## Contexto suministrado

- `AGENTS.md`, reglas permanentes, se carga solo. Secciones 8 y 9.
- `docs/diseno/arquitectura.md`: la lista de endpoints con lo que recibe y devuelve cada
  uno, el formato de error y como se sirve la interfaz desde el binario.
- `docs/contexto/enunciado.md`, **solo las secciones 3.1, 3.3 y 5**.
- El codigo ya existente de `web/`, que hoy es una sola pantalla de prueba.

La API ya esta hecha y **no se toca**.

## Instrucciones

1. **Un solo modulo con las llamadas a la API**, en `web/src/api.ts`. Toda peticion pasa por
   ahi, con las rutas como constantes. Nada de rutas escritas suelta en los componentes.
2. **Inicio de sesion.** Formulario de usuario o correo y contrasena. Si falla, muestra el
   mensaje que devuelve el servidor, sin inventar texto. Al entrar, va al catalogo.
3. **Sesion.** Al cargar, consulta quien soy. Si responde 401, lleva al inicio de sesion.
   La barra superior muestra el nombre y el rol, y un boton de cerrar sesion que llama al
   endpoint y vuelve al formulario.
4. **Catalogo**, la pantalla principal:
   - tabla de servicios de nivel 2 con codigo, nombre, nivel 1, estado, clase, criticidad y
     tipo;
   - busqueda por codigo y por nombre;
   - filtros por nivel 1, estado, clase, criticidad y tipo, combinables, con las opciones
     tomadas de los catalogos de la API, no escritas a mano;
   - paginacion, con el total a la vista;
   - los filtros y la pagina se reflejan en la direccion del navegador, para poder compartir
     o recargar sin perderlos.
5. **Ficha de un servicio:** todos sus atributos, su nivel 1, su seccion y usuario
   responsable, y sus incidencias. Lo que no tiene valor se muestra como "sin dato", **no
   como cero ni vacio**: esa diferencia es parte de lo que se evalua.
6. **El rol consulta no ve los botones de escritura.** Dejalo claro en el codigo: es
   comodidad, no seguridad. La autorizacion ya la aplica el servidor.
7. Sin libreria de componentes. Estilos propios, sobrios y legibles. Que se entienda, no que
   luzca.
8. **Una prueba en Go**, `TestWeb_RutasUsadasExisten`, que lea las rutas declaradas en
   `web/src/api.ts` y compruebe que cada una existe en `RutasAPI()`. Si la interfaz llama a
   un endpoint que no existe, esa prueba falla.

## Restricciones

- No toques nada de `internal/` salvo lo imprescindible para servir la interfaz, ni las
  migraciones, ni el importador.
- No agregues dependencias mas alla de React, el enrutador y las herramientas de Vite.
- No toques `hooks/`, `.claude/`, `data/`, `docs/contexto/` ni `docs/diseno/`.
- Ninguna contrasena ni usuario escritos en el codigo de la interfaz.
- No leas el archivo de entorno. Nunca ejecutes `make reset-dev` ni `down -v` fuera de
  `catalogo-test`.
- Si una pantalla necesita un endpoint que no existe, **no lo inventes en el cliente**:
  dilo en el reporte y deja esa parte fuera.

## Salida esperada

El codigo de la interfaz y un reporte final con: las pantallas que quedaron, las rutas de la
API que usa, los comandos con su codigo de salida, y lo que quedo fuera.

## Criterio de aceptacion

1. `make check` termina en 0, con la revision de tipos de la interfaz incluida, y
   `TestWeb_RutasUsadasExisten` entre las pruebas que pasan.
2. `docker compose up --build -d` termina en 0 y la raiz devuelve el HTML con el script de
   la interfaz ya compilada.
3. Una direccion interna como la ficha de un servicio devuelve 200 con el mismo HTML, es
   decir que recargar en una pantalla honda no da 404.
4. `grep -riE 'contrasena.*=.*"|password.*=.*"' web/src` no muestra ninguna credencial.
5. Toda ruta de la API que use la interfaz existe en el servidor, demostrado por la prueba
   del punto 8 y no a ojo.
6. Las pruebas P01 a P11 siguen pasando.

---

> Lo de arriba es el texto que se envio, sin retocar. Lo de abajo se escribio despues.

## Resultado

**Aceptado**, con una limitacion que conviene leer. Sesion:
[`20261004-160200-09-interfaz-catalogo.json`](../evidencias/sesiones/20261004-160200-09-interfaz-catalogo.json), 31 turnos.

Pantallas: inicio de sesion, catalogo con busqueda, cinco filtros combinables, paginacion y
total a la vista, y ficha de servicio. Los filtros y la pagina quedan en la direccion del
navegador, asi que recargar o compartir el enlace no los pierde.

### Comprobacion independiente

```
make check                        exit 0 (incluye la revision de tipos de la interfaz)
raiz, login y ficha honda         200 las tres, con el mismo script compilado
paquete compilado                 270381 bytes, servido por el binario
rutas de api dentro del paquete   api/auth, api/catalogos, api/servicios, api/servicios-n1
credenciales en el codigo         ninguna
P01 a P11                         siguen pasando
```

Que una direccion honda como la ficha devuelva 200 importa: significa que recargar ahi no
da 404, que es el fallo tipico de una interfaz de una sola pagina mal servida.

### Limitacion

**Nadie ha abierto esta interfaz en un navegador.** El asistente lo dijo por su cuenta: lo
que esta comprobado es el HTML servido, los tipos y que las rutas que llama existen de
verdad. Que las pantallas pinten los datos y que los filtros se vean funcionar no esta
comprobado.

Se cierra mas adelante con una prueba de extremo a extremo en contenedor, de inicio de
sesion y de un filtro. Hasta entonces, esta parte queda declarada como no verificada en
comportamiento.

### Mas alla de lo pedido

`TestWeb_RutasUsadasExisten` no solo compara las rutas declaradas en el modulo de la API:
tambien falla si cualquier otro archivo de la interfaz escribe una ruta suelta. Eso cierra
el agujero de la prueba, que era declarar una ruta en el modulo y llamar otra en un
componente.

Lo que no tiene valor se muestra como "sin dato", tambien en minimo y maximo. Esa diferencia
entre ausente y cero es parte de lo que se evalua.
