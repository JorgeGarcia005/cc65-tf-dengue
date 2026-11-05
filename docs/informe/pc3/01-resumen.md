<!-- Generado por cmd/informe desde reports/*.json. No editar a mano. -->

# 1. Resumen

El presente trabajo aborda la anticipación de casos de **dengue con signos de alarma o grave** en el
Perú, los que requieren hospitalización, para orientar el refuerzo de camas, vigilancia y cerco
epidemiológico (ODS 3: Salud y bienestar). Se usan los **1 065 274 casos** notificados
entre 2000 y 2025 del conjunto de datos abierto de vigilancia epidemiológica del MINSA.

En esta entrega se construyeron en Go las dos primeras partes del sistema. La **Parte A** carga, limpia
y valida el archivo completo con un *worker pool* que reparte bloques de bytes; el resultado es idéntico
byte a byte para cualquier número de workers y conserva 1 065 260 filas válidas. La
**Parte B** entrena una **regresión logística** con descenso de gradiente cuyo cálculo se reparte en
bloques de tamaño fijo entre goroutines, con acumuladores privados y una reducción en orden de bloque,
de modo que el gradiente concurrente es **idéntico bit a bit** al secuencial.

En un AMD Ryzen 7 5800X (16 hilos), el cálculo del gradiente sobre el conjunto de entrenamiento real alcanza un speedup
de **10,13× a 10,96×** según el sistema operativo, y la limpieza de
3,25× a 3,54×. El modelo logra un AUC de 0,6221 en prueba
y señala el mayor riesgo en Madre de Dios (20,65 %), Loreto (19,78 %) y San Martín (19,17 %). Por último, se diseñó el clúster de nodos que se
implementará en PC4 (parameter server síncrono por TCP) y se verificó su protocolo con Spin.
