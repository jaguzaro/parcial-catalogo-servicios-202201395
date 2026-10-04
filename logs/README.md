# logs/

`hooks.log` lo escribe `hooks/guard.sh`: una línea JSON por intento bloqueado, con fecha
UTC, herramienta, comando, archivo y motivo.

Se versiona a propósito. Es la prueba de que los límites se aplican de verdad y no solo
están descritos, y conviene revisarlo después de cada ejecución del asistente: a veces
muestra intentos que su reporte no menciona.

```bash
jq -r '[.ts, .tool, .reason] | @tsv' logs/hooks.log
```
