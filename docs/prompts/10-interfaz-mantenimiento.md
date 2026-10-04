# 10 - Interfaz: mantenimiento y resultado de importacion

| Campo | Valor |
|---|---|
| Fecha | 2026-10-04 |
| Herramienta | Claude Code `2.1.289`, modo headless `claude -p` |
| Modelo | `sonnet` |
| Ejecucion | `docs/evidencias/sesiones/<archivo>.json` y `.meta.txt` |
| Commit base | `542002d` |

## Objetivo

Cerrar la interfaz: formularios del catalogo, mantenimiento de la estructura organizacional
y de usuarios, y una pantalla que muestre el resultado de la importacion con sus
incidencias.

## Contexto suministrado

- `AGENTS.md`, reglas permanentes, se carga solo. Secciones 8 y 9.
- `docs/diseno/arquitectura.md`: la lista de endpoints y el formato de error.
- `docs/diseno/reglas.md`: que puede hacer cada rol, y la politica de baja con
  dependencias, que la interfaz tiene que saber mostrar.
- El codigo de `web/`, ya con inicio de sesion, catalogo y ficha.
- `docs/contexto/enunciado.md`, **solo las secciones 3.2, 3.3 y 3.4**.

## Instrucciones

1. **Falta un endpoint y hay que agregarlo**: listar las importaciones y ver una, con su
   resumen y sus incidencias. Sigue el formato de los demas listados y pide solo sesion.
   Agregalo tambien al modulo de la API de la interfaz.
2. **Formularios del catalogo:** crear y editar servicios de los dos niveles, con las
   opciones de clase, criticidad y tipo tomadas de la API. Desactivar y reactivar.
   Asignar seccion y usuario responsable, con la lista de usuarios limitada a los de la
   seccion elegida.
3. **Estructura organizacional:** las cinco entidades, cada una con su listado, su
   formulario y sus acciones de baja y alta. Que se vea de quien depende cada registro.
4. **Usuarios:** listado, alta, edicion, cambio de contrasena, baja y alta. Nunca se muestra
   ningun hash, porque la API no lo manda.
5. **Errores del servidor, mostrados tal como vienen.** En particular:
   - cuando la baja se rechaza por dependencias, **la pantalla lista los dependientes** que
     devuelve el servidor, no un mensaje generico;
   - cuando una validacion falla, el mensaje del servidor se muestra junto al campo que
     nombra.
   Nada de textos inventados en el cliente.
6. **Resultado de importacion:** el resumen con creados, actualizados, omitidos y
   observados, y la lista de incidencias con su tipo, su celda o fila y sus valores.
7. **El rol consulta no ve botones de escritura**, usando lo que ya existe en
   `sesion.tsx`. Comodidad, no seguridad.
8. Mantener `TestWeb_RutasUsadasExisten` en verde con las rutas nuevas.

## Restricciones

- Del lado del servidor, **solo** el endpoint de importaciones del punto 1. Nada mas.
- No toques las migraciones, el importador, `hooks/`, `.claude/`, `data/`,
  `docs/contexto/` ni `docs/diseno/`.
- No agregues dependencias nuevas a la interfaz.
- Ninguna credencial escrita en el codigo.
- No leas el archivo de entorno. Nunca ejecutes `make reset-dev` ni `down -v` fuera de
  `catalogo-test`.
- Si algo no cabe, **prefiere dejar una pantalla fuera y decirlo** antes que entregar media
  pantalla que no funciona. Orden de importancia si hay que recortar: resultado de
  importacion, usuarios, estructura organizacional, formularios del catalogo.

## Salida esperada

El codigo y un reporte final con: las pantallas que quedaron, las que dejaste fuera y por
que, las rutas nuevas, los comandos con su codigo de salida y lo pendiente.

## Criterio de aceptacion

1. `make check` termina en 0, con la revision de tipos y `TestWeb_RutasUsadasExisten`
   pasando.
2. Las pruebas P01 a P11 siguen pasando.
3. El endpoint nuevo responde: listar importaciones devuelve las corridas con su resumen, y
   ver una devuelve sus incidencias.
4. `docker compose up --build -d` termina en 0 y todas las direcciones de la interfaz
   devuelven 200.
5. `grep -riE 'contrasena.*=.*"|password.*=.*"' web/src` sin coincidencias.
6. Ninguna ruta de la API usada por la interfaz falta en el servidor.

---

> Lo de arriba es el texto que se envio, sin retocar. Lo de abajo se escribio despues.

## Resultado

**Aceptado.** Sesion: [`20261004-161821-10-interfaz-mantenimiento.json`](../evidencias/sesiones/20261004-161821-10-interfaz-mantenimiento.json), 57 turnos.

No quedo ninguna pantalla fuera: importaciones con sus incidencias, usuarios, las cinco
entidades de la estructura, formularios de servicios de los dos niveles y asignacion de
responsable.

### Comprobado por el orquestador

```
make check                        exit 0
listado de importaciones          total 2, la ultima completada con 0 creados y 6 observados
ficha de una importacion          10 incidencias
los dos endpoints sin sesion      401
rutas de la interfaz probadas     importaciones, usuarios, organizacion, servicios-n1,
                                  servicios/nuevo: 200 todas
P01 a P11                         siguen pasando
```

### Lo que mas importa de estas pantallas

Una baja rechazada **lista los dependientes que devuelve el servidor**, no un mensaje
generico, y con "y N mas" cuando vienen truncados. Y los errores de validacion se muestran
junto al campo que el servidor nombra. No hay textos inventados en el cliente: si el
servidor cambia un mensaje, la pantalla cambia con el.

### Desviacion del diseno

`arquitectura.md` lista un endpoint para lanzar la importacion desde la aplicacion, que no
existe: el prompt pedia solo los dos de consulta. La importacion se ejecuta por comando, que
es lo que el enunciado admite ("por comando o desde la aplicacion"). La pantalla de
importaciones solo consulta.

### Pendiente

- El mantenimiento de clases, criticidades y tipos no se hizo: hoy solo se usan como listas
  de opciones. El enunciado pide mantenerlos, asi que queda como hueco conocido.
- **Sigue sin probarse en un navegador.** Ningun formulario se ejecuto contra el servidor.
  Lo cierra la tarea siguiente.
