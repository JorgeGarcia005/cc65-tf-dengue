# 11. Recomendaciones

- **Implementar el clúster con las reglas que dejó la verificación formal.** En PC4, toda escritura a un nodo
  debe llevar plazo y la reasignación de shards debe hacerse en dos pasadas, tal como se verificó con Spin.
  Conviene mantener el modelo Promela al día con cualquier cambio del protocolo y volver a verificarlo.
- **Exigir la equivalencia bit a bit también en el clúster.** El test de PC4 debe levantar el coordinador y
  varios nodos con conexiones reales y comparar los pesos tras varias épocas con el entrenamiento
  secuencial, bajo `go test -race`.
- **Dimensionar los nodos por núcleos físicos.** Pasar de 8 a 16 workers en una CPU de 8 núcleos físicos
  aporta poco, y la parte secuencial de cada época (la reducción y la actualización) limita el speedup
  máximo (Amdahl, 1967); en Docker conviene limitar cada nodo a sus núcleos y escalar agregando nodos.
- **Medir siempre con el mismo protocolo.** Repetir las mediciones en orden barajado, usar la media recortada
  con intervalo de confianza y agrupar pasadas cortas, sobre todo en Windows. Registrar la máquina y el
  sistema operativo en cada reporte.
- **Mejorar el poder predictivo con más datos, no con más complejidad.** Integrar variables de clima o de
  vigilancia entomológica por distrito, si están disponibles como datos abiertos, aportaría más que un
  modelo más complejo. Los resultados deben presentarse siempre como priorización de zonas y grupos, no como
  diagnóstico individual.
- **Automatizar la descarga del dataset.** El portal de datos abiertos rechaza las descargas automáticas con
  `curl`; conviene documentar la descarga manual y verificar siempre el checksum antes de procesar.
