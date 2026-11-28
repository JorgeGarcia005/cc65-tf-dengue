# 2. Descripción del problema y motivación

## 2.1 Contexto: el dengue en el Perú y la carga hospitalaria

El dengue es una enfermedad viral transmitida por el mosquito *Aedes aegypti*. Es endémica en la Amazonía y en la costa norte del Perú, y es una de las enfermedades transmitidas por vectores con mayor carga en el mundo (Bhatt et al., 2013). La mayoría de los casos se recupera sin complicaciones, pero una parte evoluciona a **dengue con signos de alarma** (dolor abdominal intenso, vómitos persistentes, sangrado de mucosas, letargo, entre otros) o a **dengue grave**. La guía de la Organización Mundial de la Salud (World Health Organization [WHO], 2009), que adopta el Ministerio de Salud del Perú, indica que estos casos **requieren hospitalización** y vigilancia estrecha, porque pueden progresar a choque y muerte.

El conjunto de datos de vigilancia del MINSA muestra la escala del problema. En 2023 se notificaron <!--v:miles (total (index .S.P.Perfil.PorAno "2023"))-->256 600<!--/v--> casos, <!--v:dos (razon (total (index .S.P.Perfil.PorAno "2023")) (total (index .S.P.Perfil.PorAno "2022")))-->4,06<!--/v--> veces los de 2022, y en <!--v:.S.AnoPico-->2024<!--/v-->, el año con más casos del periodo, <!--v:miles .S.CasosPico-->271 531<!--/v--> (@fig:casos-ano, sección 6). Solo en 2023, <!--v:miles (index (index .S.P.Perfil.PorAno "2023") 1)-->29 129<!--/v--> casos fueron con signos de alarma y <!--v:miles (index (index .S.P.Perfil.PorAno "2023") 2)-->1 017<!--/v--> graves. Cuando los casos que necesitan una cama se concentran en pocas semanas y regiones, la capacidad local no alcanza, y la respuesta (camas, personal, cerco epidemiológico) llega tarde si se organiza recién cuando los casos ya están en el establecimiento.

Saber **con anticipación dónde, cuándo y en qué grupos de edad** se concentrarán los casos que necesitarán hospitalización permite redistribuir esos recursos antes del pico. Ese es el problema que aborda este trabajo.

## 2.2 Problema y usuario objetivo

El usuario del sistema es el **equipo de epidemiología de una Dirección Regional de Salud (DIRESA)** o del Centro Nacional de Epidemiología (CDC Perú), que planifica la respuesta ante el dengue: cuántas camas reservar, dónde intensificar la vigilancia y el control vectorial y a qué grupos priorizar. Hoy esa planificación se hace con los conteos de casos de las semanas anteriores. El problema es que el conteo no distingue cuántos de esos casos serán graves: dos departamentos con el mismo número de casos pueden necesitar un número muy distinto de camas.

