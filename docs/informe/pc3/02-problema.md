# 2. Descripción del problema y motivación

## 2.1 Contexto

El dengue es una enfermedad viral transmitida por el mosquito *Aedes aegypti* y es endémica en la
Amazonía y la costa norte del Perú; es una de las enfermedades transmitidas por vectores con mayor
carga en el mundo (Bhatt et al., 2013). La mayoría de los casos se recupera sin complicaciones, pero una
fracción evoluciona a **dengue con signos de alarma** (dolor abdominal intenso, vómitos persistentes,
sangrado de mucosas, letargo, entre otros) o a **dengue grave**. Según la guía de la Organización
Mundial de la Salud (World Health Organization [WHO], 2009), adoptada por el Ministerio de Salud del Perú, estos casos **requieren
hospitalización** y vigilancia estrecha, porque pueden progresar a choque y muerte.

El país vivió en 2023 y 2024 epidemias de dengue de magnitud histórica (ver la figura de casos por año
en la sección 5.3). En esos picos, los establecimientos de salud de las regiones más afectadas se
saturan: faltan camas, personal y capacidad de vigilancia justo cuando más se necesitan. Saber **con
anticipación dónde, cuándo y en qué grupos de edad** se concentrarán los casos que necesitarán
hospitalización permite redistribuir recursos antes de que el sistema colapse.

## 2.2 Pregunta de impacto social

> **¿En qué departamentos, semanas epidemiológicas y grupos de edad conviene reforzar camas,
> vigilancia y cerco epidemiológico por dengue con signos de alarma o grave?**

La pregunta se alinea con el **Objetivo de Desarrollo Sostenible 3 (Salud y bienestar)** de la Agenda 2030 (Organización de
las Naciones Unidas, 2015), en particular con la meta 3.3 (poner fin a las epidemias de enfermedades tropicales desatendidas, entre
las que la OMS incluye al dengue) y la meta 3.d (reforzar la capacidad de alerta temprana y de gestión
de riesgos para la salud). Quienes se benefician son los pacientes que llegan a tiempo a una cama, los
equipos de salud de las DIRESA que planifican la respuesta y la población de las zonas donde se
intensifica el cerco epidemiológico.

## 2.3 Conjunto de datos

Se usa el conjunto de datos abierto Vigilancia epidemiológica de dengue del MINSA y el Centro Nacional
de Epidemiología, Prevención y Control de Enfermedades (CDC Perú), publicado en la Plataforma Nacional
de Datos Abiertos (Ministerio de Salud del Perú, s.f.). Cada fila es un caso notificado entre 2000 y
2025, con su ubicación (departamento, provincia, distrito, localidad y ubigeo), su semana
epidemiológica, la edad y el sexo del paciente, el tipo de diagnóstico y la clasificación clínica.

El dataset no registra la hospitalización ni las comorbilidades del paciente, por lo que el problema se
plantea así (ADR-0003):

| Necesidad | Cómo se resuelve con estos datos |
|---|---|
| Procesar más de un millón de registros | El archivo completo se carga y limpia de forma concurrente (sección 6) |
| Medir la severidad del caso | Objetivo binario: 1 si el caso es *con signos de alarma* (CIE-10 A97.1) o *grave* (A97.2), 0 si es *sin signos* (A97.0). Clínicamente, el objetivo indica necesidad de hospitalización |
| Variables explicativas | Sin comorbilidades en los datos abiertos, se usan ubicación, semana epidemiológica, edad y sexo |

## 2.4 Por qué es un problema de cómputo concurrente y distribuido

- **Volumen:** más de un millón de filas que deben leerse, validarse y transformarse cada vez que el
  MINSA publica una nueva versión del archivo.
- **Entrenamiento iterativo:** el descenso de gradiente recorre todo el conjunto de entrenamiento en cada
  época, cientos de veces. El costo por época es lineal en el número de filas y se puede repartir.
- **Estructura aditiva:** el gradiente de la regresión logística es una **suma sobre filas**, así que se
  puede calcular por partes (bloques, workers, nodos) y combinar después. Es el caso de libro del
  paralelismo de datos con *parameter server* (Li et al., 2014).

## 2.5 Limitaciones declaradas

- Sin comorbilidades ni datos clínicos, el poder predictivo del modelo será **moderado**: sirve para
  ordenar y priorizar zonas y grupos, no para diagnosticar a un paciente.
- La categoría *con signos de alarma* se usa de forma sistemática desde 2011 (clasificación OMS 2009),
  por lo que el modelo se entrena con los casos desde ese año (sección 5.3).
- El dataset registra casos **notificados**: refleja también la capacidad de vigilancia de cada región,
  no solo la incidencia real.
