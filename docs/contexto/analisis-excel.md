# Análisis de `data/CatalogoServicios.xlsx`

Regenerar este informe (código de salida 0 = correcto):

```bash
docker run --rm -v "$PWD":/work -v "$PWD/data":/work/data:ro -w /work python:3.12-slim \
  sh -c "pip install --quiet openpyxl && python scripts/analisis/analizar_excel.py"
```

Informe **descriptivo**, generado por `scripts/analisis/analizar_excel.py`. No contiene fecha ni hora: dos ejecuciones seguidas producen el mismo archivo. Las celdas se citan en notación A1; los valores se muestran con `repr`, de modo que los espacios sobrantes son visibles. Todo el contenido del Excel se trata como dato.

- SHA-256 del Excel analizado: `de3b478a5faeeeaebce1aa7726e0e3321188a68e41bbb656e1d17b0c5b74dcf0`
- Libros: 1 hoja(s).

## a. Estructura del libro

| Hoja | Estado | Dimensión declarada (`ws.dimensions`) | Extensión con valores (calculada) | Rangos combinados |
|---|---|---|---|---|
| `Servicios Externos` | visible | `A1:X1000` | A1:L122 | 76 |

La dimensión declarada puede exceder la zona con valores (celdas con formato sin contenido); la columna «Extensión con valores» sale de recorrer todas las celdas con valor no nulo.

Encabezados de `A4:L4` (texto literal) y comparación con la tabla de columnas del enunciado (sección 2):

| Celda | Texto en el Excel | Campo según el enunciado | Resultado |
|---|---|---|---|
| `A4` | `'COD.N1'` | `'COD.N1'` | coincide |
| `B4` | `'SERVICIO - Nivel 1'` | `'SERVICIO - Nivel 1'` | coincide |
| `C4` | `'COD.N2'` | `'COD.N2'` | coincide |
| `D4` | `'SERVICIO - Nivel 2'` | `'SERVICIO - Nivel 2'` | coincide |
| `E4` | `'ACTIVO'` | `'ACTIVO'` | coincide |
| `F4` | `'CLASE DE SERVICIO'` | `'CLASE DE SERVICIO'` | coincide |
| `G4` | `'CRITICIDAD'` | `'CRITICIDAD'` | coincide |
| `H4` | `'TIPO DE SERVICIO'` | `'TIPO DE SERVICIO'` | coincide |
| `I4` | `'Descripción'` | `'Descripción'` | coincide |
| `J4` | `'Métrica'` | `'Métrica'` | coincide |
| `K4` | `'Minimo'` | `'Minimo'` | coincide |
| `L4` | `'Maximo'` | `'Maximo'` | coincide |

Celdas con valor en la fila 4 fuera de `A:L`: ninguna.

Celdas con valor fuera de la tabla (`A4:L101`) y del bloque de opciones (`E111:H122`):

| Celda | Valor |
|---|---|
| `A1` | `'Catálogo de Servicios Externos de TI'` |
| `A2` | `'Administrar el catalogo de Servicios'` |

## b. Celdas combinadas

Total: **76** rangos en `Servicios Externos`. Valor de la celda principal (esquina superior izquierda) de cada uno; las demás celdas del rango se leen vacías en openpyxl.