Tabla: Problema, usuario y lo que el sistema le entrega {#tab:usuario}

| Aspecto | Descripción |
|---|---|
| Usuario | Epidemiólogo o planificador de una DIRESA o del CDC Perú; administrador técnico del sistema |
| Decisión que toma | Dónde y cuándo reforzar camas, vigilancia y cerco epidemiológico, y a qué grupos de edad priorizar |
| Información que le falta | Qué fracción de los casos esperados será con signos de alarma o grave, por departamento, semana epidemiológica y grupo de edad |
| Lo que el sistema le entrega | El riesgo calibrado por combinación de lugar, semana y edad, los casos graves esperados, un mapa de riesgo y la predicción para un caso concreto en menos de 100 ms |
| Dónde lo ve | La aplicación web Vigía (sección 16): mapa, predicción, evaluación y panel de administración |

Nota: Elaboración propia a partir de la pregunta de impacto social (sección 2.3).

## 2.3 Pregunta de impacto social (ODS 3) y cómo la contesta el sistema

> ¿En qué departamentos, semanas epidemiológicas y grupos de edad conviene reforzar camas, vigilancia y cerco epidemiológico por dengue con signos de alarma o grave?

La pregunta se alinea con el **Objetivo de Desarrollo Sostenible 3 (Salud y bienestar)** de la Agenda 2030 (Organización de las Naciones Unidas, 2015), en particular con la meta 3.3 (poner fin a las epidemias de enfermedades tropicales desatendidas, entre las que la OMS incluye al dengue) y con la meta 3.d (reforzar la capacidad de alerta temprana y de gestión de riesgos para la salud). Se benefician los pacientes que llegan a tiempo a una cama, los equipos de las DIRESA que planifican la respuesta y la población de las zonas donde se intensifica el cerco epidemiológico.

La pregunta de impacto social dice **qué** se quiere saber. La **pregunta que resuelve el problema** dice además **cómo** lo responde este trabajo y con qué condición de éxito medible:

> ¿Puede un sistema distribuido en Go, que entrena una regresión logística sobre el millón de casos de dengue notificados por el MINSA, estimar en menos de 100 ms el riesgo de que un caso sea con signos de alarma o grave e indicar en qué departamentos, semanas epidemiológicas y grupos de edad conviene reforzar camas, vigilancia y cerco epidemiológico?

Cada parte de la pregunta tiene una respuesta verificable en este informe (@tab:pregunta).

Tabla: Partes de la pregunta que resuelve el problema y dónde se responde cada una {#tab:pregunta}

| Parte de la pregunta | Cómo se responde | Evidencia | Sección |
|---|---|---|---|
| «sistema distribuido en Go» | Coordinador y nodos que se comunican por TCP, en contenedores Docker | Pesos idénticos bit a bit al secuencial, también si cae un nodo | 9, 11 |
| «sobre el millón de casos notificados» | Carga y limpieza concurrente del archivo completo del MINSA | `reports/limpieza.json`; silver igual en secuencial y concurrente | 6, 7 |
| «estimar el riesgo de que un caso sea con signos de alarma o grave» | Regresión logística con probabilidad calibrada | AUC, calibración y Brier sobre el conjunto de prueba | 8, 17 |
| «en menos de 100 ms» | API REST medida con carga concurrente, JWT, caché fría y caliente | Percentil 95 de `cmd/benchapi` y colección Postman | 12, 13 |
| «en qué departamentos, semanas y grupos de edad» | Riesgo y casos graves esperados por departamento, semana y etapa de vida | `reports/evaluacion.json`; mapa de la SPA | 16, 18 |

Nota: Las cifras de cada evidencia se generan desde los reportes JSON del repositorio (sección 4.4).

## 2.4 Conjunto de datos

Se usa el conjunto de datos abierto *Vigilancia epidemiológica de dengue* del MINSA y del Centro Nacional de Epidemiología, Prevención y Control de Enfermedades (CDC Perú), publicado en la Plataforma Nacional de Datos Abiertos (Ministerio de Salud del Perú, s.f.). Es el caso 1 del enunciado, «Predicción de riesgo epidemiológico nacional», asignado al equipo y validado con el profesor (ADR-0003). Cada fila es un caso notificado, con su ubicación (departamento, provincia, distrito, localidad y ubigeo), el año y la semana epidemiológica, la edad y el sexo del paciente, el tipo de diagnóstico y la clasificación clínica.

Tabla: Características del conjunto de datos {#tab:dataset}

| Característica | Valor |
|---|---|
| Archivo | `<!--v:.S.P.Archivo-->vigilancia_dengue_2000_2025.csv<!--/v-->` (<!--v:mb .S.P.Bytes-->117,9<!--/v--> MB) |
| Filas (casos notificados) | <!--v:miles .S.P.Perfil.Filas-->1 065 274<!--/v--> |
| Periodo | <!--v:.S.AnoMin-->2000<!--/v-->–<!--v:.S.AnoMax-->2025<!--/v--> |
| Filas válidas tras la limpieza (silver) | <!--v:miles .S.L.L.Salidas-->1 065 260<!--/v--> |
| Filas para el modelo (desde <!--v:.S.M.S.AnoMin-->2011<!--/v-->) | <!--v:miles .S.M.S.Train.Filas-->765 297<!--/v--> de entrenamiento y <!--v:miles .S.M.S.Test.Filas-->190 819<!--/v--> de prueba |
| Proporción de casos con signos de alarma o grave (entrenamiento) | <!--v:pct .S.M.S.Train.Tasa-->12,55<!--/v--> % |
| Variables del modelo | <!--v:.S.M.S.D-->31<!--/v--> |

Nota: Fuente: `reports/perfil_bronze.json`, `reports/limpieza.json` y `reports/split.json`. El SHA-256 del archivo queda en `data/MANIFEST.sha256`.

El dataset no registra la hospitalización ni las comorbilidades del paciente, por lo que el problema se plantea así (ADR-0003):

- **Objetivo binario.** Vale 1 si el caso es *con signos de alarma* (CIE-10 A97.1) o *grave* (A97.2), y 0 si es *sin signos* (A97.0). Clínicamente, el 1 indica necesidad de hospitalización (WHO, 2009).
- **Variables explicativas.** Sin datos clínicos en el archivo abierto, se usan las que sí trae: departamento, semana epidemiológica, edad y sexo (sección 8.1).
- **Periodo del modelo.** La categoría *con signos de alarma* se usa de forma sistemática desde 2011, cuando se adoptó la clasificación de la OMS de 2009; antes de ese año casi no aparece (@fig:casos-ano). Por eso el modelo se entrena con los casos desde ese año y el perfil del dataset (sección 6) usa todos.
- **Desbalance de clases.** Solo el <!--v:pct .S.M.S.Train.Tasa-->12,55<!--/v--> % de las filas de entrenamiento es positivo, así que la pérdida se pondera por clase (King y Zeng, 2001) y la probabilidad se calibra (sección 8.2).

## 2.5 Por qué una solución concurrente y distribuida

- **Volumen.** Más de un millón de filas que deben leerse, validarse y transformarse cada vez que el MINSA publica una versión nueva del archivo. La limpieza secuencial lee y valida fila por fila; repartida en bloques entre workers, el tiempo baja con los núcleos disponibles (sección 7.6).
- **Entrenamiento iterativo.** El descenso de gradiente recorre todo el conjunto de entrenamiento en cada época, durante cientos de épocas (<!--v:.S.M.E.Epocas-->600<!--/v--> en el entrenamiento de referencia). El costo por época crece en proporción al número de filas y se puede repartir entre núcleos y entre máquinas.
- **Estructura aditiva.** El gradiente de la regresión logística es una **suma sobre filas**: se puede calcular por partes (bloques, workers, nodos) y sumar después. Es el caso típico del paralelismo de datos con *parameter server* (Li et al., 2014).
- **Servicio en línea.** Un mismo modelo atiende muchas consultas a la vez desde la SPA: la API debe responder en menos de 100 ms bajo carga concurrente, lo que exige un servidor que atienda cada petición en su propia goroutine sin bloquear a las demás (sección 12.1).

## 2.6 Trabajos relacionados

Se revisaron dos grupos de trabajos: los que estudian cómo **distribuir el entrenamiento** de un modelo (de ellos sale la arquitectura) y los que estudian el **riesgo de dengue grave** (de ellos sale la formulación del problema). Para cada uno se indica qué propone, qué toma este trabajo y qué no.

**MapReduce (Dean y Ghemawat, 2008).** Propone expresar un cómputo sobre datos masivos como una función *map* que se aplica a cada partición y una *reduce* que combina los resultados; el sistema reparte las particiones y reintenta las tareas que fallan. Un paso de descenso de gradiente encaja en ese molde: el *map* calcula el gradiente de una partición y el *reduce* los suma. Este trabajo toma esa idea (cada nodo calcula sobre su shard y el coordinador suma) y la reasignación de trabajo cuando un nodo falla (sección 9.4). No toma el motor genérico: relanzar un trabajo MapReduce por época obligaría a releer los datos en cada una.

**Spark (Zaharia et al., 2016).** Resuelve ese problema manteniendo los datos en memoria entre iteraciones. Este trabajo toma la misma decisión: cada nodo carga su shard una sola vez, al registrarse, y lo reutiliza en todas las épocas; solo viajan los pesos y los gradientes (sección 9.2). No usa Spark: el enunciado exige Go y biblioteca estándar, y un coordinador propio permite fijar el **orden** de la suma, del que depende el resultado exacto (sección 8.4).

**Parameter server (Li et al., 2014).** Los nodos calculan gradientes sobre su partición y un servidor central mantiene los pesos; el diseño admite actualizaciones **asíncronas** para que un nodo lento no detenga a los demás. Este trabajo adopta la topología en estrella con servidor central, pero en su variante **síncrona**: el coordinador espera los gradientes de todos los shards antes de actualizar. Con la asíncrona, el modelo final dependería del orden de llegada de los mensajes y no se podría comparar con el secuencial (sección 9.1).

**Ring all-reduce (Sergeev y Del Balso, 2018).** Los nodos suman los gradientes entre sí en un anillo, sin servidor central, lo que reparte el ancho de banda cuando los gradientes son grandes (redes neuronales con millones de pesos). Se descarta por dos razones medidas: el gradiente de este modelo tiene <!--v:.S.M.S.D-->31<!--/v--> variables más el sesgo, así que el mensaje es pequeño y la red ocupa una fracción menor de la época (sección 11.3); y el orden de la suma dependería de la posición en el anillo.

**Aritmética de punto flotante (Goldberg, 1991).** La suma de punto flotante no es asociativa: sumar los mismos números en otro orden puede dar un resultado distinto en los últimos bits. De aquí sale la regla de determinismo del proyecto: los bloques de datos tienen límites fijos y los parciales se suman siempre en orden de índice de bloque, sin importar cuántos workers o nodos los calcularon (sección 8.4).

**Riesgo de dengue grave con aprendizaje automático (Huang et al., 2020).** Comparan regresión logística, bosques aleatorios, *gradient boosting*, máquinas de soporte vectorial y redes neuronales para predecir dengue grave en 798 pacientes de un hospital de Taiwán, con datos demográficos y pruebas de laboratorio del primer día. Muestran que la edad y el sexo aportan información y que, con datos clínicos, los modelos sirven para el triaje. Este trabajo toma la formulación (un objetivo binario de severidad) y la regresión logística como modelo base interpretable. No puede tomar las variables de laboratorio, que el dataset abierto no trae; por eso su objetivo es **priorizar zonas, semanas y grupos**, no diagnosticar a un paciente, y su capacidad de discriminación es menor (sección 18.6).

**Carga global del dengue (Bhatt et al., 2013)** estima dónde ocurre la enfermedad en el mundo a partir de la ocurrencia y de covariables ambientales. Este trabajo responde una pregunta operativa distinta: qué casos ya notificados necesitarán hospitalización, y dónde y cuándo se concentran.

**Eventos raros en regresión logística (King y Zeng, 2001).** Cuando una clase es poco frecuente, la regresión logística subestima su probabilidad. Proponen ponderar las clases y corregir la probabilidad; este trabajo pondera la pérdida por clase y calibra la probabilidad final (sección 8.2).

**Verificación de modelos con Spin (Holzmann, 2003).** Spin explora todos los intercalados posibles de un sistema concurrente descrito en Promela y busca violaciones de aserciones y de propiedades de lógica temporal lineal (LTL). Este trabajo modela el protocolo entre el coordinador y los nodos y lo verifica antes y después de implementarlo (sección 10).

Tabla: Resumen de los trabajos relacionados y de lo que toma este trabajo de cada uno {#tab:relacionados}

| Trabajo | Qué propone | Qué toma este trabajo | Qué no toma, y por qué |
|---|---|---|---|
| Dean y Ghemawat (2008) | *map* por partición y *reduce* que combina; reintento de tareas | Gradiente por shard y suma en el coordinador; reasignación del shard de un nodo caído | El motor genérico: relanzaría un trabajo por época |
| Zaharia et al. (2016) | Datos en memoria entre iteraciones | Cada nodo carga su shard una vez y lo reutiliza en todas las épocas | Spark: el enunciado exige Go y el orden de la suma debe ser fijo |
| Li et al. (2014) | Servidor central de parámetros, actualizaciones asíncronas | Topología en estrella con servidor central | La asincronía: el resultado dependería del orden de llegada |
| Sergeev y Del Balso (2018) | Suma en anillo, sin servidor central | — | Mensajes pequeños; el orden de la suma dependería del anillo |
| Goldberg (1991) | La suma de punto flotante no es asociativa | Bloques fijos y reducción en orden de bloque | — |
| Huang et al. (2020) | Modelos de severidad del dengue con datos clínicos | Objetivo binario de severidad; regresión logística como modelo interpretable | Variables de laboratorio: no están en el dataset abierto |
| Bhatt et al. (2013) | Carga global del dengue | Contexto del problema | Su pregunta es dónde ocurre la enfermedad, no qué casos se agravan |
| King y Zeng (2001) | Corrección de la regresión logística con clases desbalanceadas | Pérdida ponderada por clase y calibración | — |
| Holzmann (2003) | Verificación de modelos de sistemas concurrentes | Modelo Promela del protocolo, aserciones, LTL y mutantes | — |

Nota: Elaboración propia. Las secciones indicadas en el texto muestran dónde se aplica cada decisión.

Frente a los sistemas distribuidos revisados, pensados para miles de máquinas, este trabajo prioriza la **reproducibilidad**: el modelo distribuido es idéntico bit a bit al secuencial con cualquier número de nodos y ante la caída de uno, y el protocolo está verificado formalmente. Frente a los trabajos clínicos, prioriza lo que se puede hacer con datos abiertos de alcance nacional: ordenar el riesgo por lugar, semana y edad para planificar la respuesta.
