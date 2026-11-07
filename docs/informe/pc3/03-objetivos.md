# 3. Objetivos

## 3.1 Objetivo general

Estimar el riesgo de dengue con signos de alarma o grave en el Perú mediante una regresión logística
entrenada de forma concurrente y distribuida en Go.

## 3.2 Objetivos específicos

1. Analizar el conjunto de datos de vigilancia epidemiológica de dengue y las técnicas de programación
   concurrente y distribuida aplicables a su procesamiento.
2. Diseñar el modelo de regresión logística distribuida y la estrategia de paralelización del cálculo
   del gradiente.
3. Implementar en Go la carga y limpieza concurrente de los datos y el entrenamiento paralelo del
   modelo, sin condiciones de carrera.
4. Validar la equivalencia entre las versiones secuencial y concurrente, el speedup obtenido y la
   capacidad del modelo para responder la pregunta de impacto social.
