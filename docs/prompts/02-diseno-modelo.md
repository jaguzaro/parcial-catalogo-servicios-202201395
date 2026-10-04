# 02 - Diseno del modelo de datos y de la aplicacion

| Campo | Valor |
|---|---|
| Fecha | 2026-10-04 |
| Herramienta | Claude Code `2.1.289`, modo headless `claude -p` |
| Modelo | `opus` |
| Ejecucion | `docs/evidencias/sesiones/<archivo>.json` y `.meta.txt` |
| Commit base | `e4b96a6` |

## Objetivo

Dejar escrito el diseno completo antes de escribir codigo: modelo de datos, diccionario,
mapeo del Excel a la base, reglas de negocio y arquitectura. Lo que salga de aqui es lo que
se va a implementar, asi que tiene que ser lo bastante preciso como para escribir las
migraciones sin volver a decidir nada.

## Contexto suministrado

- `AGENTS.md`, reglas permanentes, se carga solo. La seccion 6 trae los hallazgos del Excel
  y las reglas de lectura ya decididas.
- `docs/contexto/enunciado.md`, completo. El diseno tiene que cubrir las secciones 3, 5 y 6.
- `docs/contexto/analisis-excel.md`, lo que contiene el archivo, con celdas citadas.
- `docs/contexto/decisiones.md`, decisiones ya tomadas. Las D08 a D13 son reglas de
  importacion y las D14 a D19 son el stack. **Ninguna esta en discusion.**

## Instrucciones

Escribe cuatro archivos en `docs/diseno/`.

### 1. `modelo.md`

- Diagrama entidad relacion en Mermaid (`erDiagram`), con todas las entidades y sus
  relaciones.
- Diccionario de datos: una tabla por entidad, con columna, tipo, nulo o no nulo, llave
  primaria, llaves foraneas, restricciones de unicidad y una linea de descripcion.
- Toda entidad del diagrama tiene que aparecer en el diccionario, y al reves.
- Cubre lo de la seccion 3.2 del enunciado (empresa, area, departamento, seccion, puesto,
  usuario), lo de la 3.3 (servicios de nivel 1 y 2, clase, criticidad, tipo, asignacion de
  responsable) y lo que haga falta para sesiones y para la trazabilidad de la importacion.

### 2. `mapeo-excel.md`

- Tabla de las columnas A a L: columna del Excel, nombre del campo, tabla y columna de
  destino, tipo, transformacion si la hay.
- Una seccion por cada uno de los seis casos de la seccion 3.4 del enunciado, diciendo que
  hace el importador y citando la decision que lo respalda (D08 a D13).
- Como se registran las incidencias y que informacion lleva cada una.
- Que cuenta como creado, actualizado, omitido y observado en el resumen de importacion, y
  que hace que la segunda corrida no duplique nada.

### 3. `reglas.md`

- Unicidad de codigos: global en servicios, y dentro del padre en las unidades
  organizacionales.
- Prohibicion de registros huerfanos y de asociar a un padre inactivo.
- **Politica de baja logica con dependencias.** La decision ya tomada es: rechazar la baja
  mientras existan hijos activos, y decir cuales son. Nunca desactivar en cascada en
  silencio. Escribe como se aplica entidad por entidad y que devuelve el servidor.
- Validacion de minimo y maximo, incluido que un dato ausente no se convierte en cero.
- Autenticacion, sesiones y autorizacion por rol, segun D15 y D16. Que puede hacer cada rol
  y que pasa si un usuario se desactiva con la sesion abierta.
- Responsable de un servicio: solo un usuario que pertenezca a la seccion responsable.

### 4. `arquitectura.md`

- Stack, tomado de D14 a D19. No lo vuelvas a decidir, solo explica como encajan las piezas.
- Estructura de carpetas que va a tener el repositorio, con una linea por carpeta.
- Lista de endpoints: metodo, ruta, quien puede llamarlo, que recibe y que devuelve,
  incluidos los codigos de error.
- Como se sirve la interfaz desde el binario de Go y como se construye en Docker.
- Contenedores, volumen y healthcheck.

### 5. Al final de `modelo.md`

