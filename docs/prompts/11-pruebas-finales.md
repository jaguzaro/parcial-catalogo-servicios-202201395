# 11 - Persistencia, prueba de extremo a extremo y evidencias

| Campo | Valor |
|---|---|
| Fecha | 2026-10-04 |
| Herramienta | Claude Code `2.1.289`, modo headless `claude -p` |
| Modelo | `sonnet` |
| Ejecucion | `docs/evidencias/sesiones/<archivo>.json` y `.meta.txt` |
| Commit base | `0e33b6c` |

## Objetivo

Cerrar las pruebas: P12 de persistencia, una prueba de extremo a extremo que demuestre que
la interfaz funciona de verdad en un navegador, y un comando que guarde la evidencia de una
corrida completa con su fecha y su commit.

## Contexto suministrado

- `AGENTS.md`, reglas permanentes, se carga solo. Sobre todo la seccion 9 y los limites de
  la 4.
- `docs/contexto/enunciado.md`, **solo las secciones 5 y 6**.
- El `Makefile` y `compose.test.yaml` que ya existen.

## Instrucciones

### 1. P12, persistencia (lo mas importante de esta tarea)

`make test-persistence` deja de ser un aviso y pasa a comprobar de verdad, sobre el proyecto
de pruebas `catalogo-test` y nunca sobre los datos de evaluacion:

1. levanta la pila y aplica migraciones;
2. escribe un dato reconocible;
3. reinicia los contenedores **sin borrar el volumen**;
4. comprueba que el dato sigue ahi;
5. termina en 0 si persistio y distinto de 0 si no.

Que imprima lo que va haciendo, para que se pueda leer la salida y creerla.

### 2. `make evidence`

Ejecuta `make check` y guarda la salida en
`docs/evidencias/pruebas/<fecha>-<commit>.txt`, con una cabecera que diga fecha, commit,
version de la herramienta y el comando ejecutado. Si `make check` falla, la evidencia se
guarda igual y el comando termina distinto de 0: una evidencia que solo existe cuando todo
sale bien no sirve de nada.

### 3. Prueba de extremo a extremo

Dos casos, no mas, con Playwright en contenedor y contra la pila de pruebas:

- **Inicio de sesion:** entrar con una cuenta sembrada y llegar al catalogo.
- **Filtro:** aplicar un filtro del catalogo y comprobar que la tabla cambia y que el total
  mostrado corresponde.

Las credenciales salen de variables de entorno o se siembran en la propia prueba. **Ninguna
escrita en el codigo.** Objetivo `make e2e`, y que no entre en `make check` para no volverlo
lento.

Si no logras que corra en contenedor en un tiempo razonable, **dejalo fuera y dilo**. Es la
parte opcional de esta tarea; los puntos 1 y 2 no lo son.

## Restricciones

- **Nunca** ejecutes `make reset-dev`, ni `down -v`, ni nada destructivo fuera del proyecto
  `catalogo-test`. La prueba de persistencia reinicia contenedores, **no borra volumenes**.
- No toques `hooks/`, `.claude/`, `data/`, `docs/contexto/` ni `docs/diseno/`.
- No cambies la aplicacion. Si una prueba falla por un defecto real del codigo, **no lo
  tapes**: reportalo.
- No leas el archivo de entorno.
- No agregues Playwright a la imagen de la aplicacion: va en su propio contenedor.

## Salida esperada

Los objetivos nuevos del `Makefile`, las pruebas, y un reporte final con: la salida de
`make test-persistence`, el archivo que genero `make evidence`, el resultado de la prueba de
extremo a extremo o el motivo de dejarla fuera, y cualquier defecto que hayas encontrado.

## Criterio de aceptacion

1. `make test-persistence` termina en 0 y su salida muestra el dato escrito antes del
   reinicio y leido despues.
2. La prueba de persistencia **no** borra ningun volumen. Dilo en el reporte y muestra el
   comando de reinicio que usas.
3. `make evidence` crea un archivo en `docs/evidencias/pruebas/` con fecha y commit en el
   nombre y en la cabecera, y termina en 0.
4. `make check` sigue terminando en 0, y P01 a P11 siguen pasando.
5. Si `make e2e` queda hecho: termina en 0 y sus dos casos pasan. Si no, el reporte explica
   por que y `make check` no se rompe por ello.
6. `make reset-dev` sigue exigiendo `CONFIRMAR=si` y no se ejecuto en ningun momento.

---

> Lo de arriba es el texto que se envio, sin retocar. Lo de abajo se escribio despues.

## Resultado

**Aceptado.** Sesion: [`20261004-163032-11-pruebas-finales.json`](../evidencias/sesiones/20261004-163032-11-pruebas-finales.json), 20 turnos.

Los tres objetivos quedaron hechos, incluido el opcional.

### Comprobado por el orquestador

```
make test-persistence   exit 0 (ejecutado dos veces)
make e2e                exit 0, 2 pruebas pasadas
make check              exit 0
make evidence           exit 0, genero su archivo con fecha y commit
```

### P12, persistencia

Escribe un dato con un codigo unico, reinicia y lo vuelve a leer:

```
escrito antes del reinicio:  P12-...-61997 | Marcador de persistencia P12
leido despues del reinicio:  P12-...-61997 | Marcador de persistencia P12
P12 OK: el dato persistio tras el reinicio (mismo volumen, mismo contenido)
```

Lo importante de como esta hecho: **el reinicio es `stop` y luego `up`, no `down`**.
Comprobado leyendo `scripts/persistencia.sh`, donde la palabra `down` solo aparece en un
comentario que explica justamente eso. El script tambien compara el nombre del volumen antes
y despues, asi que detectaria un reinicio que lo cambiara. Y opera solo sobre
`catalogo-test`, nunca sobre los datos de evaluacion.

### La prueba de navegador cierra una duda abierta

Las dos tareas de interfaz se entregaron con una limitacion escrita: nadie las habia abierto
en un navegador. `make e2e` lo cierra. Entra con la cuenta de consulta, llega al catalogo,
filtra por un nivel 1 y comprueba que la tabla cambia, que todas las filas son de ese nivel
y que el total mostrado coincide con el que responde la API.

Dos pruebas, no veinte. Playwright corre en su propio contenedor y no entra en `make check`,
para no volver lento el comando que se usa en cada cambio.

### El fallo de esta tarea, y por que se conserva

La primera version de `make e2e` dejaba el catalogo importado en la base de pruebas, y las
pruebas de Go exigen esa base sin catalogo: 15 fallaron. Lo detecto `make evidence` y el
archivo de esa corrida fallida **se conserva a proposito**, porque demuestra que la
evidencia se guarda tambien cuando el resultado es malo. Una evidencia que solo existe
cuando todo sale bien no se distingue de no haber corrido nada.

La correccion: `make e2e` limpia su base al terminar, pase o falle. La corrida buena
inmediatamente posterior termina en 0. Explicado en
[`pruebas/README.md`](../evidencias/pruebas/README.md).

Con esto **P01 a P12 estan completas**.
