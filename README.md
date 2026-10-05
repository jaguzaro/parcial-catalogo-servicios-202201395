# Catalogo de servicios de TI

Aplicacion web que sistematiza el catalogo de servicios de TI de
`data/CatalogoServicios.xlsx`, con usuarios, roles y estructura organizacional
(Empresa, Area, Departamento, Seccion, Puesto, Usuario). Autenticacion local, PostgreSQL y
ejecucion completa con Docker.

Joel Guzaro, carne 202201395. USAC, Laboratorio de Software Avanzado.

Como se resolvio: [`docs/RESOLUCION.md`](docs/RESOLUCION.md).

## Requisitos

Solo Docker. No hace falta instalar Go, Node ni PostgreSQL: todo corre en contenedores.

- Docker 28 o superior y Docker Compose v2, es decir `docker compose` y no
  `docker-compose`. Probado con Docker 28.3.2.
- GNU Make.

## Puesta en marcha

```bash
git clone <url-del-repositorio>
cd parcial-catalogo-servicios-202201395

make setup                      # crea .env a partir de .env.example
docker compose up --build -d    # construye y levanta
```

La primera construccion tarda varios minutos, porque compila la interfaz y el binario.

Comprobacion:

```bash
curl http://localhost:8080/healthz      # {"estado":"ok"}
docker compose ps                       # db y app en healthy
```

La aplicacion queda en **http://localhost:8080**.

Las migraciones se aplican al arrancar. Para cargar los datos:

```bash
make seed-demo    # cuentas de evaluacion y 3 asignaciones de responsable
make import       # importa el Excel: 12 servicios de nivel 1 y 46 de nivel 2
```

Los dos comandos son idempotentes: repetirlos no duplica registros.

### Si el puerto 8080 esta ocupado

`docker compose up` falla con `port is already allocated`. Para ver que lo ocupa:

```bash
docker ps --format '{{.Names}} {{.Ports}}' | grep 8080
```

La solucion es cambiar `APP_PORT` en `.env`. El resto de este archivo asume el 8080.

## Cuentas de evaluacion

Las crea `make seed-demo` leyendo `.env`. Los valores de ejemplo son:

| Rol | Usuario | Contrasena |
|---|---|---|
| Administrador | `admin.ejemplo` | `EJEMPLO-admin-cambiar-1` |
| Consulta | `consulta.ejemplo` | `EJEMPLO-consulta-cambiar-1` |

Para usar otras credenciales hay que editar las variables `DEMO_*` de `.env` antes de
ejecutar `make seed-demo`. El comando nunca modifica un usuario que ya existe, ni su
contrasena; cambiarla despues se hace desde la aplicacion.

`.env` no se versiona. En el repositorio solo esta `.env.example`.

## Roles

- **Administrador**: mantiene usuarios, estructura organizacional, servicios y catalogos.
- **Consulta**: solo lectura de los datos funcionales, sin acceso a hashes ni secretos.

La autorizacion se aplica en el servidor. `TestP03` recorre las 38 rutas de escritura de la
API y exige 403 para el rol consulta.

## Variables de entorno

Todas estan documentadas en `.env.example`.

| Variable | Para que |
|---|---|
| `POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PASSWORD` | Base de datos |
| `APP_PORT` | Puerto de la aplicacion en la maquina anfitriona |
| `SESSION_TTL` | Duracion de la sesion |
| `COOKIE_SECURE` | `false` para `http://localhost`; `true` detras de HTTPS |
| `EXCEL_PATH` | Ruta del Excel dentro del contenedor, montado de solo lectura |
| `DEMO_*` | Cuentas de evaluacion |
| `APP_TEST_PORT` | Puerto del entorno de pruebas |

## Pruebas

```bash
make check              # lint, pruebas, limites del harness y SHA-256 del Excel
make test-persistence   # P12: reinicia sin borrar el volumen y comprueba los datos
make e2e                # dos pruebas de navegador: inicio de sesion y un filtro
make evidence           # ejecuta make check y guarda la salida con fecha y commit
```

`make check` debe terminar con codigo 0. Las pruebas usan un proyecto y una base aparte,
`catalogo-test` y `catalogo_test`, asi que no tocan los datos de evaluacion.

Los doce escenarios del enunciado:

| Escenarios | Donde estan |
|---|---|
| P01 a P11 | `tests/integracion/`, con el identificador en el nombre (`TestP01_...`) |
| P12 | `make test-persistence`. Reiniciar contenedores no cabe en una prueba de Go |

Salidas reales de las ejecuciones: [`docs/evidencias/pruebas/`](docs/evidencias/pruebas/).

## Comandos

```bash
make help               # lista todos los objetivos
make up                 # construye y levanta
make down               # apaga sin borrar datos
make logs               # sigue los logs
make migrate            # aplica las migraciones pendientes
make import             # importa el Excel
make seed-demo          # cuentas de evaluacion y asignaciones
make reset-test         # destruye y recrea solo el entorno de pruebas
```

Los logs de un servicio concreto se consultan con `docker compose logs -f app` o
`docker compose logs -f db`.

## Apagado y reinicio

Apagado normal, que conserva los datos:

```bash
make down
docker compose up -d
```

Reinicio destructivo, que borra la base de evaluacion:

```bash
make reset-dev CONFIRMAR=si
```

Sin `CONFIRMAR=si` no hace nada. Despues hay que volver a ejecutar `make seed-demo` y
`make import`.

## El archivo original

`data/CatalogoServicios.xlsx` se conserva sin modificaciones. Su SHA-256 esta fijado en
`docs/contexto/excel-sha256.txt` y `make check` lo verifica en cada ejecucion:

```bash
shasum -a 256 -c docs/contexto/excel-sha256.txt
```

## Estructura

```text
cmd/catalogo/      binario: servir, migrar, importar, sembrar-demo, salud
internal/          auth, catalogo, db y migraciones, demo, http, importador,
                   organizacion, usuarios, web
web/               interfaz (React, Vite, TypeScript), servida por el binario
tests/integracion/ escenarios P01 a P11
e2e/               pruebas de navegador (Playwright)
scripts/           analisis del Excel, persistencia, evidencias
docs/              contexto, prompts, diseno y evidencias
hooks/             limites de operacion del asistente de IA
reports/           auditoria con subagentes
data/              el Excel original, de solo lectura
```
