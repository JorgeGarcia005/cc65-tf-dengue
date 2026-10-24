<!-- Generado por cmd/informe desde reports/perfil_bronze.json. No editar a mano. -->

# 2. Datos: perfil del dataset original (capa bronze)

## 2.1 Fuente

- **Dataset:** Vigilancia epidemiológica de dengue, MINSA / CDC Perú, Plataforma Nacional de
  Datos Abiertos (<https://www.datosabiertos.gob.pe/dataset/vigilancia-epidemiol%C3%B3gica-de-dengue>).
- **Archivo:** `vigilancia_dengue_2000_2025.csv` · 117,9 MB · sha256 `7865e8b2f5a7aef1e8dce159444891f2c2e59b311d574f28c55aaca5a9abb8ba`.
- **Unidad de análisis:** un caso de dengue notificado a la Red Nacional de Epidemiología.
- **Filas:** 1 065 274 (85 mal formadas: 85 por comillas y
  0 por número de columnas).

## 2.2 Variables

| Columna | Vacíos | Valores distintos | Valores más frecuentes |
|---|---:|---:|---|
| `departamento` | 0 | 23 | PIURA (239078); LIMA (124514); LORETO (113103) |
| `provincia` | 0 | 124 | PIURA (113982); LIMA (107710); MAYNAS (73793) |
| `distrito` | 0 | 625 | PIURA (40003); IQUITOS (31023); SULLANA (29913) |
| `localidad` | 53 909 | 10 378 | NULL (69726); (vacío) (53909); IQUITOS (21475) |
| `enfermedad` | 0 | 3 | DENGUE SIN SIGNOS DE ALARMA (944608); DENGUE CON SIGNOS DE ALARMA (116384); DENGUE GRAVE (4197) |
| `ano` | 0 | 26 | 2024 (271531); 2023 (256600); 2017 (68290) |
| `semana` | 0 | 53 | 16 (51401); 15 (48317); 18 (47713) |
| `diagnostic` | 0 | 3 | A97.0 (944608); A97.1 (116384); A97.2 (4197) |
| `tipo_dx` | 0 | 2 | C (955057); P (110132) |
| `diresa` | 23 | 33 | 32 (122429); 16 (113102); 31 (104690) |
| `ubigeo` | 0 | 662 | 200101 (40003); 160101 (31023); 200601 (29913) |
| `localcod` | 149 626 | 10 210 | (vacío) (149626); 0 (25605); 1601010001 (20079) |
| `edad` | 0 | 123 | 15 (25523); 14 (25322); 16 (24876) |
| `tipo_edad` | 0 | 3 | A (1058214); M (6644); D (331) |
| `sexo` | 0 | 2 | F (572164); M (493025) |

## 2.3 Variable objetivo

El objetivo es **1** si el caso es *dengue con signos de alarma* (CIE-10 A97.1) o *dengue grave*
(A97.2) y **0** si es *dengue sin signos de alarma* (A97.0). Según la guía clínica, los casos
con signos de alarma requieren hospitalización, por lo que el objetivo funciona como indicador
de **necesidad de hospitalización** (validado con el profesor, ADR-0003).

- Sin signos (0): **944 608** · Con signos o grave (1): **120 581** · Proporción de positivos:
  **11,32 %**. Hay desbalance de clases, que se trata con pesos por clase.

| Año | Sin signos (A97.0) | Con signos (A97.1) | Grave (A97.2) | % positivos |
|---|---:|---:|---:|---:|
| 2000 | 5 557 | 0 | 0 | 0,0 |
| 2001 | 23 276 | 0 | 250 | 1,1 |
| 2002 | 8 071 | 1 | 14 | 0,2 |
| 2003 | 3 334 | 0 | 15 | 0,4 |
| 2004 | 9 512 | 0 | 34 | 0,4 |
| 2005 | 5 625 | 0 | 15 | 0,3 |
| 2006 | 4 018 | 0 | 4 | 0,1 |
| 2007 | 6 309 | 0 | 35 | 0,6 |
| 2008 | 12 789 | 0 | 35 | 0,3 |
| 2009 | 13 396 | 0 | 11 | 0,1 |
| 2010 | 16 669 | 134 | 39 | 1,0 |
| 2011 | 24 400 | 3 508 | 176 | 13,1 |
| 2012 | 23 444 | 4 860 | 199 | 17,7 |
| 2013 | 9 540 | 3 477 | 72 | 27,1 |
| 2014 | 12 939 | 4 203 | 92 | 24,9 |
| 2015 | 29 385 | 6 308 | 123 | 18,0 |
| 2016 | 21 656 | 3 381 | 123 | 13,9 |
| 2017 | 60 389 | 7 667 | 234 | 11,6 |
| 2018 | 3 523 | 1 111 | 64 | 25,0 |
| 2019 | 12 174 | 2 955 | 158 | 20,4 |
| 2020 | 41 821 | 5 883 | 225 | 12,7 |
| 2021 | 40 874 | 3 767 | 129 | 8,7 |
| 2022 | 55 285 | 7 707 | 212 | 12,5 |
| 2023 | 226 454 | 29 129 | 1 017 | 11,7 |
| 2024 | 244 743 | 26 052 | 736 | 9,9 |
| 2025 | 29 425 | 6 241 | 185 | 17,9 |

**Hallazgo:** la categoría *con signos de alarma* aparece de forma marginal en 2010 y se usa
de forma sistemática desde 2011, tras la adopción de la clasificación de la OMS de 2009. Antes
los casos se registran casi todos como *sin signos*, así que esos años no sirven para aprender
el objetivo: **el modelo se entrena con los años desde 2011**, mientras que la carga y la
limpieza procesan todas las filas.

## 2.4 Edad por etapa de vida (MINSA)

| Etapa | Casos |
|---|---:|
| nino_0_11 | 176 747 |
| adolescente_12_17 | 146 780 |
| joven_18_29 | 247 590 |
| adulto_30_59 | 400 993 |
| adulto_mayor_60_mas | 93 079 |

## 2.5 Procesamiento concurrente del perfil

El archivo se lee en memoria y se divide en **113 bloques** de ~1 MiB alineados a fin
de línea. Un *worker pool* procesa los bloques y cada uno deja su resultado en un slot propio;
los resultados se unen en orden de bloque. El perfil con 2 workers tomó
0,609 s frente a 1,112 s con uno (speedup 1,83×), y es
**idéntico** al secuencial: verificado por `cmd/perfil`.