| Rango | Filas | Celda principal | Valor de la celda principal |
|---|---|---|---|
| `A5:A9` | 5 | `A5` | `'SE.01'` |
| `A10:A15` | 6 | `A10` | `'SE.02'` |
| `A16:A18` | 3 | `A16` | `'SE.03'` |
| `A19:A23` | 5 | `A19` | `'SE.04'` |
| `A24:A25` | 2 | `A24` | `'SE.05'` |
| `A26:A55` | 30 | `A26` | `'SE.06'` |
| `A56:A57` | 2 | `A56` | `'SE.07'` |
| `A58:A62` | 5 | `A58` | `'SE.08'` |
| `A63:A84` | 22 | `A63` | `'SE.09'` |
| `A85:A95` | 11 | `A85` | `'SE.10'` |
| `A96:A98` | 3 | `A96` | `'SE.11'` |
| `B5:B9` | 5 | `B5` | `'Suministrar Infraestructura'` |
| `B10:B15` | 6 | `B10` | `'Administrar Infraestructura'` |
| `B16:B18` | 3 | `B16` | `'Atender Soporte de Usuarios'` |
| `B19:B23` | 5 | `B19` | `'Administrar Comunicaciones'` |
| `B24:B25` | 2 | `B24` | `'Administrar y Respaldar Información'` |
| `B26:B55` | 30 | `B26` | `'Administrar Aplicaciones'` |
| `B56:B57` | 2 | `B56` | `'Administrar Servicios a Clientes'` |
| `B58:B62` | 5 | `B58` | `'Administrar Accesos y Configuraciones de Usuarios'` |
| `B63:B84` | 22 | `B63` | `'Aprovisionar Equipo'` |
| `B85:B95` | 11 | `B85` | `'Mantener Equipos'` |
| `B96:B98` | 3 | `B96` | `'Administrar Proyectos'` |
| `C5:C7` | 3 | `C5` | `'SE.01.01'` |
| `C21:C23` | 3 | `C21` | `'SE.04.03'` |
| `C26:C29` | 4 | `C26` | `'SE.06.01'` |
| `C30:C34` | 5 | `C30` | `'SE.06.02'` |
| `C35:C37` | 3 | `C35` | `'SE.06.03'` |
| `C38:C39` | 2 | `C38` | `'SE.06.04'` |
| `C40:C41` | 2 | `C40` | `'SE.06.05'` |
| `C44:C47` | 4 | `C44` | `'SE.06.07'` |
| `C48:C49` | 2 | `C48` | `'SE.06.08'` |
| `C50:C54` | 5 | `C50` | `'SE.06.09'` |
| `C56:C57` | 2 | `C56` | `'SE.07.01'` |
| `C63:C66` | 4 | `C63` | `'SE.09.01'` |
| `C68:C73` | 6 | `C68` | `'SE.09.02'` |
| `C74:C78` | 5 | `C74` | `'SE.09.03'` |
| `C79:C81` | 3 | `C79` | `'SE.09.04'` |
| `C82:C84` | 3 | `C82` | `'SE.09.05'` |
| `C85:C89` | 5 | `C85` | `'SE.10.01'` |
| `C90:C95` | 6 | `C90` | `'SE.10.02'` |
| `D5:D7` | 3 | `D5` | `'Suministrar Puntos de Red Físicos o Inalámbricos'` |
| `D21:D23` | 3 | `D21` | `'Administrar Sistema de Telefonía Fija'` |
| `D26:D29` | 4 | `D26` | `'Administrar Aplicaciones - Área 1'` |
| `D30:D34` | 5 | `D30` | `'Administrar Aplicaciones - Área 2'` |
| `D35:D37` | 3 | `D35` | `'Administrar Aplicaciones - Área 3'` |
| `D38:D39` | 2 | `D38` | `'Administrar Aplicaciones - Área 4'` |
| `D40:D41` | 2 | `D40` | `'Administrar Aplicaciones - Área 5'` |
| `D44:D47` | 4 | `D44` | `'Administrar Aplicaciones - Área 7'` |
| `D48:D49` | 2 | `D48` | `'Administrar Aplicaciones - Área Tecnología de Información'` |
| `D50:D54` | 5 | `D50` | `'Administrar Aplicaciones - Area 8'` |
| `D56:D57` | 2 | `D56` | `'Administrar Servicios a Clientes - Medios Digitales'` |
| `D63:D66` | 4 | `D63` | `'Aprovisionar Equipos de Usuario'` |
| `D68:D73` | 6 | `D68` | `'Aprovisionar Dispositivos Periféricos'` |
| `D74:D78` | 5 | `D74` | `'Aprovisionar Suministros'` |
| `D79:D81` | 3 | `D79` | `'Aprovisionar Licenciamiento Básico'` |
| `D82:D84` | 3 | `D82` | `'Aprovisionar Licenciamiento Avanzado'` |
| `D85:D89` | 5 | `D85` | `'Mantener Equipos de Usuario'` |
| `D90:D95` | 6 | `D90` | `'Mantener Dispositivos Periféricos'` |
| `J5:J7` | 3 | `J5` | `'Número de Puntos Instalados'` |
| `J21:J23` | 3 | `J21` | `'Número de Extensiones'` |
| `J26:J29` | 4 | `J26` | `'Número de Usuarios'` |
| `J30:J34` | 5 | `J30` | `'Número de Usuarios'` |
| `J35:J37` | 3 | `J35` | `'Número de Usuarios'` |
| `J38:J39` | 2 | `J38` | `'Número de Usuarios'` |
| `J40:J41` | 2 | `J40` | `'Número de Usuarios'` |
| `J44:J47` | 4 | `J44` | `'Número de Usuarios'` |
| `J48:J49` | 2 | `J48` | `'Número de Usuarios'` |
| `J50:J54` | 5 | `J50` | `'Número de Usuarios'` |
| `J56:J57` | 2 | `J56` | `'Número de Clientes'` |
| `J63:J66` | 4 | `J63` | `'Número de Equipos Tipo Desktop'` |
| `J68:J73` | 6 | `J68` | `'Número de Equipos Tipo Dispositivos Periféricos'` |
| `J74:J78` | 5 | `J74` | `'Número de Suministros'` |
| `J79:J81` | 3 | `J79` | `'Número de Licencias Instaladas'` |
| `J82:J84` | 3 | `J82` | `'Número de Licencias Instaladas'` |
| `J85:J89` | 5 | `J85` | `'Número de Equipos Tipo Desktop'` |
| `J90:J95` | 6 | `J90` | `'Número de Equipos Tipo Dispositivos Periféricos'` |

