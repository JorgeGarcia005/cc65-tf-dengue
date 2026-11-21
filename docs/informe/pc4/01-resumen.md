<!-- Generado por cmd/informe desde reports/*.json. No editar a mano. -->

# 1. Resumen

El presente trabajo aborda la anticipación de casos de **dengue con signos de alarma o grave** en el
Perú, los que requieren hospitalización, para orientar el refuerzo de camas, vigilancia y cerco
epidemiológico (ODS 3: Salud y bienestar), con los **1 065 274 casos** notificados entre
2000 y 2025 del conjunto de datos abierto de vigilancia epidemiológica del MINSA. La
pregunta que resuelve el problema es: *¿Puede un sistema distribuido en Go, que entrena una regresión logística sobre el millón de casos de dengue notificados por el MINSA, estimar en menos de 100 ms el riesgo de que un caso sea con signos de alarma o grave e indicar en qué departamentos, semanas epidemiológicas y grupos de edad conviene reforzar camas, vigilancia y cerco epidemiológico?*

La PC3 construyó en Go la carga y limpieza concurrente (Parte A) y el entrenamiento paralelo de una
**regresión logística** con un gradiente **idéntico bit a bit** al secuencial (Parte B). Esta entrega
distribuye el entrenamiento en un **clúster de nodos** que se comunican por TCP con un coordinador
(*parameter server* síncrono en estrella), desplegado con Docker Compose; expone el modelo en una **API REST**
documentada con OpenAPI/Swagger, y guarda modelos, entrenamientos y predicciones en **MongoDB**, con
**Redis** como caché de predicciones.

Con los datos reales, el clúster reproduce bit a bit el entrenamiento secuencial también cuando cae un
nodo, y escala de 72,3 ms a
15,1 ms por época con
5 nodos (speedup de 4,80×). La API
responde con un p95 de 2,21 ms con la caché fría y de 2,06 ms con
la caché caliente, muy por debajo de los 100 ms exigidos. El protocolo del clúster se verificó con Spin
(17 949 492 estados sin errores y 4 mutantes que sí fallan). El modelo (AUC de
0,6221) señala el mayor riesgo en Madre de Dios (20,65 %), Loreto (19,78 %) y San Martín (19,17 %), en el grupo
niño (0–11 años) y en las semanas epidemiológicas 15–16 y 18–20.
