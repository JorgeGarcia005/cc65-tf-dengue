# 3. Objetivos

## 3.1 Objetivo general

Responder la pregunta que resuelve el problema: **¿Puede un sistema distribuido en Go, que entrena una regresión logística sobre el millón de casos de dengue notificados por el MINSA, estimar en menos de 100 ms el riesgo de que un caso sea con signos de alarma o grave e indicar en qué departamentos, semanas epidemiológicas y grupos de edad conviene reforzar camas, vigilancia y cerco epidemiológico?** Para ello, estimar el riesgo de dengue con signos de alarma o grave en el Perú mediante una regresión logística
entrenada de forma concurrente y distribuida en Go, y exponerla como un servicio.

## 3.2 Objetivos específicos

1. Analizar el conjunto de datos de vigilancia epidemiológica de dengue y las técnicas de programación
   concurrente y distribuida aplicables a su procesamiento.
2. Diseñar el modelo de regresión logística distribuida y la estrategia de paralelización del cálculo
   del gradiente.
3. Implementar en Go la carga y limpieza concurrente de los datos y el entrenamiento paralelo del
   modelo, sin condiciones de carrera.
4. Distribuir el entrenamiento en un clúster de nodos que se comunican por TCP con un coordinador,
   desplegado con Docker, tolerante a la caída de un nodo y con el mismo resultado que el secuencial.
5. Exponer el modelo en una API REST documentada con OpenAPI que responda en menos de 100 ms, y persistir
   modelos, entrenamientos y predicciones en MongoDB con Redis como caché.
6. Validar la equivalencia entre las versiones secuencial, concurrente y distribuida, el speedup obtenido,
   la latencia de la API y la capacidad del modelo para responder la pregunta.