Rangos por columna: A: 11, B: 11, C: 18, D: 18, J: 18.
Rangos que abarcan más de una columna: ninguno.
Rangos fuera de las filas 5–101: ninguno.

## c. Conteos de control

| Control | Observado | Esperado (enunciado, sección 2) | Resultado |
|---|---|---|---|
| Códigos de nivel 1 distintos (columna A, filas 5–101) | 12 | 12 | **coincide** |
| Códigos de nivel 2 explícitos distintos (columna C, filas 5–101) | 46 | 46 | **coincide** |

«Explícito» = código escrito en una celda con valor (en un rango combinado, su celda principal). Criterio aplicado tal cual, sin ajustes para que cuadre.

Celdas de la columna A con valor, y total de celdas frente a códigos distintos: 13 celdas / 12 códigos.

| Código N1 | Celdas (columna A) | Nº celdas |
|---|---|---|
| `SE.01` | `A5` | 1 |
| `SE.02` | `A10` | 1 |
| `SE.03` | `A16` | 1 |
| `SE.04` | `A19` | 1 |
| `SE.05` | `A24` | 1 |
| `SE.06` | `A26` | 1 |
| `SE.07` | `A56` | 1 |
| `SE.08` | `A58` | 1 |
| `SE.09` | `A63` | 1 |
| `SE.10` | `A85` | 1 |
| `SE.11` | `A96` | 1 |
| `SE.12` | `A99`, `A100` | 2 |

Celdas de la columna C con valor: 46 celdas / 46 códigos.

