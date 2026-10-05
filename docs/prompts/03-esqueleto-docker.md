# 03 - Esqueleto que arranca, Docker y rutina de validacion

| Campo | Valor |
|---|---|
| Fecha | 2026-10-04 |
| Herramienta | Claude Code `2.1.289`, modo headless `claude -p` |
| Modelo | `sonnet` |
| Ejecucion | `docs/evidencias/sesiones/<archivo>.json` y `.meta.txt` |
| Commit base | `b099f6c` |

## Objetivo

Que el proyecto se levante desde un clon limpio con `docker compose up --build -d`, que
`/healthz` responda, y que exista un `make check` que de 0 y sirva para comprobar todo lo
que venga despues. Sin logica de negocio todavia: esto es el andamio y la rutina de
validacion.

## Contexto suministrado

- `AGENTS.md`, reglas permanentes, se carga solo. Seccion 8: arquitectura y convenciones.
- `docs/diseno/arquitectura.md`, el diseno a implementar: piezas, estructura de carpetas,
  subcomandos del binario, construccion en Docker, contenedores, volumen y healthcheck.
- `docs/contexto/enunciado.md`, **solo las secciones 5 y 6**, que son las que fijan los
  requisitos de Docker, reproducibilidad y pruebas.

El modelo de datos no hace falta aqui. Las migraciones del esquema son otra tarea.

## Instrucciones

### Lo que tiene que quedar funcionando

1. **Modulo de Go** con `cmd/catalogo/` y los subcomandos de `arquitectura.md`:
   `servir`, `migrar`, `importar`, `sembrar-demo` y `salud`.
   - `servir` aplica las migraciones pendientes y escucha en el puerto 8080.
   - `salud` consulta `http://127.0.0.1:8080/healthz` y devuelve 0 si responde 200. Lo usa
     el healthcheck, porque la imagen final no trae `curl`.
   - `importar` y `sembrar-demo` **todavia no se implementan**: que impriman que faltan y
     terminen con codigo distinto de 0. Se implementan en su propia tarea.
2. **`internal/config`** lee y valida las variables de entorno, y falla con un mensaje
   claro si falta alguna.
3. **`internal/db`** abre el pool con `pgx/v5` y expone la comprobacion que usa `/healthz`.
4. **`goose` con migraciones embebidas** ya conectado, con una sola migracion inicial que
   deje el esquema vacio preparado. El esquema completo llega despues; lo que importa hoy
   es que `migrar` corra y sea idempotente.
5. **`internal/http`** con el enrutador, `/healthz` y el formato de error de
   `arquitectura.md`. `/healthz` devuelve 200 con `{"estado":"ok"}` si la base responde, y
   503 con `{"estado":"sin_base"}` si no.
6. **Interfaz minima** en `web/` con React, Vite y TypeScript: una sola pantalla que diga
   el nombre del sistema y consulte `/healthz`. Se compila en Docker y el binario la sirve
   con `embed`, desde el mismo origen.
7. **Docker**: `Dockerfile` en tres etapas (web, go, final), `.dockerignore`,
   `compose.yaml` y `compose.test.yaml`, tal como estan descritos en `arquitectura.md`.
   La imagen final corre con un usuario sin privilegios.
8. **`.env.example`** con todas las variables documentadas y valores de ejemplo que se vean
   de ejemplo a simple vista. Incluye las variables de las cuentas de evaluacion
   (administrador y consulta, con su usuario y contrasena) y `APP_PORT`, para que el puerto
   se pueda cambiar si 8080 esta ocupado.
9. **Al menos una prueba de Go de verdad**, por ejemplo sobre `internal/config`. Que
   `make test` pase, pero que no pase por estar vacio.

### El `Makefile`

Objetivos, con un `make help` que los liste:

| Objetivo | Que hace |
|---|---|
| `setup` | Crea `.env` a partir de `.env.example` si no existe. Si existe, no lo toca |
| `up` | `docker compose up --build -d` |
| `down` | Apaga sin borrar el volumen |
| `logs` | Sigue los logs |
| `migrate` | Aplica las migraciones |
| `test` | Pruebas de Go en contenedor, contra el proyecto `catalogo-test` y la base `catalogo_test` |
| `lint` | `gofmt -l` sin salida, `go vet ./...` y revision de tipos de la interfaz, todo en contenedor |
| `check` | `lint`, luego `test`, luego `bash hooks/test-guard.sh`, luego la comprobacion del SHA-256 del Excel contra `docs/contexto/excel-sha256.txt`. Si algo falla, termina distinto de 0 |
| `reset-test` | Destruye y recrea **solo** el proyecto `catalogo-test` |
| `import`, `seed-demo`, `test-persistence` | Existen y avisan de que todavia no estan implementados, terminando distinto de 0 |
| `reset-dev` | Destruye los datos de desarrollo. Exige `CONFIRMAR=si` o no hace nada |

`check` es la rutina que va a usar todo el trabajo siguiente, asi que que sea rapida y que
diga con claridad cual de los cuatro pasos fallo.

## Restricciones

- Todo corre en contenedores. La maquina solo tiene Docker: nada de instalar Go, Node o
  `psql` en el anfitrion.
