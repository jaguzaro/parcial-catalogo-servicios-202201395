# pruebas/

Salidas reales de comandos, sin editar. Los nombres con fecha y commit los genera
`make evidence`.

| Archivo | Que es |
|---|---|
| `<fecha>-<commit>.txt` | Una corrida de `make check` completa, guardada por `make evidence`. La cabecera trae fecha, commit, versiones y el comando |
| `06-p03-sin-middleware.txt` | `TestP03` ejecutado **a proposito** con el middleware de rol quitado de una ruta. Falla solo en esa, lo que demuestra que la prueba detecta lo que dice detectar. El middleware quedo restaurado |
| `06-make-check.txt`, `07-make-check.txt` | Corridas de `make check` al cerrar esas tareas |
| `07-importacion.txt` | Las dos primeras importaciones seguidas, con sus resumenes y sus incidencias |
| `08-verificacion-criterios.txt` | Comprobacion de los criterios del catalogo contra la API levantada |

## La corrida de `make check` que aparece fallida

`20261004-163442-0e33b6c.txt` termina con codigo 2 y 15 pruebas en rojo. **Se conserva a
proposito**, por dos motivos.

El primero: demuestra que `make evidence` guarda la salida tambien cuando el resultado es
malo. Una evidencia que solo existe cuando todo sale bien no sirve para nada, porque no se
puede distinguir de no haber corrido nada.

El segundo: ese fallo fue real y vale como ejemplo. La primera version de `make e2e` dejaba
el catalogo importado en la base de pruebas, y las pruebas de Go exigen esa base sin
catalogo. Quien lo escribio lo detecto con este mismo comando y lo corrigio: ahora `make
e2e` limpia su base al terminar, pase o falle. La corrida buena inmediatamente posterior,
`20261004-163555-0e33b6c.txt`, termina en 0.

El ciclo completo de este tipo de correcciones esta en
[`../harness/ciclo-01.md`](../harness/ciclo-01.md).
