# Catalogo de servicios de TI

Aplicacion web que sistematiza el catalogo de servicios de TI de
`data/CatalogoServicios.xlsx`, con usuarios, roles y estructura organizacional
(Empresa, Area, Departamento, Seccion, Puesto, Usuario). Autenticacion local, PostgreSQL y
ejecucion completa con Docker.

**Joel Guzaro, carne 202201395.** USAC, Laboratorio de Software Avanzado.
Como se resolvio, con decisiones y evidencias: [`docs/RESOLUCION.md`](docs/RESOLUCION.md).

## Requisitos

Solo Docker. Ni Go, ni Node, ni PostgreSQL en la maquina: todo corre en contenedores.

- Docker 28 o superior y Docker Compose v2 (`docker compose`, no `docker-compose`).
  Probado con Docker 28.3.2.
- GNU Make.

## Antes de empezar: el puerto

La aplicacion usa el **8080**. Si lo tienes ocupado, `docker compose up` falla con
`port is already allocated`. Para comprobarlo:

```bash
docker ps --format '{{.Names}} {{.Ports}}' | grep 8080
```

Si esta ocupado, cambia `APP_PORT` en `.env` despues del primer paso, o exportalo:
`export APP_PORT=8081`. El resto de este archivo asume 8080.

## Puesta en marcha

```bash
git clone <url-del-repositorio>
cd parcial-catalogo-servicios-202201395

make setup                      # crea .env a partir de .env.example
docker compose up --build -d    # construye y levanta
```

La primera construccion tarda varios minutos: compila la interfaz y el binario.

Comprueba que esta arriba:

```bash
curl http://localhost:8080/healthz      # {"estado":"ok"}
docker compose ps                       # db y app en healthy
```

Abre **http://localhost:8080**.

Las migraciones se aplican solas al arrancar. Para cargar los datos:

```bash
make seed-demo    # cuentas de evaluacion y 3 asignaciones de responsable
make import       # importa el Excel: 12 servicios de nivel 1 y 46 de nivel 2
```

Los dos son idempotentes: ejecutarlos dos veces no duplica nada.

## Cuentas de evaluacion

Las crea `make seed-demo` leyendo `.env`. Los valores de ejemplo son:

| Rol | Usuario | Contrasena |
|---|---|---|
| Administrador | `admin.ejemplo` | `EJEMPLO-admin-cambiar-1` |
| Consulta | `consulta.ejemplo` | `EJEMPLO-consulta-cambiar-1` |

Son credenciales de demostracion y se ven de demostracion a proposito. Para cambiarlas,
edita `DEMO_*` en `.env` **antes** de `make seed-demo`. El comando nunca modifica un
usuario que ya existe, tampoco su contrasena: para cambiarla despues, hazlo desde la
aplicacion.

`.env` no se versiona. En el repositorio solo esta `.env.example`.

## Que hace cada rol

- **Administrador**: mantiene usuarios, estructura organizacional, servicios y catalogos.
- **Consulta**: solo lee los datos funcionales. No ve hashes ni secretos.

La autorizacion se aplica **en el servidor**. Que la interfaz oculte un boton no es el
control: `TestP03` recorre las 38 rutas de escritura de la API y exige 403 para el rol
consulta.

## Variables de entorno

Todas estan documentadas en `.env.example`.

| Variable | Para que |
|---|---|
| `POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PASSWORD` | Base de datos |
| `APP_PORT` | Puerto de la aplicacion en la maquina (8080) |
| `SESSION_TTL` | Duracion de la sesion (8h) |
| `COOKIE_SECURE` | `false` para `http://localhost`. En HTTPS, `true` |
| `EXCEL_PATH` | Ruta del Excel dentro del contenedor, montado de solo lectura |
| `DEMO_*` | Cuentas de evaluacion |
| `APP_TEST_PORT` | Puerto del entorno de pruebas (18080) |

## Pruebas

```bash
make check              # lint, pruebas, limites del harness y SHA-256 del Excel
make test-persistence   # P12: reinicia sin borrar el volumen y comprueba que el dato sigue
make e2e                # dos pruebas de navegador: inicio de sesion y un filtro
make evidence           # corre make check y guarda la salida con fecha y commit
```

`make check` tiene que terminar en 0. Las pruebas usan un proyecto y una base aparte
(`catalogo-test`, `catalogo_test`), asi que **no tocan los datos de evaluacion**.

Los doce escenarios del enunciado:

| | Donde esta |
|---|---|
| P01 a P11 | `tests/integracion/`, con el identificador en el nombre (`TestP01_...`) |
| P12 | `make test-persistence`. Reiniciar contenedores no cabe en una prueba de Go, asi que vive en su propio comando |

## Comandos

```bash
make help               # lista todos
make up                 # construye y levanta
make down               # apaga SIN borrar datos
make logs               # sigue los logs
make migrate            # aplica migraciones pendientes
make import             # importa el Excel
make seed-demo          # cuentas de evaluacion y asignaciones
make reset-test         # destruye y recrea SOLO el entorno de pruebas
```

Logs de un servicio concreto: `docker compose logs -f app` o `... db`.

## Apagar y reiniciar

**Apagado normal**, conserva los datos:

```bash
make down
docker compose up -d     # vuelve a levantar con todo donde estaba
```

**Reinicio destructivo**, borra la base de evaluacion:

```bash
make reset-dev CONFIRMAR=si
```

Sin `CONFIRMAR=si` no hace nada. Despues hay que volver a ejecutar `make seed-demo` y
`make import`.

## El archivo original

`data/CatalogoServicios.xlsx` se conserva sin modificaciones. Su SHA-256 esta fijado en
`docs/contexto/excel-sha256.txt` y `make check` lo verifica en cada corrida:

```bash
shasum -a 256 -c docs/contexto/excel-sha256.txt
```

## Estructura

```text
cmd/catalogo/      binario: servir, migrar, importar, sembrar-demo, salud
internal/          auth, catalogo, db y migraciones, demo, http, importador,
                   organizacion, usuarios, web
web/               interfaz (React, Vite, TypeScript), servida por el binario
tests/integracion/ P01 a P11
e2e/               pruebas de navegador (Playwright)
scripts/           analisis del Excel, persistencia, evidencias
docs/              contexto, prompts, diseno y evidencias
hooks/             limites de operacion del asistente de IA
reports/           auditoria con subagentes
data/              el Excel original, de solo lectura
```
