# 14. Recomendaciones

- **Medir el clúster en máquinas separadas.** Las mediciones de la sección 9 usan nodos en una sola PC y la
  red de *loopback*. Repetir el benchmark con nodos en equipos distintos mostraría el costo real de la red;
  como el tráfico por época no crece con los nodos, se espera que la eficiencia siga alta mientras el
  cómputo por nodo domine la época.
- **Dimensionar la caché por el patrón de consultas.** Las consultas reales se repiten por departamento y
  semana; conviene revisar en el registro de predicciones la proporción de aciertos de caché y ajustar el
  TTL de Redis a la frecuencia con que se publica un modelo nuevo.
- **Mantener el modelo formal al día.** Cada cambio del protocolo debe reflejarse en `spin/barrera.pml` y
  pasar por `spin/verificar.sh`, que ya corre en el CI; los mutantes muestran que la verificación detecta
  los errores de diseño que el código evita.
- **Completar la seguridad en TB2.** La API todavía no autentica: en TB2 se agregan JWT y CORS como
  *middleware*, sin cambiar las rutas, y la colección `usuarios` de MongoDB.
- **Mejorar el poder predictivo con más datos, no con más complejidad.** Integrar variables de clima o de
  vigilancia entomológica por distrito, si están disponibles como datos abiertos, aportaría más que un
  modelo más complejo. Los resultados deben presentarse siempre como priorización de zonas y grupos, no
  como diagnóstico individual.
- **Servir la SPA de TB2 en su propia imagen.** La imagen del backend lleva solo el coordinador y el nodo;
  la SPA (React) conviene servirla desde una imagen estática aparte, para que el backend siga pequeño y
  cada parte se despliegue por separado.
