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

### Pregunta que resuelve el problema

La pregunta de impacto social dice **qué** se quiere saber. La pregunta que resuelve el problema planteado
dice además **cómo** lo responde este trabajo y con qué condición de éxito medible:

> **¿Puede un sistema distribuido en Go, que entrena una regresión logística sobre el millón de casos de dengue notificados por el MINSA, estimar en menos de 100 ms el riesgo de que un caso sea con signos de alarma o grave e indicar en qué departamentos, semanas epidemiológicas y grupos de edad conviene reforzar camas, vigilancia y cerco epidemiológico?**

Cada parte de la pregunta tiene una respuesta verificable en el informe:

| Parte de la pregunta | Cómo se responde | Dónde |
|---|---|---|
| «sistema distribuido en Go» | Coordinador y nodos por TCP en Docker; pesos idénticos bit a bit al secuencial, también si cae un nodo | Secciones 8 y 9 |
| «sobre el millón de casos notificados» | Carga y limpieza concurrente del archivo completo del MINSA | Secciones 5 y 6 |
| «estimar el riesgo de que un caso sea con signos de alarma o grave» | Regresión logística con probabilidad calibrada | Sección 7 |
| «en menos de 100 ms» | API REST medida con carga concurrente (p95) y con Postman | Sección 10 |
| «en qué departamentos, semanas y grupos de edad» | Riesgo por departamento, semana epidemiológica y etapa de vida | Secciones 7 y 13 |

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

## 2.6 Trabajos relacionados

| Enfoque | Cómo distribuye el entrenamiento | Qué toma este trabajo y qué no |
|---|---|---|
| MapReduce (Dean & Ghemawat, 2008) y Spark (Zaharia et al., 2016) | Cada iteración del descenso de gradiente es un trabajo *map* (gradiente por partición) y *reduce* (suma); Spark mantiene los datos en memoria entre iteraciones | Se toma la idea de calcular por partición y sumar. No se usa un motor genérico: para una regresión logística, un coordinador propio en Go evita relanzar tareas en cada época y permite fijar el orden de la suma |
| *Parameter server* (Li et al., 2014) | Los nodos calculan gradientes sobre su partición y un servidor central mantiene los pesos; admite actualizaciones asíncronas | Se adopta la topología en estrella en su variante **síncrona**: la asíncrona cambia el resultado según el orden de llegada de los nodos |
| *Ring all-reduce* (Sergeev & Del Balso, 2018) | Los nodos suman los gradientes entre sí en anillo, sin servidor central | Se descarta: el orden de la suma dependería del anillo y, con pocos nodos y mensajes pequeños, el servidor central no es el cuello de botella (sección 9) |

Frente a estos sistemas, pensados para escalar a miles de máquinas, este trabajo prioriza la
**reproducibilidad**: el modelo distribuido es idéntico bit a bit al secuencial con cualquier número de
nodos y ante la caída de uno, y el protocolo se verificó formalmente con Spin (Holzmann, 2003). En el
dominio, los estudios de carga global del dengue (Bhatt et al., 2013) estiman dónde ocurre la enfermedad;
este trabajo se concentra en una pregunta operativa distinta: qué casos notificados necesitarán
hospitalización y dónde y cuándo se concentran.