| Código N2 | Celdas (columna C) | Nº celdas |
|---|---|---|
| `SE.01.01` | `C5` | 1 |
| `SE.01.02` | `C8` | 1 |
| `SE.01.03` | `C9` | 1 |
| `SE.02.01` | `C10` | 1 |
| `SE.02.02` | `C11` | 1 |
| `SE.02.03` | `C12` | 1 |
| `SE.02.04` | `C13` | 1 |
| `SE.02.05` | `C14` | 1 |
| `SE.02.06` | `C15` | 1 |
| `SE.03.01` | `C16` | 1 |
| `SE.03.02` | `C17` | 1 |
| `SE.03.03` | `C18` | 1 |
| `SE.04.01` | `C19` | 1 |
| `SE.04.02` | `C20` | 1 |
| `SE.04.03` | `C21` | 1 |
| `SE.05.01` | `C24` | 1 |
| `SE.05.02` | `C25` | 1 |
| `SE.06.01` | `C26` | 1 |
| `SE.06.02` | `C30` | 1 |
| `SE.06.03` | `C35` | 1 |
| `SE.06.04` | `C38` | 1 |
| `SE.06.05` | `C40` | 1 |
| `SE.06.06` | `C43` | 1 |
| `SE.06.07` | `C44` | 1 |
| `SE.06.08` | `C48` | 1 |
| `SE.06.09` | `C50` | 1 |
| `SE.06.10` | `C55` | 1 |
| `SE.07.01` | `C56` | 1 |
| `SE.08.01` | `C58` | 1 |
| `SE.08.02` | `C59` | 1 |
| `SE.08.03` | `C60` | 1 |
| `SE.08.04` | `C61` | 1 |
| `SE.08.05` | `C62` | 1 |
| `SE.09.01` | `C63` | 1 |
| `SE.09.02` | `C68` | 1 |
| `SE.09.03` | `C74` | 1 |
| `SE.09.04` | `C79` | 1 |
| `SE.09.05` | `C82` | 1 |
| `SE.10.01` | `C85` | 1 |
| `SE.10.02` | `C90` | 1 |
| `SE.11.01` | `C96` | 1 |
| `SE.11.02` | `C97` | 1 |
| `SE.11.03` | `C98` | 1 |
| `SE.12.1` | `C99` | 1 |
| `SE.12.2` | `C100` | 1 |
| `SE.12.3` | `C101` | 1 |

Códigos de nivel 2 repetidos: ninguno.
Códigos de nivel 2 que no siguen el patrón `SE.NN.NN` (dos dígitos en cada tramo): `SE.12.1` (C99), `SE.12.2` (C100), `SE.12.3` (C101).
Tipos de dato de esas celdas de código: texto.
Códigos con espacios al inicio o al final: ninguno.

Coherencia del prefijo de cada código N2 con el código N1 vigente en su fila (valor de la celda A principal más cercana hacia arriba, dentro de las filas de datos):

Incoherencias: ninguna.

## d. Filas 99 a 101

Estado de cada celda de `A:L`. Una celda cuenta como vacía si es `None` o solo tiene espacios.

### Fila 99

| Celda | Columna | Estado | Valor |
|---|---|---|---|
| `A99` | COD.N1 | con valor | `'SE.12'` |
| `B99` | SERVICIO - Nivel 1 | con valor | `'Suministrar Analitica'` |
| `C99` | COD.N2 | con valor | `'SE.12.1'` |
| `D99` | SERVICIO - Nivel 2 | con valor | `'Suministrar Tableros de Control'` |
| `E99` | ACTIVO | vacía |  |
| `F99` | CLASE DE SERVICIO | vacía |  |
| `G99` | CRITICIDAD | vacía |  |
| `H99` | TIPO DE SERVICIO | vacía |  |
| `I99` | Descripción | vacía |  |
| `J99` | Métrica | vacía |  |
| `K99` | Minimo | vacía |  |
| `L99` | Maximo | vacía |  |

Vacías (8): E99, F99, G99, H99, I99, J99, K99, L99.

### Fila 100