- **No leas `.env`.** Usa `.env.example`. Si necesitas probar, `make setup` lo crea.
- **Nunca ejecutes `reset-dev`, ni `down -v`, ni nada destructivo fuera del proyecto
  `catalogo-test`.**
- No toques `data/`, `hooks/`, `.claude/`, `docs/contexto/` ni `docs/diseno/`.
- Nada de logica de negocio: ni tablas del modelo, ni autenticacion, ni catalogo, ni
  importador. Solo el andamio.
- Fija las versiones de las imagenes base, sin `latest`.
- Si algo del diseno no se puede implementar tal cual, no lo cambies por tu cuenta:
  implementa lo que si se pueda y dilo en el reporte.

## Salida esperada

Los archivos del esqueleto y un reporte final con: los comandos que ejecutaste y su codigo
de salida, la salida de `/healthz`, la de `make check`, y lo que haya quedado pendiente.

## Criterio de aceptacion

Todos desde la raiz del repositorio y con Docker corriendo:

1. `make setup` termina en 0 y crea `.env` si no existia.
2. `docker compose up --build -d` termina en 0.
3. `docker compose ps` muestra `db` como `healthy` y `app` levantada.
4. `curl -fsS http://localhost:8080/healthz` devuelve 200 con `{"estado":"ok"}`.
5. `curl -fsS http://localhost:8080/` devuelve el HTML de la interfaz.
6. `make check` termina en 0, y su salida deja ver los cuatro pasos.
7. `make migrate` dos veces seguidas: las dos en 0, la segunda sin aplicar nada.
8. `git status --short` no muestra `.env` ni `node_modules`.
9. `docker compose down` y de nuevo `docker compose up -d`: `/healthz` vuelve a responder
   200, sin reconstruir.

---

> Lo de arriba es el texto que se envio, sin retocar. Lo de abajo se escribio despues.

## Resultado

**Aceptado**, en dos corridas del mismo prompt.

| Corrida | Sesion | Resultado |
|---|---|---|
| 1a | [`20261004-012704-...json`](../evidencias/sesiones/20261004-012704-03-esqueleto-docker.json) | **Cortada** en el turno 35 al agotarse la cuota de uso de la herramienta. `exit_code: 1` |
| 2a | [`20261004-033213-...json`](../evidencias/sesiones/20261004-033213-03-esqueleto-docker.json) | Cumplida, 11 turnos |

La primera dejo escrito todo el esqueleto pero sin el `Makefile` y sin ejecutar nada. La
segunda se encontro ese codigo, comprobo que compilaba y arrancaba sin tocarlo, escribio el
`Makefile` y recorrio los nueve criterios.

Las dos corridas tienen SHA-256 distinto porque entre una y otra se anoto aqui mismo el
estado en que habia quedado el trabajo. Ese texto formaba parte del prompt que leyo la
segunda, y le sirvio para saber que faltaba. **No es una iteracion de mejora:** no se
corrigio ninguna instruccion, solo se dejo constancia de lo ya hecho.

### Criterios, comprobados de forma independiente

| # | Criterio | Resultado |
|---|---|---|
| 1 | `make setup` | 0 |
| 2 | `docker compose up --build -d` | 0 con `APP_PORT=18081`. Ver la nota del puerto |
| 3 | `docker compose ps` | `db` y `app` las dos `healthy` |
| 4 | `/healthz` | `HTTP 200` con `{"estado":"ok"}` |
| 5 | `/` | Devuelve el HTML de la interfaz |
| 6 | `make check` | **0**, con los cuatro pasos a la vista |
| 7 | `make migrate` dos veces | Las dos en 0. La segunda: "la base ya estaba en la version 1, nada que aplicar" |
| 8 | `git status` | El archivo de entorno aparece como ignorado, no como pendiente de versionar |
| 9 | `down` y `up -d` sin reconstruir | `/healthz` vuelve a 200, las dos `healthy` |

Salida de `make check`:

```
##### [3/4] hooks/test-guard.sh #####
test-guard: 33 OK, 0 fallas
##### [4/4] SHA-256 del Excel #####
data/CatalogoServicios.xlsx: OK
CHECK OK: los cuatro pasos pasaron
```

Comprobado tambien, sin que lo pidiera el prompt: `/api/x` devuelve 404 con el formato de
error del diseno, `{"error":{"codigo":"NO_ENCONTRADO",...}}`.

### El puerto 8080 ocupado

En la maquina de Joel el 8080 lo atiende `exe_1-adminer-1`, un contenedor de otro proyecto,
asi que `docker compose up` fallaba con "port is already allocated". El trabajador no lo
detuvo, que es lo correcto, y repitio las pruebas con `APP_PORT=18081`.

No es un defecto: `APP_PORT` existe justo para esto. Pero hay que documentarlo en el README,
porque al catedratico le puede pasar lo mismo.

### Pendiente

- `make import`, `make seed-demo` y `make test-persistence` solo avisan y salen distinto de
  0, como pedia el prompt. Se implementan con su tarea.
- `make test` pasa con pruebas de verdad, pero solo cubre `config` y `http`. `internal/db`
  no tiene pruebas.
- El esquema de las 14 tablas no esta: la migracion inicial solo deja la base preparada.

El hook le bloqueo al trabajador un `grep` cuyo texto mencionaba el archivo de entorno. No
intento rodearlo y comprobo el estado con `git status`.
