<!-- Generado por cmd/informe desde reports/*.json. No editar a mano. -->

# 10. Conclusiones

- **La paralelización del gradiente escala con los núcleos físicos.** En el AMD Ryzen 7 5800X (16 hilos), el speedup
  sobre los datos reales llega a 10,96× con 32 workers. La eficiencia baja de
  83,29 % con 8 workers a 63,13 % con 16, porque a partir de 8
  los workers comparten núcleos físicos (SMT). En el CI con 2 vCPU el mejor speedup fue
  1,64×: el mismo código aprovecha el hardware que encuentra.
- **El determinismo no se negocia por velocidad.** Con bloques de límites fijos y una reducción en orden de
  bloque, todas las configuraciones medidas dieron un gradiente idéntico bit a bit al
  secuencial. Esa misma propiedad permite repartir los bloques entre nodos en PC4 sin cambiar el
  resultado.
- **La concurrencia tiene un costo fijo.** Por debajo de unas 10 000 filas, repartir el trabajo
  cuesta más de lo que se gana: el diseño solo conviene a partir de varios bloques por worker. La limpieza
  escala menos (3,54×) porque está limitada por la memoria y el análisis de texto, no por el
  cálculo.
- **Medir bien es parte del resultado.** En Windows el reloj de Go avanza a saltos de fracciones de
  milisegundo y las pasadas cortas se medían como cero; agrupar pasadas y repetir la medición en Linux
  (Debian) dio cifras coherentes en ambos sistemas.
- **El modelo responde la pregunta de impacto, con límites.** Con un AUC de 0,6221, solo
  con edad, sexo, lugar y semana, el modelo ordena el riesgo mejor que el azar pero no lo determina. Aun
  así identifica dónde reforzar: por departamento, Madre de Dios (20,65 %), Loreto (19,78 %) y San Martín (19,17 %); por edad, el grupo
  niño (0–11 años), con 14,79 % de riesgo calibrado.
- **Verificar antes de implementar evitó dos errores.** El modelo Promela del protocolo del clúster encontró
  un bloqueo del coordinador al enviar a un nodo caído y un ciclo de *timeouts* al reasignar shards,
  ambos corregidos en el diseño antes de escribir el código de PC4.
