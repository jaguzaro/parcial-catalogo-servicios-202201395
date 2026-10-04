# logs/

`hooks.log` lo escribe `hooks/guard.sh`. Es una linea JSON por intento bloqueado, con
fecha UTC, herramienta, comando, archivo y motivo.

Se versiona a proposito. Es la prueba de que los limites se aplican de verdad y no solo
estan descritos. Conviene revisarlo despues de cada ejecucion del asistente, porque a
veces muestra intentos que su reporte no menciona.

```bash
jq -r '[.ts, .tool, .reason] | @tsv' logs/hooks.log
```
