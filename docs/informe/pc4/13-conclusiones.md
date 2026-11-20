<!-- Generado por cmd/informe desde reports/*.json. No editar a mano. -->

# 13. Conclusiones

La pregunta que resuelve el problema era: *¿Puede un sistema distribuido en Go, que entrena una regresión logística sobre el millón de casos de dengue notificados por el MINSA, estimar en menos de 100 ms el riesgo de que un caso sea con signos de alarma o grave e indicar en qué departamentos, semanas epidemiológicas y grupos de edad conviene reforzar camas, vigilancia y cerco epidemiológico?* La respuesta es **sí, con los límites del
dato disponible**:

- **El sistema distribuido entrena el mismo modelo que el secuencial.** Con un parcial por bloque y la
  reducción en orden global de bloque, el clúster da pesos idénticos bit a bit al entrenamiento secuencial
  con cualquier número de nodos y también cuando un nodo cae a mitad del entrenamiento y su shard se
  reasigna. La distribución no cambia la respuesta, solo el tiempo.
- **La distribución escala porque la época está dominada por el cómputo.** Con 5
  nodos de un núcleo, la época baja de 92,0 ms a
  22,5 ms (speedup de 4,09×,
  eficiencia de 81,84 %). La red y la espera en la barrera ocupan
  3,42 % de la época con 5 nodos y los bytes por época
  casi no cambian con los nodos: la fracción serial de Karp-Flatt se mantiene baja y no crece.
- **La API cumple el requisito de latencia con margen.** Con 16 clientes concurrentes contra el
  clúster en Docker, el p95 fue de 3,51 ms sin caché y de 2,88 ms
  desde Redis, sin errores: el cálculo de una predicción es un producto punto de pocas decenas de
  variables, y la caché recorta el resto.
- **El modelo responde dónde, cuándo y en quién reforzar, con límites.** Con un AUC de 0,6221,
  solo con edad, sexo, lugar y semana, el modelo ordena el riesgo mejor que el azar pero no lo determina.
  Aun así identifica dónde reforzar: por departamento, Madre de Dios (20,65 %), Loreto (19,78 %) y San Martín (19,17 %); por edad, el grupo
  niño (0–11 años), con 14,79 % de riesgo calibrado; y por semana, las camas hacen más
  falta en las semanas 15–16 y 18–20, cuando se concentran los casos graves.
- **Verificar el protocolo antes y después de implementarlo evitó errores reales.** Spin recorrió
  17 949 492 estados del modelo sin errores, y los 4 mutantes (envío bloqueante, reasignación en
  una pasada, plazo global y primer nodo sano) dan contraejemplo: el modelo distingue cada decisión del
  diseño del error que corrige. La revisión entre agentes encontró además fallas de cierre y de medición
  que los tests no cubrían, y quedaron corregidas con un test cada una.
