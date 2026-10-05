# 14 - Que el sistema quede usable con un solo comando

| Campo | Valor |
|---|---|
| Fecha | 2026-10-04 |
| Herramienta | Claude Code `2.1.289`, modo headless `claude -p` |
| Modelo | `sonnet` |
| Ejecucion | `docs/evidencias/sesiones/<archivo>.json` y `.meta.txt` |
| Commit base | `71a5af5` |

## Objetivo

Hoy, en un clon limpio, `docker compose up --build -d` levanta el sistema pero deja la base
sin cuentas y sin catalogo, asi que las credenciales que documenta el README devuelven 401 y
la aplicacion parece rota. Quien la evalue puede ejecutar solo ese comando.

Que tras ese unico comando el sistema quede usable: con el catalogo importado y las cuentas
creadas.

## Contexto suministrado

- `AGENTS.md`, reglas permanentes, se carga solo. Secciones 8 y 9.
- `cmd/catalogo/main.go`, donde estan los subcomandos.
- `internal/demo/demo.go` y `internal/importador/`, que ya hacen el trabajo y **no cambian
  de comportamiento**.

## Instrucciones

1. Al arrancar `servir`, despues de aplicar las migraciones, ejecutar la importacion del
   Excel y luego el sembrado de datos de demostracion, **en ese orden**. El sembrado crea
   asignaciones sobre servicios importados, asi que al reves falla.
2. Las dos operaciones ya son idempotentes: en un arranque posterior no deben crear nada.
   Reutiliza el codigo existente; no lo dupliques.
3. **Un arranque nunca debe quedarse sin servir por esto.** Si la importacion o el sembrado
   fallan, por ejemplo si falta el Excel, se registra el error en el log y el servidor
   arranca igual, para que la ruta de salud y la interfaz respondan y se pueda diagnosticar.
4. Que quede claro en el log que se hizo y que se omitio por existir ya.
5. Una variable de entorno nueva lo desactiva, documentada en `.env.example` con su valor
   por defecto. Con la variable en falso, el arranque se comporta como hoy.
6. `make import` y `make seed-demo` siguen funcionando por separado, igual que ahora.

## Restricciones

- No cambies el comportamiento de la importacion ni del sembrado, ni sus mensajes.
- No toques migraciones, pruebas, interfaz, `hooks/`, `.claude/`, `data/`,
  `docs/contexto/` ni `docs/diseno/`.
- No leas el archivo de entorno. Nunca `make reset-dev` ni `down -v` fuera de
  `catalogo-test`.

## Salida esperada

Los cambios y un reporte final con el nombre de la variable nueva, lo que imprime el log en
el primer arranque y en el segundo, y los comandos ejecutados con su codigo de salida.

## Criterio de aceptacion

Sobre un **clon limpio**, con un nombre de carpeta distinto para que el volumen sea nuevo:

1. `make setup` y luego `docker compose up --build -d`, y **nada mas**, terminan en 0.
2. Tras ese unico comando: iniciar sesion con las credenciales de `.env.example` devuelve
   200, el catalogo responde 46 servicios de nivel 2 y 12 de nivel 1.
3. `docker compose restart app`: el log dice que no creo nada, y los conteos siguen en 12 y
   46.
4. `make check` termina en 0 y P01 a P11 siguen pasando.
5. `make import` y `make seed-demo` a mano siguen terminando en 0.
6. Con la variable nueva en falso, el arranque no importa ni siembra, y la ruta de salud
   responde igual.

---

> Lo de arriba es el texto que se envio, sin retocar. Lo de abajo se escribio despues.

## Resultado

**Aceptado.** Sesion: [`20261004-212304-14-arranque-autonomo.json`](../evidencias/sesiones/20261004-212304-14-arranque-autonomo.json), 26 turnos.

La variable nueva es `AUTO_INICIO`, con `true` por defecto. El arranque aplica migraciones,
importa y siembra, en ese orden, y nunca impide servir: si algo falla lo registra y el
servidor arranca igual.

### De donde salio esta tarea

De probar lo que nadie habia probado: clonar el repositorio en una carpeta nueva y ejecutar
**solo** lo que dice la documentacion. Aparecieron dos defectos que ninguna prueba
automatica podia ver, porque las pruebas trabajan contra una base que ellas mismas preparan:

1. `make seed-demo` terminaba con **codigo 2** en un clon limpio. Crea tres asignaciones de
   responsable sobre servicios importados, asi que necesita que `make import` haya corrido
   antes, y el README los tenia en el orden contrario.
2. Con solo `docker compose up --build -d`, el sistema levantaba pero sin cuentas: las
   credenciales que documenta el README devolvian 401. Para quien evalua, eso se ve como una
   aplicacion que no funciona.

Lo primero se corrigio en el README. Lo segundo es esta tarea.

### Comprobacion independiente

Sobre un volumen nuevo, ejecutando unicamente el comando documentado:

```
docker compose up --build -d             exit 0
ruta de salud                            200, estado ok
login con las credenciales del ejemplo   200
servicios de nivel 1                     12
servicios de nivel 2                     46
```

Tras reiniciar el contenedor de la aplicacion:

```
servicios de nivel 2                     46, no se duplicaron
log: "importacion sin cambios, el catalogo ya estaba cargado ... omitidos=78"
make check                               exit 0
```

### Desviacion del asistente, y por que se acepta

El criterio pedia probarlo en un clon limpio en otra carpeta. El asistente no lo hizo: lo
probo sobre el proyecto de pruebas con un volumen nuevo, y **explico por que** — clonar
fuera de la carpeta del proyecto se lo prohibe la seccion 4 de `AGENTS.md`, y un clon no
habria incluido sus propios cambios sin versionar.

El razonamiento es correcto en los dos puntos. La parte que el no podia cubrir la hizo el
orquestador, que si puede trabajar fuera del proyecto.