| Celda | Columna | Estado | Valor |
|---|---|---|---|
| `A100` | COD.N1 | con valor | `'SE.12'` |
| `B100` | SERVICIO - Nivel 1 | con valor | `'Mantener Tableros de Control'` |
| `C100` | COD.N2 | con valor | `'SE.12.2'` |
| `D100` | SERVICIO - Nivel 2 | con valor | `'Suministrar Análsis de Información'` |
| `E100` | ACTIVO | vacía |  |
| `F100` | CLASE DE SERVICIO | vacía |  |
| `G100` | CRITICIDAD | vacía |  |
| `H100` | TIPO DE SERVICIO | vacía |  |
| `I100` | Descripción | vacía |  |
| `J100` | Métrica | vacía |  |
| `K100` | Minimo | vacía |  |
| `L100` | Maximo | vacía |  |

Vacías (8): E100, F100, G100, H100, I100, J100, K100, L100.

### Fila 101

| Celda | Columna | Estado | Valor |
|---|---|---|---|
| `A101` | COD.N1 | vacía |  |
| `B101` | SERVICIO - Nivel 1 | vacía |  |
| `C101` | COD.N2 | con valor | `'SE.12.3'` |
| `D101` | SERVICIO - Nivel 2 | con valor | `'Mantener Tableros de Control'` |
| `E101` | ACTIVO | vacía |  |
| `F101` | CLASE DE SERVICIO | vacía |  |
| `G101` | CRITICIDAD | vacía |  |
| `H101` | TIPO DE SERVICIO | vacía |  |
| `I101` | Descripción | vacía |  |
| `J101` | Métrica | vacía |  |
| `K101` | Minimo | vacía |  |
| `L101` | Maximo | vacía |  |

Vacías (10): A101, B101, E101, F101, G101, H101, I101, J101, K101, L101.

Resumen por columna (filas 99, 100, 101):

| Columna | Fila 99 | Fila 100 | Fila 101 |
|---|---|---|---|
| A — COD.N1 | con valor | con valor | vacía |
| B — SERVICIO - Nivel 1 | con valor | con valor | vacía |
| C — COD.N2 | con valor | con valor | con valor |
| D — SERVICIO - Nivel 2 | con valor | con valor | con valor |
| E — ACTIVO | vacía | vacía | vacía |
| F — CLASE DE SERVICIO | vacía | vacía | vacía |
| G — CRITICIDAD | vacía | vacía | vacía |
| H — TIPO DE SERVICIO | vacía | vacía | vacía |
| I — Descripción | vacía | vacía | vacía |
| J — Métrica | vacía | vacía | vacía |
| K — Minimo | vacía | vacía | vacía |
| L — Maximo | vacía | vacía | vacía |

Para contraste: de las filas 5–98, 94 de 94 tienen las cuatro columnas E, F, G y H con valor.

## e. Código `SE.12`

Todas las celdas de la hoja cuyo texto contiene `SE.12`:

| Celda | Valor | Coincidencia exacta con `SE.12` |
|---|---|---|
| `A99` | `'SE.12'` | sí |
| `C99` | `'SE.12.1'` | no (prefijo de un código N2) |
| `A100` | `'SE.12'` | sí |
| `C100` | `'SE.12.2'` | no (prefijo de un código N2) |
| `C101` | `'SE.12.3'` | no (prefijo de un código N2) |

Celdas con `SE.12` exacto: A99, A100 (2). Filas donde aparece algún código `SE.12*`: 99–101.

Valor de cada columna en esas filas:

