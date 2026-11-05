# 3. Objetivos

## 3.1 Objetivo general

Desarrollar en Go un sistema concurrente y distribuido que procese los casos de dengue notificados en el
Perú y entrene una regresión logística distribuida para estimar el riesgo de dengue con signos de alarma
o grave, de modo que se pueda priorizar el refuerzo de camas, vigilancia y cerco epidemiológico por
departamento, semana epidemiológica y grupo de edad.

## 3.2 Objetivos específicos

1. Cargar, limpiar y validar de forma concurrente el archivo completo de vigilancia de dengue, con un
   resultado idéntico al de una ejecución secuencial y un reporte de calidad por regla (Parte A).
2. Diseñar el modelo de regresión logística y la forma de distribuir su entrenamiento, garantizando que el
   gradiente calculado en paralelo sea idéntico bit a bit al secuencial (Parte B).
3. Paralelizar el cálculo del gradiente dentro de un nodo con goroutines, canales y acumuladores
   privados, sin condiciones de carrera, y medir su speedup con un protocolo reproducible.
4. Diseñar y verificar formalmente el protocolo entre el coordinador y los nodos del clúster, que se
   implementará en PC4.
5. Dar una primera respuesta a la pregunta de impacto social con el modelo entrenado sobre datos reales.

**Alcance de esta entrega (PC3):** objetivos 1, 2, 3 y 5 implementados y medidos; objetivo 4 diseñado y
verificado con Spin. El clúster por TCP, la API, las bases de datos y la interfaz web corresponden a PC4
y TB2.