Una tabla que relacione cada escenario P01 a P12 de la seccion 6 del enunciado con la parte
del diseno que lo hace posible. Sirve para detectar lo que falta antes de programarlo.

## Restricciones

- **Nada de codigo.** Ni migraciones, ni Go, ni `Dockerfile`, ni `Makefile`, ni SQL en
  archivos aparte. Fragmentos de SQL dentro del diccionario, para precisar un tipo o una
  restriccion, estan bien.
- No crees archivos fuera de `docs/diseno/`.
- No vuelvas a abrir las decisiones D08 a D19. Si crees que alguna esta equivocada, no la
  cambies: escribelo en tu reporte final y sigue con el resto.
- No inventes requisitos. Si algo no esta en el enunciado, o no se decide aqui, dilo.
- Si el enunciado y `AGENTS.md` se contradicen en algo, detente y reportalo.

## Salida esperada

Los cuatro archivos y un reporte final con: las entidades que definiste, las decisiones de
diseno que tuviste que tomar tu, lo que quedo sin resolver y cualquier contradiccion que
hayas encontrado.

## Criterio de aceptacion

1. Existen los cuatro archivos en `docs/diseno/`.
2. El diagrama Mermaid abre con `erDiagram` y toda entidad que aparece en el tiene su
   tabla en el diccionario, con llave primaria y restricciones. Sin entidades sueltas en
   ninguno de los dos lados.
3. El mapeo tiene una fila por cada columna de la A a la L, las doce.
4. Los seis casos de la seccion 3.4 del enunciado tienen cada uno su seccion, y cada una
   cita la decision D08 a D13 que la respalda.
5. La tabla de escenarios cubre P01 a P12, los doce, y ninguno queda sin una parte del
   diseno que lo sostenga.
6. Las seis entidades de la seccion 3.2 del enunciado y los catalogos de clase, criticidad
   y tipo estan en el diccionario.
7. No hay archivos nuevos fuera de `docs/diseno/`.

---

> Lo de arriba es el texto que se envio, sin retocar. El SHA-256 que quedo registrado en
> las evidencias corresponde a esa version. Lo de abajo se escribio despues, asi que el
> hash del archivo de hoy ya no coincide.

## Resultado

Aceptado. Sesion:
[`20261004-010144-02-diseno-modelo.json`](../evidencias/sesiones/20261004-010144-02-diseno-modelo.json),
16 turnos, 12 minutos con opus.

| Criterio | Resultado |
|---|---|
| 1. Los cuatro archivos | Cumplido |
| 2. Diagrama y diccionario cuadran | Cumplido, 14 entidades en los dos lados, ninguna suelta |
| 3. Doce filas, de A a L | Cumplido |
| 4. Los seis casos citan su decision | Cumplido, D08 a D13 aparecen todas |
| 5. P01 a P12 | Cumplido, los doce |
| 6. Entidades de la seccion 3.2 y los tres catalogos | Cumplido |
| 7. Nada fuera de `docs/diseno/` | Cumplido, `git status` solo muestra lo que ya habia |

Comprobado aparte con un script que saca las entidades del diagrama y las busca en el
diccionario, las columnas del mapeo y los escenarios, sin leer el reporte del asistente.

Dos cosas que hizo bien y conviene dejar anotadas:

- Encontro un hueco en mis decisiones. Ninguna de D08 a D13 cubria el caso 3, el formato de
  los codigos, y lo dijo en vez de inventarse una cita. De ahi salieron D20 y D21.
- El guardian le bloqueo un `awk` con `/```mermaid/` por el falso positivo conocido. No
  intento rodearlo: rehizo la comprobacion con `grep`, que es lo que pide `AGENTS.md`.

Decisiones suyas que revise y acepte: D22 a D26. La que mas mire fue la de reimportacion,
porque en el reporte parecia contradecirse; al leer la seccion resulto ser una fusion a
tres bandas bien planteada, no una contradiccion.

Quedan sin fijar las versiones de las imagenes de Node y Go, y los comandos del `Makefile`.
Las dos cosas llegan con la siguiente tarea.

El contexto quedo en `AGENTS.md` v3, secciones 7 y 8.