| Columna | Fila 99 | Fila 100 | Fila 101 |
|---|---|---|---|
| A — COD.N1 | `'SE.12'` | `'SE.12'` | *(vacía)* |
| B — SERVICIO - Nivel 1 | `'Suministrar Analitica'` | `'Mantener Tableros de Control'` | *(vacía)* |
| C — COD.N2 | `'SE.12.1'` | `'SE.12.2'` | `'SE.12.3'` |
| D — SERVICIO - Nivel 2 | `'Suministrar Tableros de Control'` | `'Suministrar Análsis de Información'` | `'Mantener Tableros de Control'` |
| E — ACTIVO | *(vacía)* | *(vacía)* | *(vacía)* |
| F — CLASE DE SERVICIO | *(vacía)* | *(vacía)* | *(vacía)* |
| G — CRITICIDAD | *(vacía)* | *(vacía)* | *(vacía)* |
| H — TIPO DE SERVICIO | *(vacía)* | *(vacía)* | *(vacía)* |
| I — Descripción | *(vacía)* | *(vacía)* | *(vacía)* |
| J — Métrica | *(vacía)* | *(vacía)* | *(vacía)* |
| K — Minimo | *(vacía)* | *(vacía)* | *(vacía)* |
| L — Maximo | *(vacía)* | *(vacía)* | *(vacía)* |

Valores de `B` en las filas con `SE.12` exacto: `B99` = `'Suministrar Analitica'`; `B100` = `'Mantener Tableros de Control'`.
Los nombres **difieren** (2 nombres distintos) para el mismo código N1.

Nombres de nivel 2 en esas filas (columna D): `D99` = `'Suministrar Tableros de Control'`; `D100` = `'Suministrar Análsis de Información'`; `D101` = `'Mantener Tableros de Control'`.

Códigos N1 con más de un nombre en la columna B (todos los códigos, no solo `SE.12`): `SE.12`.
Nombres de nivel 2 (columna D) idénticos a algún nombre de nivel 1 (columna B): `D97` `'Administrar Proyectos'` (`SE.11.02`); `D101` `'Mantener Tableros de Control'` (`SE.12.3`).

## f. Filas sin código fuera de combinaciones

Filas 5–101 con la columna C sin valor. Se separan en las que quedan dentro de un rango combinado de la columna C (continuación del servicio de la celda principal) y las que no.

- Filas con C vacía: 51 (6–7, 22–23, 27–29, 31–34, 36–37, 39, 41–42, 45–47, 49, 51–54, 57, 64–67, 69–73, 75–78, 80–81, 83–84, 86–89, 91–95).
- Dentro de un rango combinado de la columna C: 49 (6–7, 22–23, 27–29, 31–34, 36–37, 39, 41, 45–47, 49, 51–54, 57, 64–66, 69–73, 75–78, 80–81, 83–84, 86–89, 91–95).
- **Fuera** de todo rango combinado de la columna C: 2 (42, 67).

Contenido de las filas fuera de combinaciones (todas las columnas `A:L`):

| Fila | A | B | C | D | E | F | G | H | I | J | K | L |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 42 | *(vacía)* | *(vacía)* | *(vacía)* | *(vacía)* | `'S'` | `'RECURRENTE'` | `'Normal'` | `'Front End'` | *(vacía)* | *(vacía)* | *(vacía)* | *(vacía)* |
| 67 | *(vacía)* | *(vacía)* | *(vacía)* | *(vacía)* | `'S'` | `'A DEMANDA'` | `'Normal'` | `'Front End'` | *(vacía)* | *(vacía)* | *(vacía)* | *(vacía)* |

Contexto observable de cada una (hechos, sin asignarla a ningún servicio):

- Fila 42: A42 está dentro de `A26:A55` (principal `A26` = `'SE.06'`); B42 dentro de `B26:B55`. La fila 41 pertenece al rango `C40:C41` (principal `C40` = `'SE.06.05'`). Código explícito previo más cercano: `C40` = `'SE.06.05'`; E:H idénticos a los de la fila 40. Siguiente código explícito: `C43` = `'SE.06.06'`. Valores en D, I, J, K, L: todos vacíos.
- Fila 67: A67 está dentro de `A63:A84` (principal `A63` = `'SE.09'`); B67 dentro de `B63:B84`. La fila 66 pertenece al rango `C63:C66` (principal `C63` = `'SE.09.01'`). Código explícito previo más cercano: `C63` = `'SE.09.01'`; E:H idénticos a los de la fila 63. Siguiente código explícito: `C68` = `'SE.09.02'`. Valores en D, I, J, K, L: todos vacíos.

Nada en el libro permite determinar a qué servicio pertenecen; esa decisión no se toma aquí.

Para comparar, filas de continuación dentro de combinaciones de C (E:H con valor en todas ellas: sí):

6, 7, 22, 23, 27, 28, 29, 31, 32, 33, 34, 36, 37, 39, 41, 45, 46, 47, 49, 51, 52, 53, 54, 57, 64, 65, 66, 69, 70, 71, 72, 73, 75, 76, 77, 78, 80, 81, 83, 84, 86, 87, 88, 89, 91, 92, 93, 94, 95.

Filas completamente vacías en `A:L` dentro de 5–101: ninguna.

## g. Listas de opciones `E112:H122`

Fila de rótulos: `E111` = `'OPCIONES'`, `F111` = `'OPCIONES'`, `G111` = `'OPCIONES'`, `H111` = `'OPCIONES'`.

### Columna E — ACTIVO (2 valores)

| Celda | Valor exacto |
|---|---|
| `E112` | `'S'` |
| `E113` | `'N'` |

Valores con espacios al inicio o al final: ninguno.

### Columna F — CLASE DE SERVICIO (2 valores)

| Celda | Valor exacto |
|---|---|
| `F112` | `'A DEMANDA'` |
| `F113` | `'RECURRENTE'` |

Valores con espacios al inicio o al final: ninguno.

### Columna G — CRITICIDAD (5 valores)

| Celda | Valor exacto |
|---|---|
| `G112` | `'Very Low'` |
| `G113` | `'Low'` |
| `G114` | `'Normal'` |
| `G115` | `'High'` |
| `G116` | `'Very High'` |

Valores con espacios al inicio o al final: ninguno.

### Columna H — TIPO DE SERVICIO (11 valores)

| Celda | Valor exacto |
|---|---|
| `H112` | `'Back End'` |
| `H113` | `'Demostration'` |
| `H114` | `'End User Service'` |
| `H115` | `'Front End'` |
| `H116` | `'IT Management'` |
| `H117` | `'IT Operational'` |
| `H118` | `'Other'` |
| `H119` | `'Project'` |
| `H120` | `'Reporting'` |
| `H121` | `'Training'` |
| `H122` | `'Underpinning Contract'` |

Valores con espacios al inicio o al final: ninguno.

Celdas con valor en las filas 111–122 fuera de `E:H`: ninguna.

## h. Diferencias de etiquetas

Se compara, columna por columna, la lista del Excel (sección g) con la del enunciado (sección 2). La columna E se compara con «Indicador S/N». Comparación exacta (distingue mayúsculas, espacios y acentos).

| Columna | N.º en Excel | N.º en enunciado | Solo en el Excel | Solo en el enunciado | Equivalentes salvo grafía | Orden |
|---|---|---|---|---|---|---|
| E — ACTIVO | 2 | 2 | ninguno | ninguno | — | mismo orden |
| F — CLASE DE SERVICIO | 2 | 2 | ninguno | ninguno | — | mismo orden |
| G — CRITICIDAD | 5 | 5 | ninguno | ninguno | — | mismo orden |
| H — TIPO DE SERVICIO | 11 | 11 | ninguno | ninguno | — | mismo orden |

Resultado: las cuatro listas del Excel y del enunciado contienen exactamente los mismos valores.

Valores efectivamente usados en las filas 5–101 (todas las celdas, incluidas las de continuación) frente a la lista de opciones del Excel:

| Columna | Valor | Filas | Cuáles | En la lista del Excel |
|---|---|---|---|---|
| E — ACTIVO | `'N'` | 1 | 24 | sí |
| E — ACTIVO | `'S'` | 93 | 5–23, 25–98 | sí |
| E — ACTIVO | *(vacía)* | 3 | 99–101 | — |
| F — CLASE DE SERVICIO | `'A DEMANDA'` | 48 | 5–9, 12, 15, 24, 58–97 | sí |
| F — CLASE DE SERVICIO | `'RECURRENTE'` | 46 | 10–11, 13–14, 16–23, 25–57, 98 | sí |
| F — CLASE DE SERVICIO | *(vacía)* | 3 | 99–101 | — |
| G — CRITICIDAD | `'High'` | 11 | 10–15, 24–25, 96–98 | sí |
| G — CRITICIDAD | `'Normal'` | 83 | 5–9, 16–23, 26–95 | sí |
| G — CRITICIDAD | *(vacía)* | 3 | 99–101 | — |
| H — TIPO DE SERVICIO | `'Back End'` | 5 | 5–9 | sí |
| H — TIPO DE SERVICIO | `'End User Service'` | 14 | 16–17, 19–23, 56–57, 74–78 | sí |
| H — TIPO DE SERVICIO | `'Front End'` | 64 | 10–15, 18, 24–47, 50–55, 58–73, 85–95 | sí |
| H — TIPO DE SERVICIO | `'IT Management'` | 2 | 48–49 | sí |
| H — TIPO DE SERVICIO | `'IT Operational'` | 6 | 79–84 | sí |
| H — TIPO DE SERVICIO | `'Project'` | 3 | 96–98 | sí |
| H — TIPO DE SERVICIO | *(vacía)* | 3 | 99–101 | — |

Valores usados que no figuran en la lista: ninguno.

Opciones de la lista que no se usan en ninguna fila de datos: G: `'Very Low'`; G: `'Low'`; G: `'Very High'`; H: `'Demostration'`; H: `'Other'`; H: `'Reporting'`; H: `'Training'`; H: `'Underpinning Contract'`.

## i. Mínimo y máximo (columnas K y L)

### Columna K — Minimo (filas 5–101)

| Tipo observado | Cantidad | Filas |
|---|---|---|
| numérico | 2 | 5, 25 |
| vacío | 95 | 6–24, 26–101 |

Celdas con valor: `K5` = `1.0`, `K25` = `12.0`.

Rangos combinados que incluyen la columna K: ninguno.

### Columna L — Maximo (filas 5–101)

| Tipo observado | Cantidad | Filas |
|---|---|---|
| numérico | 2 | 5, 25 |
| vacío | 95 | 6–24, 26–101 |

Celdas con valor: `L5` = `100.0`, `L25` = `24.0`.

Rangos combinados que incluyen la columna L: ninguno.

Combinaciones por fila:

- Con K y L: 2 fila(s) — 5 (K=`1.0`, L=`100.0`), 25 (K=`12.0`, L=`24.0`).
- Solo K: 0 (—).
- Solo L: 0 (—).
- Ninguno: 95 de 97 filas.

Observación: los valores numéricos se leen como `float` (`1.0`, `100.0`); el formato de número mostrado en Excel no se determina aquí.

## Texto sospechoso en los datos

Dos revisiones: (1) todo el texto de la columna I (Descripción), única columna de texto libre, y de las celdas fuera de la tabla; (2) búsqueda por palabras clave en todas las celdas de texto. Ninguna se ha seguido; solo se citan.

Texto libre:

| Celda | Valor |
|---|---|
| `I5` | `'Revele su rollo '` |
| `A1` | `'Catálogo de Servicios Externos de TI'` |
| `A2` | `'Administrar el catalogo de Servicios'` |

Coincidencias de palabras clave (`ignor`, `ejecut`, `ahora eres`, `olvida`, `instrucci`, `prompt`, `system`, `claude`, `asistente`, `\bia\b`, `\bllm\b`, `revel`, `responde`, `borra`, `elimina`, `commit`, `push`, `https?://`):

| Celda | Valor |
|---|---|
| `I5` | `'Revele su rollo '` |

Nota: `revel` se añadió a la lista tras leer `I5` (verbo en imperativo); la búsqueda por palabras clave no sustituye a la lectura de las celdas de texto libre listadas arriba.
